package report

import (
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/Maxime-Quesnel/groma/internal/component"
	"github.com/Maxime-Quesnel/groma/internal/rule"
	"github.com/Maxime-Quesnel/groma/internal/style"
)

const (
	indent      = "      "
	maxEvidence = 5
	wrapAt      = 100
)

// Text writes a check: one line per component, with its warnings under it,
// then every red flag, then a summary line that counts the findings the
// configuration silenced.
func Text(w io.Writer, target string, components []*component.Component, findings []rule.Finding, silenced int) {
	st := style.For(w)
	type key struct{ path, kind string }
	byComponent := map[key][]rule.Finding{}
	for _, f := range findings {
		byComponent[key{f.Path, f.Kind}] = append(byComponent[key{f.Path, f.Kind}], f)
	}
	fmt.Fprintf(w, "%s\n\n", st.Dim(fmt.Sprintf("Checking %s · %s", target, inventory(components))))

	width := 0
	for _, c := range components {
		width = max(width, utf8.RuneCountInString(escape(c.Path)))
	}
	var redFlags []rule.Finding
	for _, c := range components {
		red, warnings := split(byComponent[key{c.Path, c.Kind.String()}])
		redFlags = append(redFlags, red...)
		mark, counts := st.Green("✔"), ""
		switch {
		case len(red) > 0:
			mark = st.BoldRed("✖")
			counts = st.Red(count(len(red), "red flag") + " ↓")
			if len(warnings) > 0 {
				counts += st.Dim(" · ") + st.Yellow(count(len(warnings), "warning"))
			}
		case len(warnings) > 0:
			mark = st.Yellow("▲")
			counts = st.Yellow(count(len(warnings), "warning"))
		}
		line := fmt.Sprintf("  %s %s %s", mark, style.Pad(c.Kind.String(), 7, st.Dim), style.Pad(escape(c.Path), width, noStyle))
		fmt.Fprintln(w, strings.TrimRight(line+"  "+counts, " "))
		for _, f := range warnings {
			block(w, st, f)
		}
	}

	if len(redFlags) > 0 {
		fmt.Fprintf(w, "\n%s\n", st.BoldRed("Red flags"))
		shown := ""
		for _, f := range redFlags {
			if f.Path != shown {
				fmt.Fprintf(w, "  %s %s\n", st.BoldRed("✖"), escape(f.Path))
				shown = f.Path
			}
			block(w, st, f)
		}
	}
	line := summary(st, len(components), findings)
	if silenced > 0 {
		line += st.Dim(fmt.Sprintf(" · %d silenced by .groma.yml or groma:disable", silenced))
	}
	fmt.Fprintf(w, "\n%s\n", line)
}

func block(w io.Writer, st style.Style, f rule.Finding) {
	fmt.Fprintf(w, "%s%s %s\n", indent, st.Bold(f.Rule.Title), st.Dim("· "+f.Rule.ID))
	for _, e := range f.Evidence[:min(len(f.Evidence), maxEvidence)] {
		fmt.Fprintf(w, "%s%s %s\n", indent, st.Dim("›"), escape(mask(e)))
	}
	if extra := len(f.Evidence) - maxEvidence; extra > 0 {
		fmt.Fprintf(w, "%s%s\n", indent, st.Dim(fmt.Sprintf("› and %d more", extra)))
	}
	for i, line := range wrap("Fix: "+f.Rule.Remediation, wrapAt-len(indent)) {
		if i == 0 {
			line = st.Cyan("Fix:") + strings.TrimPrefix(line, "Fix:")
		}
		fmt.Fprintf(w, "%s%s\n", indent, line)
	}
}

func split(findings []rule.Finding) (red, warnings []rule.Finding) {
	for _, f := range findings {
		if f.Rule.Level == rule.RedFlag {
			red = append(red, f)
		} else {
			warnings = append(warnings, f)
		}
	}
	return red, warnings
}

