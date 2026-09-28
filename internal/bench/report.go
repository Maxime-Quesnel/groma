package bench

import (
	"cmp"
	"fmt"
	"io"
	"maps"
	"slices"
	"strings"

	"github.com/Maxime-Quesnel/groma/internal/style"
)

// Text writes the report. judge names the judge model the runs used, or is
// empty when it was claude plugin eval's default.
func Text(w io.Writer, s Suite, r Result, judge string, runs []Run, scores []Score) {
	st := style.For(w)
	short := func(agent string) string { return cmp.Or(strings.TrimPrefix(agent, s.Plugin+":"), "no agent") }
	failed, undecided := 0, 0
	for _, run := range runs {
		if run.Failed {
			failed++
		}
		if run.TimedOut && len(run.Dispatched) == 0 {
			undecided++
		}
	}
	fmt.Fprintf(w, "%s  %s\n", st.Bold("groma bench · "+s.Plugin),
		st.Dim(fmt.Sprintf("%s · %s · about $%.2f at list price", count(len(runs), "run"), duration(r.DurationSeconds), r.CostUSD)))
	fmt.Fprintln(w, st.Dim(fmt.Sprintf("model %s · judge %s · Claude Code %s",
		cmp.Or(r.Suite.ModelOverride, "Claude Code's default"), cmp.Or(judge, "claude plugin eval's default"), r.ClaudeVersion)))
	if r.Partial {
		fmt.Fprintln(w, st.Yellow(fmt.Sprintf("⚠ Partial: the run stopped early (%s), so fewer runs than planned were scored.", r.PartialReason)))
	}

	fmt.Fprintf(w, "\n%s\n", st.Dim(fmt.Sprintf("%-30s %-20s %-20s %-11s %-11s %s", "AGENT", "PRECISION", "RECALL", "CHECKS", "JUDGE", "BASIS")))
	for _, sc := range scores {
		p, pOK := sc.Precision()
		pLo, pHi := sc.PrecisionInterval()
		rc, rOK := sc.Recall()
		rLo, rHi := sc.RecallInterval()
		fmt.Fprintf(w, "%s %s %s %s %s %s\n",
			style.Pad(short(sc.Agent), 30, st.Bold),
			rate(st, p, pLo, pHi, pOK).pad(20), rate(st, rc, rLo, rHi, rOK).pad(20),
			mean(st, sc.Checks).pad(11), mean(st, sc.Judge).pad(11),
			st.Dim(fmt.Sprintf("%s · %d expected · %d other", count(sc.Scenarios(), "scenario"), sc.Expected, sc.Other)))
		if len(sc.Instead) > 0 {
			fmt.Fprintf(w, "  %s\n", st.Yellow("› missed, dispatched instead: "+listCounts(sc.Instead, short)))
		}
		if len(sc.WrongCases) > 0 {
			fmt.Fprintf(w, "  %s\n", st.Red("› dispatched wrongly in: "+listCounts(sc.WrongCases, func(c string) string { return c })))
		}
	}

	fmt.Fprintf(w, "\n%s\n", st.Dim(fmt.Sprintf("%-52s %-28s %s", "CASE", "EXPECTS", "DISPATCHED PER RUN")))
	byCase := map[string][]Run{}
	for _, run := range runs {
		byCase[run.Case] = append(byCase[run.Case], run)
	}
	for _, c := range s.Cases {
		if len(byCase[c.Name]) == 0 {
			continue
		}
		expects := make([]string, len(c.Expect))
		for i, a := range c.Expect {
			expects[i] = short(a)
		}
		fmt.Fprintf(w, "%-52s %s %s\n", c.Name, style.Pad(cmp.Or(strings.Join(expects, ", "), "no agent"), 28, st.Dim), outcomes(st, c, byCase[c.Name], short))
	}

	fmt.Fprintln(w)
	for _, line := range []string{
		"Precision: of the runs that dispatched the agent, the share that should have.",
		"Recall: of the runs that should have dispatched it, the share that did.",
		"Intervals are 95%, widened when a scenario's rephrasings disagree, since they are not independent runs.",
		"Checks are exact graders; judge is a model's verdict. When they disagree, suspect the judge.",
	} {
		fmt.Fprintln(w, st.Dim(line))
	}
	if undecided > 0 {
		fmt.Fprintln(w, st.Dim(fmt.Sprintf("%s timed out before dispatching any agent: they decided nothing and are left out.", count(undecided, "run"))))
	}
	if failed > 0 {
		fmt.Fprintln(w, st.Dim(fmt.Sprintf("%s ended in an error such as a timeout; timed-out runs are left out of checks and judge.", count(failed, "run"))))
	}
}

