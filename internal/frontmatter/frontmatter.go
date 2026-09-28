// Package frontmatter parses the YAML header of Markdown files the way
// skills, subagents and commands use it: top-level keys holding a scalar, a
// list or a nested map. It keeps line numbers, and records what a YAML parser
// rejects, since Claude Code then drops every field of the header.
package frontmatter

import (
	"fmt"
	"regexp"
	"strings"
)

type Header struct {
	// Found reports that the file opens with a --- line.
	Found bool
	// Misplaced reports a --- line that would open a header if it came
	// first, after blank lines: the header is then read as body text.
	Misplaced bool
	Fields    []Field
	// Problems are what a YAML parser rejects in the header.
	Problems []Problem
	Body     string
	// BodyLine is the line of the file where Body starts.
	BodyLine int
}

type Field struct {
	Key string
	// Line is where the field starts in the file, and EndLine the last
	// line its value runs to.
	Line, EndLine int
	// Value is a scalar with its quotes removed and its lines joined.
	Value string
	// List holds the items of a flow ([a, b]) or block (- a) list.
	List   []string
	IsList bool
	// IsMap marks a nested mapping, which is kept unparsed.
	IsMap bool
}

type Problem struct {
	Line int
	Text string
}

func (h Header) Field(key string) (Field, bool) {
	for _, f := range h.Fields {
		if f.Key == key {
			return f, true
		}
	}
	return Field{}, false
}

// Value returns a scalar field's value, or "" when the field is absent.
func (h Header) Value(key string) string {
	f, _ := h.Field(key)
	return f.Value
}

func (h Header) Has(key string) bool {
	_, ok := h.Field(key)
	return ok
}

// Items returns a field's entries, whether it is a list or a string that
// separates them with commas or spaces, as tools and allowed-tools may:
// "Read, Bash(git add *)" gives two entries.
func (f Field) Items() []string {
	if f.IsList {
		return f.List
	}
	return splitOutsideParens(f.Value, ", \t")
}

func Parse(content []byte) Header {
	text := strings.ReplaceAll(string(content), "\r\n", "\n")
	text = strings.TrimPrefix(text, string(rune(0xFEFF)))
	lines := strings.Split(text, "\n")
	h := Header{Body: text, BodyLine: 1}
	if !delimiter(lines[0]) {
		for _, line := range lines {
			if strings.TrimSpace(line) != "" {
				h.Misplaced = delimiter(line)
				break
			}
		}
		return h
	}
	h.Found = true
	for i := 1; i < len(lines); i++ {
		if delimiter(lines[i]) {
			h.parse(lines[1:i], 2)
			h.Body = strings.Join(lines[i+1:], "\n")
			h.BodyLine = i + 2
			return h
		}
	}
	h.Problems = append(h.Problems, Problem{1, "the header opened on line 1 is never closed by a --- line"})
	return h
}

func delimiter(line string) bool {
	return strings.TrimRight(line, " \t") == "---"
}

var (
	keyLine        = regexp.MustCompile(`^([A-Za-z0-9_][A-Za-z0-9_.-]*)[ \t]*:(?:[ \t]+(.*?))?[ \t]*$`)
	blockIndicator = regexp.MustCompile(`^[|>][0-9+-]*(?:[ \t]+#.*)?$`)
	nestedKey      = regexp.MustCompile(`^[A-Za-z0-9_"'][^:]*:(?:[ \t]|$)`)
)

// ParseDocument reads a whole file as YAML, as a configuration file is, with
// the same subset and checks as a Markdown header.
func ParseDocument(content []byte) Header {
	text := strings.TrimPrefix(strings.ReplaceAll(string(content), "\r\n", "\n"), string(rune(0xFEFF)))
	var h Header
	h.Found = true
	h.parse(strings.Split(text, "\n"), 1)
	return h
}

