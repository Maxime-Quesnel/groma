// Package report renders findings for the terminal.
package report

import (
	"cmp"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/Maxime-Quesnel/groma/internal/rule"
)

const indent = "          "

func Text(w io.Writer, findings []rule.Finding) {
	sorted := slices.Clone(findings)
	slices.SortStableFunc(sorted, func(a, b rule.Finding) int {
		if c := cmp.Compare(b.Rule.Severity, a.Rule.Severity); c != 0 {
			return c
		}
		return strings.Compare(a.Rule.ID, b.Rule.ID)
	})

	for _, f := range sorted {
		header := fmt.Sprintf("%-8s  %s", strings.ToUpper(f.Rule.Severity.String()), f.Rule.ID)
		if f.Subject != "" {
			header += "  " + f.Subject
		}
		fmt.Fprintln(w, header)
		fmt.Fprintf(w, "%s%s\n", indent, f.Rule.Title)
		for _, e := range f.Evidence {
			fmt.Fprintf(w, "%s- %s\n", indent, e)
		}
		fmt.Fprintf(w, "%sfix: %s\n\n", indent, f.Fix())
	}
	fmt.Fprintln(w, summary(sorted))
}

func summary(sorted []rule.Finding) string {
	if len(sorted) == 0 {
		return "No findings."
	}
	var counts []string
	for i := 0; i < len(sorted); {
		j := i
		for j < len(sorted) && sorted[j].Rule.Severity == sorted[i].Rule.Severity {
			j++
		}
		counts = append(counts, fmt.Sprintf("%d %s", j-i, sorted[i].Rule.Severity))
		i = j
	}
	noun := "findings"
	if len(sorted) == 1 {
		noun = "finding"
	}
	return fmt.Sprintf("%d %s: %s", len(sorted), noun, strings.Join(counts, ", "))
}