// cell is a table value whose styled form carries escapes, so it pads on
// the plain form's width.
type cell struct{ plain, styled string }

func (c cell) pad(width int) string {
	return c.styled + strings.Repeat(" ", max(0, width-len([]rune(c.plain))))
}

// grade colors a share: green when it is high, red when it is low.
func grade(st style.Style, v float64) func(string) string {
	switch {
	case v >= 0.9:
		return st.Green
	case v >= 0.7:
		return st.Yellow
	}
	return st.Red
}

func rate(st style.Style, value, lo, hi float64, ok bool) cell {
	if !ok {
		return cell{"—", st.Dim("—")}
	}
	v, interval := fmt.Sprintf("%.2f", value), fmt.Sprintf("%.2f–%.2f", lo, hi)
	return cell{v + "  " + interval, grade(st, value)(v) + "  " + st.Dim(interval)}
}

func mean(st style.Style, m Mean) cell {
	v, n, ok := m.Value()
	if !ok {
		return cell{"—", st.Dim("—")}
	}
	value, basis := fmt.Sprintf("%.2f", v), fmt.Sprintf("n=%d", n)
	return cell{value + "  " + basis, grade(st, v)(value) + "  " + st.Dim(basis)}
}

// outcomes sums up what a case's runs dispatched, marking the right
// dispatches, the wrong ones and the runs cut short.
func outcomes(st style.Style, c Case, runs []Run, short func(string) string) string {
	type outcome struct {
		label       string
		right, late bool
	}
	seen := map[outcome]int{}
	var order []outcome
	for _, run := range runs {
		names := make([]string, len(run.Dispatched))
		for i, a := range run.Dispatched {
			names[i] = short(a)
		}
		right := len(c.Expect) == 0 && len(names) == 0 ||
			slices.ContainsFunc(run.Dispatched, func(a string) bool { return slices.Contains(c.Expect, a) })
		o := outcome{cmp.Or(strings.Join(names, "+"), "no agent"), right, run.TimedOut}
		if seen[o] == 0 {
			order = append(order, o)
		}
		seen[o]++
	}
	slices.SortStableFunc(order, func(a, b outcome) int { return cmp.Compare(seen[b], seen[a]) })
	parts := make([]string, len(order))
	for i, o := range order {
		text := fmt.Sprintf("%s ×%d", o.label, seen[o])
		switch {
		case o.late:
			parts[i] = st.Dim("⏱ " + text + " (timed out)")
		case o.right:
			parts[i] = st.Green("✔ " + text)
		default:
			parts[i] = st.Red("✖ " + text)
		}
	}
	return strings.Join(parts, "  ")
}

func listCounts(m map[string]int, name func(string) string) string {
	keys := slices.SortedFunc(maps.Keys(m), func(a, b string) int {
		return cmp.Or(cmp.Compare(m[b], m[a]), strings.Compare(a, b))
	})
	parts := make([]string, len(keys))
	for i, k := range keys {
		parts[i] = fmt.Sprintf("%s ×%d", name(k), m[k])
	}
	return strings.Join(parts, ", ")
}

func count(n int, noun string) string {
	if n == 1 {
		return "1 " + noun
	}
	return fmt.Sprintf("%d %ss", n, noun)
}

func duration(seconds float64) string {
	minutes := int(seconds+30) / 60
	if minutes < 60 {
		return fmt.Sprintf("%d min", max(minutes, 1))
	}
	return fmt.Sprintf("%d h %02d", minutes/60, minutes%60)
}