// parse reads the header's lines, the first of which is line first of the
// file.
func (h *Header) parse(lines []string, first int) {
	lineOf := func(i int) int { return i + first }
	for i := 0; i < len(lines); {
		line := lines[i]
		if trimmed := strings.TrimSpace(line); trimmed == "" || strings.HasPrefix(trimmed, "#") {
			i++
			continue
		}
		if line[0] == ' ' || line[0] == '\t' {
			h.problem(lineOf(i), "this indented line belongs to no field")
			i++
			continue
		}
		m := keyLine.FindStringSubmatch(line)
		if m == nil {
			h.problem(lineOf(i), fmt.Sprintf("%q is not a key: value line", clip(line)))
			i++
			continue
		}
		f := Field{Key: m[1], Line: lineOf(i)}
		if prev, dup := h.Field(f.Key); dup {
			h.problem(f.Line, fmt.Sprintf("%s is set twice, on lines %d and %d", f.Key, prev.Line, f.Line))
		}
		value := m[2]
		// A value continues on the lines indented under its key; a block list
		// may also start its items at column 0.
		end := i + 1
		for end < len(lines) {
			next := lines[end]
			if strings.TrimSpace(next) == "" || next[0] == ' ' || next[0] == '\t' ||
				value == "" && (strings.HasPrefix(next, "- ") || next == "-") {
				end++
				continue
			}
			break
		}
		// A flow list or map runs to its closing bracket, which may sit at
		// column 0.
		for end < len(lines) && openFlow(value, lines[i+1:end]) {
			end++
		}
		nested := lines[i+1 : end]
		for len(nested) > 0 && strings.TrimSpace(nested[len(nested)-1]) == "" {
			nested = nested[:len(nested)-1]
		}
		for j, next := range nested {
			if strings.HasPrefix(next, "\t") {
				h.problem(lineOf(i+1+j), "a tab indents this line; YAML only allows spaces")
			}
		}
		f.EndLine = f.Line + len(nested)
		h.value(&f, value, nested)
		h.Fields = append(h.Fields, f)
		i = end
	}
}

func (h *Header) value(f *Field, value string, nested []string) {
	if value == "" {
		// A flow list or map may open on the line after its key.
		for i, line := range nested {
			if trimmed := strings.TrimSpace(line); trimmed != "" {
				if trimmed[0] == '[' || trimmed[0] == '{' {
					value, nested = trimmed, nested[i+1:]
				}
				break
			}
		}
	}
	switch {
	case value == "":
		first := ""
		for _, line := range nested {
			if first = strings.TrimSpace(line); first != "" {
				break
			}
		}
		switch {
		case first == "-" || strings.HasPrefix(first, "- "):
			f.IsList = true
			for _, line := range nested {
				if item, ok := strings.CutPrefix(strings.TrimSpace(line), "-"); ok && (item == "" || item[0] == ' ') {
					f.List = append(f.List, unquote(stripComment(item)))
				}
			}
		case nestedKey.MatchString(first):
			f.IsMap = true
		default:
			f.Value = fold(nested)
		}
	case blockIndicator.MatchString(value):
		f.Value = block(nested, value[0] == '|')
	case value[0] == '"' || value[0] == '\'':
		text := strings.Join(append([]string{value}, trimAll(nested)...), "\n")
		s, rest, ok := quoted(text)
		if !ok {
			h.problem(f.Line, fmt.Sprintf("the quote opening %s's value is never closed", f.Key))
			return
		}
		if rest = strings.TrimSpace(rest); rest != "" && !strings.HasPrefix(rest, "#") {
			h.problem(f.Line, fmt.Sprintf("%s's value goes on after its closing quote", f.Key))
		}
		f.Value = s
	case value[0] == '[':
		text := strings.Join(append([]string{value}, trimAll(nested)...), " ")
		inner, rest, ok := flow(text, '[', ']')
		if !ok {
			h.problem(f.Line, fmt.Sprintf("the [ opening %s's list is never closed", f.Key))
			return
		}
		// Claude Code reads a hint such as [file] [pages] as plain text,
		// although strict YAML rejects it.
		if rest = strings.TrimSpace(stripComment(" " + rest)); rest != "" {
			f.Value = text
			return
		}
		f.IsList = true
		for _, item := range splitOutsideParens(inner, ",") {
			f.List = append(f.List, unquote(item))
		}
	case value[0] == '{':
		if _, _, ok := flow(strings.Join(append([]string{value}, trimAll(nested)...), " "), '{', '}'); !ok {
			h.problem(f.Line, fmt.Sprintf("the { opening %s's map is never closed", f.Key))
		}
		f.IsMap = true
	case strings.ContainsRune("@`*!", rune(value[0])):
		h.problem(f.Line, fmt.Sprintf("%s's value starts with %c, which YAML reserves: quote the value", f.Key, value[0]))
	default:
		f.Value = fold(append([]string{stripComment(value)}, nested...))
	}
}

