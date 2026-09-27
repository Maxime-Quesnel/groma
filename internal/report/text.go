package report

import (
	"cmp"
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"
	"unicode"

	"github.com/Maxime-Quesnel/groma/internal/rule"
)

const (
	indent      = "          "
	maxEvidence = 5
)

func Text(w io.Writer, findings []rule.Finding) {
	sorted := slices.Clone(findings)
	slices.SortStableFunc(sorted, func(a, b rule.Finding) int {
		return cmp.Or(cmp.Compare(b.Rule.Severity, a.Rule.Severity), strings.Compare(a.Rule.ID, b.Rule.ID))
	})

	for _, f := range sorted {
		fmt.Fprintf(w, "%-8s  %s", strings.ToUpper(f.Rule.Severity.String()), f.Rule.ID)
		if f.Subject != "" {
			fmt.Fprintf(w, "  %s", escape(f.Subject))
		}
		fmt.Fprintf(w, "\n%s%s\n", indent, f.Rule.Title)
		for _, e := range f.Evidence[:min(len(f.Evidence), maxEvidence)] {
			fmt.Fprintf(w, "%s- %s\n", indent, escape(e))
		}
		if extra := len(f.Evidence) - maxEvidence; extra > 0 {
			fmt.Fprintf(w, "%s- and %d more\n", indent, extra)
		}
		fmt.Fprintf(w, "%sfix: %s\n\n", indent, f.Fix())
	}
	fmt.Fprintln(w, summary(findings))
}

func summary(findings []rule.Finding) string {
	if len(findings) == 0 {
		return "No findings."
	}
	bySeverity := map[rule.Severity]int{}
	for _, f := range findings {
		bySeverity[f.Rule.Severity]++
	}
	var counts []string
	for s := rule.Critical; s >= rule.Low; s-- {
		if n := bySeverity[s]; n > 0 {
			counts = append(counts, fmt.Sprintf("%d %s", n, s))
		}
	}
	noun := "findings"
	if len(findings) == 1 {
		noun = "finding"
	}
	return fmt.Sprintf("%d %s: %s", len(findings), noun, strings.Join(counts, ", "))
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
