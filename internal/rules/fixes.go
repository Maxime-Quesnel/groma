package rules

import (
	"bytes"
	"encoding/json"
	"regexp"
	"slices"
	"strings"

	"github.com/Maxime-Quesnel/groma/internal/component"
	"github.com/Maxime-Quesnel/groma/internal/config"
	"github.com/Maxime-Quesnel/groma/internal/fix"
)

// Fixes returns the edits that correct what the rules find in the tree's
// targets, safe ones only unless unsafe is set, for the rules the
// configuration and the components leave on.
func Fixes(t *component.Tree, cfg config.Config, unsafe bool) []fix.Edit {
	var edits []fix.Edit
	for _, c := range t.Targets {
		for _, r := range All {
			if r.Fix == nil || !slices.Contains(r.Kinds, c.Kind) || !applies(r, c, t, cfg) || len(r.Check(c, t)) == 0 {
				continue
			}
			for _, e := range r.Fix(c, t, unsafe) {
				e.Rule = r.ID
				edits = append(edits, e)
			}
		}
	}
	return edits
}

// lineRange returns the byte offsets of lines first to last of content,
// counted from 1, without the last line's newline.
func lineRange(content []byte, first, last int) (start, end int) {
	line := 1
	for i := 0; i < len(content) && line < first; i++ {
		if content[i] == '\n' {
			line++
			start = i + 1
		}
	}
	end = start
	for line <= last && end < len(content) {
		next := bytes.IndexByte(content[end:], '\n')
		if next < 0 {
			return start, len(content)
		}
		if line == last {
			return start, end + next
		}
		end += next + 1
		line++
	}
	return start, end
}

// lineAt returns line n of content, counted from 1.
func lineAt(content []byte, n int) string {
	start, end := lineRange(content, n, n)
	return string(content[start:end])
}

// replaceInLine returns an edit that replaces the first old in line n of
// the component's file with new.
func replaceInLine(c *component.Component, n int, old, new string) (fix.Edit, bool) {
	start, end := lineRange(c.Content, n, n)
	i := strings.Index(string(c.Content[start:end]), old)
	if i < 0 || old == new {
		return fix.Edit{}, false
	}
	return fix.Edit{Path: c.Path, Start: start + i, End: start + i + len(old), New: new}, true
}

// jsonString writes s as a JSON string, the way hooks files write it.
func jsonString(s string) string {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	enc.Encode(s)
	return strings.TrimSpace(b.String())
}

// jsonValueEdits returns edits that replace each "key": <old> in the
// component's JSON file with new, where old and new are JSON values as
// written. A value written with other escapes than jsonString's isn't found,
// and so isn't changed.
func jsonValueEdits(c *component.Component, key, old, new string) []fix.Edit {
	re := regexp.MustCompile(`"` + regexp.QuoteMeta(key) + `"\s*:\s*(` + regexp.QuoteMeta(old) + `)\s*[,}\]]`)
	var edits []fix.Edit
	for _, m := range re.FindAllSubmatchIndex(c.Content, -1) {
		edits = append(edits, fix.Edit{Path: c.Path, Start: m[2], End: m[3], New: new})
	}
	return edits
}

// listFix edits the entries of a frontmatter list, such as tools. change
// returns an entry's replacement and whether it changes, with "" to remove
// it; it sees every entry, to tell whether a replacement is already listed.
// No edit comes back when nothing changes or no entry would remain, since an
// empty tools field grants every tool, nor when the field is written in a
// form groma doesn't rewrite: over several lines of a flow list, quoted as a
// whole, or with a comment.
func listFix(c *component.Component, key string, change func(entry string, all []string) (string, bool)) []fix.Edit {
	f, ok := c.Header.Field(key)
	if !ok || !headerReadable(c) {
		return nil
	}
	all := f.Items()
	var kept int
	var edits []fix.Edit
	apply := func(entry string) (string, bool) {
		replacement, changed := change(entry, all)
		if changed && replacement != "" && replacement != entry && slices.Contains(all, replacement) {
			replacement = ""
		}
		if !changed || replacement != "" {
			kept++
		}
		return replacement, changed
	}

	if f.Line == f.EndLine {
		line := lineAt(c.Content, f.Line)
		_, value, _ := strings.Cut(line, ":")
		raw := strings.TrimSpace(value)
		if raw == "" || strings.Contains(raw, " #") || raw[0] == '"' || raw[0] == '\'' {
			return nil
		}
		flow := strings.HasPrefix(raw, "[") && strings.HasSuffix(raw, "]")
		inner, seps, join := raw, ", \t", " "
		if strings.Contains(raw, ",") {
			join = ", "
		}
		if flow {
			inner, seps, join = raw[1:len(raw)-1], ",", ", "
		}
		var out []string
		changedAny := false
		for _, token := range splitRaw(inner, seps) {
			entry := strings.Trim(token, `"'`)
			replacement, changed := apply(entry)
			switch {
			case !changed:
				out = append(out, token)
			case replacement != "":
				out = append(out, strings.Replace(token, entry, replacement, 1))
				changedAny = true
			default:
				changedAny = true
			}
		}
		if !changedAny || kept == 0 {
			return nil
		}
		value = strings.Join(out, join)
		if flow {
			value = "[" + value + "]"
		}
		start, end := lineRange(c.Content, f.Line, f.Line)
		prefix := line[:strings.Index(line, raw)]
		return []fix.Edit{{Path: c.Path, Start: start, End: end, New: prefix + value}}
	}

	if !f.IsList {
		return nil
	}
	for n := f.Line + 1; n <= f.EndLine; n++ {
		line := lineAt(c.Content, n)
		item, isItem := strings.CutPrefix(strings.TrimSpace(line), "- ")
		if !isItem {
			if strings.TrimSpace(line) != "" {
				return nil
			}
			continue
		}
		entry := strings.Trim(strings.TrimSpace(item), `"'`)
		replacement, changed := apply(entry)
		if !changed {
			continue
		}
		start, end := lineRange(c.Content, n, n)
		if replacement == "" {
			edits = append(edits, fix.Edit{Path: c.Path, Start: start, End: min(end+1, len(c.Content))})
		} else if e, ok := replaceInLine(c, n, entry, replacement); ok {
			edits = append(edits, e)
		}
	}
	if kept == 0 {
		return nil
	}
	return edits
}

// splitRaw splits s at any of seps outside parentheses and quotes, keeping
// each entry as written.
func splitRaw(s, seps string) []string {
	var parts []string
	depth, start := 0, 0
	var quote rune
	for i, r := range s + string(seps[0]) {
		switch {
		case quote != 0:
			if r == quote {
				quote = 0
			}
		case r == '"' || r == '\'':
			quote = r
		case r == '(':
			depth++
		case r == ')':
			depth--
		case depth == 0 && strings.ContainsRune(seps, r):
			if part := strings.TrimSpace(s[start:min(i, len(s))]); part != "" {
				parts = append(parts, part)
			}
			start = i + 1
		}
	}
	return parts
}