// openFlow reports whether a flow list or map starting in value or on the
// next line is still open after lines.
func openFlow(value string, lines []string) bool {
	text := strings.TrimSpace(value + " " + strings.Join(trimAll(lines), " "))
	if text == "" || text[0] != '[' && text[0] != '{' {
		return false
	}
	closing := byte(']')
	if text[0] == '{' {
		closing = '}'
	}
	_, _, closed := flow(text, text[0], closing)
	return !closed
}

func (h *Header) problem(line int, text string) {
	h.Problems = append(h.Problems, Problem{line, text})
}

// fold joins the lines of a plain scalar: lines with a space, blank lines
// with a newline.
func fold(lines []string) string {
	var b strings.Builder
	blank := false
	for _, line := range lines {
		line = strings.TrimSpace(line)
		switch {
		case line == "":
			blank = true
			continue
		case b.Len() == 0:
		case blank:
			b.WriteString("\n")
		default:
			b.WriteString(" ")
		}
		b.WriteString(line)
		blank = false
	}
	return b.String()
}

// block reads a literal (|) or folded (>) block scalar, indented under its key.
func block(lines []string, literal bool) string {
	indent := -1
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			indent = len(line) - len(strings.TrimLeft(line, " "))
			break
		}
	}
	if indent < 0 {
		return ""
	}
	dedented := make([]string, len(lines))
	for i, line := range lines {
		if len(line) >= indent {
			dedented[i] = line[indent:]
		} else {
			dedented[i] = strings.TrimSpace(line)
		}
	}
	if literal {
		return strings.Join(dedented, "\n")
	}
	return fold(dedented)
}

// quoted reads a YAML quoted scalar at the start of text and returns it
// unescaped, with what follows its closing quote.
func quoted(text string) (value, rest string, ok bool) {
	q := text[0]
	var b strings.Builder
	for i := 1; i < len(text); i++ {
		c := text[i]
		switch {
		case q == '\'' && c == '\'' && i+1 < len(text) && text[i+1] == '\'':
			b.WriteByte('\'')
			i++
		case q == '"' && c == '\\' && i+1 < len(text):
			i++
			switch text[i] {
			case 'n':
				b.WriteByte('\n')
			case 't':
				b.WriteByte('\t')
			default:
				b.WriteByte(text[i])
			}
		case c == q:
			return strings.Join(strings.Fields(b.String()), " "), text[i+1:], true
		default:
			b.WriteByte(c)
		}
	}
	return "", "", false
}

// flow returns what lies between an opening bracket at the start of text and
// its matching closing bracket, and what follows it.
func flow(text string, open, close byte) (inner, rest string, ok bool) {
	depth := 0
	var quote byte
	for i := 0; i < len(text); i++ {
		c := text[i]
		switch {
		case quote != 0:
			if c == quote {
				quote = 0
			}
		case c == '"' || c == '\'':
			quote = c
		case c == open:
			depth++
		case c == close:
			if depth--; depth == 0 {
				return text[1:i], text[i+1:], true
			}
		}
	}
	return "", "", false
}

// splitOutsideParens splits s at any of seps outside parentheses and quotes,
// so "Bash(git add *), Read" gives two entries.
func splitOutsideParens(s, seps string) []string {
	var items []string
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
			if item := unquote(s[start:min(i, len(s))]); item != "" {
				items = append(items, item)
			}
			start = i + 1
		}
	}
	return items
}

func stripComment(s string) string {
	if i := strings.Index(s, " #"); i >= 0 {
		return s[:i]
	}
	return s
}

func trimAll(lines []string) []string {
	out := make([]string, len(lines))
	for i, line := range lines {
		out[i] = strings.TrimSpace(line)
	}
	return out
}

func unquote(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 && (s[0] == '"' || s[0] == '\'') && s[len(s)-1] == s[0] {
		return s[1 : len(s)-1]
	}
	return s
}

func clip(s string) string {
	if r := []rune(s); len(r) > 60 {
		return string(r[:60]) + "…"
	}
	return s
}