// inventory counts the components by kind: 3 skills, 1 agent, 1 hooks file.
func inventory(components []*component.Component) string {
	counts := map[component.Kind]int{}
	for _, c := range components {
		counts[c.Kind]++
	}
	var parts []string
	for _, k := range []component.Kind{component.Plugin, component.Skill, component.Agent, component.Command, component.Hooks} {
		if n := counts[k]; n > 0 {
			noun := k.String()
			if k == component.Hooks {
				noun = "hooks file"
			}
			parts = append(parts, count(n, noun))
		}
	}
	return strings.Join(parts, ", ")
}

func summary(st style.Style, components int, findings []rule.Finding) string {
	red, warnings := split(findings)
	checked := count(components, "component")
	switch {
	case len(red) > 0:
		line := st.BoldRed("✖ " + count(len(red), "red flag"))
		if len(warnings) > 0 {
			line += st.Dim(" · ") + st.Yellow(count(len(warnings), "warning"))
		}
		return line + st.Dim(" in "+checked)
	case len(warnings) > 0:
		return st.Yellow("▲ "+count(len(warnings), "warning")) + st.Dim(" in "+checked+", no red flags")
	}
	return st.Green("✔ No red flags or warnings in " + checked + ".")
}

func count(n int, noun string) string {
	if n == 1 {
		return fmt.Sprintf("1 %s", noun)
	}
	return fmt.Sprintf("%d %ss", n, noun)
}

func noStyle(s string) string { return s }

// wrap breaks text into lines of at most width runes, at spaces.
func wrap(text string, width int) []string {
	var lines []string
	line := ""
	for _, word := range strings.Fields(text) {
		if line != "" && len([]rune(line))+1+len([]rune(word)) > width {
			lines = append(lines, line)
			line = ""
		}
		if line != "" {
			line += " "
		}
		line += word
	}
	return append(lines, line)
}

// Evidence quotes commands and lines from the files being checked, which can
// carry credentials. These are the common shapes: an Authorization header,
// user:password in a URL, a token in a query string, well-known key prefixes.
var secretShapes = []*regexp.Regexp{
	regexp.MustCompile(`(?i)(authorization:\s*(?:(?:bearer|token|basic)\s+)?)[^\s"']+`),
	regexp.MustCompile(`(://[^/\s:@]+:)[^/\s@]+(@)`),
	regexp.MustCompile(`(?i)([?&](?:access_token|api_?key|key|password|secret|sig|token)=)[^&\s"']+`),
	regexp.MustCompile(`()\b(?:sk-[A-Za-z0-9_-]{16,}|gh[pousr]_[A-Za-z0-9]{20,}|github_pat_[A-Za-z0-9_]{20,}|xox[abprs]-[A-Za-z0-9-]{10,}|AKIA[0-9A-Z]{16})`),
}

func mask(s string) string {
	for _, re := range secretShapes {
		s = re.ReplaceAllString(s, "${1}****${2}")
	}
	return s
}

// Paths and evidence come from the files being checked, so anything a
// terminal wouldn't print as-is is spelled out, and a crafted file name or
// content can't send control sequences to the terminal.
func escape(s string) string {
	if !strings.ContainsFunc(s, notGraphic) {
		return s
	}
	var b strings.Builder
	for _, r := range s {
		if notGraphic(r) {
			q := strconv.QuoteRuneToGraphic(r)
			b.WriteString(q[1 : len(q)-1])
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func notGraphic(r rune) bool {
	return !unicode.IsGraphic(r)
}

// Diff writes a unified diff in color, with secrets masked and untrusted
// text escaped like the rest of the report. Only the display is masked: the
// file keeps its real content.
func Diff(w io.Writer, lines []string) {
	st := style.For(w)
	for _, line := range lines {
		shown := escape(mask(line))
		switch {
		case strings.HasPrefix(line, "---") || strings.HasPrefix(line, "+++"):
			shown = st.Bold(shown)
		case strings.HasPrefix(line, "@@"):
			shown = st.Cyan(shown)
		case strings.HasPrefix(line, "-"):
			shown = st.Red(shown)
		case strings.HasPrefix(line, "+"):
			shown = st.Green(shown)
		}
		fmt.Fprintln(w, shown)
	}
}
