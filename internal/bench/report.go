package bench

import (
	"cmp"
	"fmt"
	"io"
	"maps"
	"slices"
	"strings"
)

func Text(w io.Writer, s Suite, r Result, runs []Run, scores []Score) {
	short := func(agent string) string { return cmp.Or(strings.TrimPrefix(agent, s.Plugin+":"), "no agent") }
	failed := 0
	for _, run := range runs {
		if run.Failed {
			failed++
		}
	}
	fmt.Fprintf(w, "groma bench · %s · %s · Claude Code %s · %d runs · %s · about $%.2f at list price\n",
		s.Plugin, cmp.Or(r.Suite.ModelOverride, "Claude Code's default model"), r.ClaudeVersion, len(runs), duration(r.DurationSeconds), r.CostUSD)
	if r.Partial {
		fmt.Fprintf(w, "PARTIAL: the run stopped early (%s); fewer runs than planned were scored.\n", r.PartialReason)
	}

	fmt.Fprintf(w, "\n%-30s %-19s %-19s %-12s %s\n", "AGENT", "PRECISION", "RECALL", "WORK", "BASIS")
	for _, sc := range scores {
		p, pOK := sc.Precision()
		pLo, pHi := sc.PrecisionInterval()
		rc, rOK := sc.Recall()
		rLo, rHi := sc.RecallInterval()
		work := "—"
		if v, ok := sc.WorkScore(); ok {
			work = fmt.Sprintf("%.2f (n=%d)", v, sc.WorkRuns)
		}
		fmt.Fprintf(w, "%-30s %-19s %-19s %-12s %d scenarios, %d runs expected, %d other\n",
			short(sc.Agent), rate(p, pLo, pHi, pOK), rate(rc, rLo, rHi, rOK), work, sc.Scenarios(), sc.Expected, sc.Other)
		if len(sc.Instead) > 0 {
			fmt.Fprintf(w, "  missed, dispatched instead: %s\n", counts(sc.Instead, short))
		}
		if len(sc.WrongCases) > 0 {
			fmt.Fprintf(w, "  dispatched wrongly in: %s\n", counts(sc.WrongCases, func(c string) string { return c }))
		}
	}

	fmt.Fprintf(w, "\n%-52s %-28s %s\n", "CASE", "EXPECTS", "DISPATCHED PER RUN")
	byCase := map[string][]Run{}
	for _, run := range runs {
		byCase[run.Case] = append(byCase[run.Case], run)
	}
	for _, c := range s.Cases {
		caseRuns := byCase[c.Name]
		if len(caseRuns) == 0 {
			continue
		}
		got := map[string]int{}
		for _, run := range caseRuns {
			names := make([]string, len(run.Dispatched))
			for i, a := range run.Dispatched {
				names[i] = short(a)
			}
			got[cmp.Or(strings.Join(names, "+"), "no agent")]++
		}
		expects := make([]string, len(c.Expect))
		for i, a := range c.Expect {
			expects[i] = short(a)
		}
		fmt.Fprintf(w, "%-52s %-28s %s\n", c.Name, cmp.Or(strings.Join(expects, ", "), "no agent named"), counts(got, func(k string) string { return k }))
	}

	fmt.Fprintln(w, "\nPrecision: of the runs that dispatched the agent, the share that should have. Recall: of the runs")
	fmt.Fprintln(w, "that should have dispatched it, the share that did. Work: the share of the cases' work graders passed")
	fmt.Fprintln(w, "when the agent was rightly dispatched, on cases that grade the work. In brackets, a 95% interval: the")
	fmt.Fprintln(w, "wider of a Wilson interval over runs and a bootstrap over scenarios, since runs of one scenario are")
	fmt.Fprintln(w, "not independent.")
	if failed > 0 {
		fmt.Fprintf(w, "%d runs ended in an error, such as a timeout; they are scored on what they did before.\n", failed)
	}
}

func rate(value, lo, hi float64, ok bool) string {
	if !ok {
		return "—"
	}
	return fmt.Sprintf("%.2f (%.2f–%.2f)", value, lo, hi)
}

func counts(m map[string]int, name func(string) string) string {
	keys := slices.SortedFunc(maps.Keys(m), func(a, b string) int {
		return cmp.Or(cmp.Compare(m[b], m[a]), strings.Compare(a, b))
	})
	parts := make([]string, len(keys))
	for i, k := range keys {
		parts[i] = fmt.Sprintf("%s ×%d", name(k), m[k])
	}
	return strings.Join(parts, ", ")
}

func duration(seconds float64) string {
	return fmt.Sprintf("%dm%02ds", int(seconds)/60, int(seconds)%60)
}
