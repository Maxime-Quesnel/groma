package report

import (
	"cmp"
	"fmt"
	"io"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode"

	"github.com/Maxime-Quesnel/groma/internal/rule"
	"github.com/Maxime-Quesnel/groma/internal/style"
)

const (
	indent      = "           "
	maxEvidence = 5
	wrapAt      = 100
)

func Text(w io.Writer, findings []rule.Finding) {
	st := style.For(w)
	sorted := slices.Clone(findings)
	slices.SortStableFunc(sorted, func(a, b rule.Finding) int {
		return cmp.Or(cmp.Compare(b.Rule.Severity, a.Rule.Severity), strings.Compare(a.Rule.ID, b.Rule.ID))
	})

	for _, f := range sorted {
		fmt.Fprintf(w, "%s %s\n", label(st, f.Rule.Severity), st.Bold(f.Rule.Title))
		where := st.Dim(f.Rule.ID)
		if f.Subject != "" {
			where = escape(f.Subject) + st.Dim(" · "+f.Rule.ID)
		}
		fmt.Fprintf(w, "%s%s\n", indent, where)
		for _, e := range f.Evidence[:min(len(f.Evidence), maxEvidence)] {
			fmt.Fprintf(w, "%s%s %s\n", indent, st.Dim("›"), escape(mask(e)))
		}
		if extra := len(f.Evidence) - maxEvidence; extra > 0 {
			fmt.Fprintf(w, "%s%s\n", indent, st.Dim(fmt.Sprintf("› and %d more", extra)))
		}
		for i, line := range wrap("Fix: "+f.Fix(), wrapAt-len(indent)) {
			if i == 0 {
				line = st.Cyan("Fix:") + strings.TrimPrefix(line, "Fix:")
			}
			fmt.Fprintf(w, "%s%s\n", indent, line)
		}
		fmt.Fprintln(w)
	}
	fmt.Fprintln(w, summary(st, findings))
}

// label is the severity column: the name padded to one width, then colored,
// so that titles line up.
func label(st style.Style, s rule.Severity) string {
	text := fmt.Sprintf(" %-8s ", strings.ToUpper(s.String()))
	switch s {
	case rule.Critical:
		return st.Badge(text)
	case rule.High:
		return st.BoldRed(text)
	case rule.Medium:
		return st.Yellow(text)
	}
	return st.Blue(text)
}

func summary(st style.Style, findings []rule.Finding) string {
	if len(findings) == 0 {
		return st.Green("✔ No findings.")
	}
	bySeverity := map[rule.Severity]int{}
	for _, f := range findings {
		bySeverity[f.Rule.Severity]++
	}
	parts := []string{st.BoldRed(fmt.Sprintf("✖ %d %s", len(findings), plural(len(findings), "finding")))}
	for s := rule.Critical; s >= rule.Low; s-- {
		if n := bySeverity[s]; n > 0 {
			parts = append(parts, fmt.Sprintf("%d %s", n, s))
		}
	}
	return strings.Join(parts, st.Dim(" · "))
}

func plural(n int, noun string) string {
	if n == 1 {
		return noun
	}
	return noun + "s"
}

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

// Evidence quotes commands and lines from the files being audited, which can
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

// Subjects and evidence come from the files being audited, so anything a
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
