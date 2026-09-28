package bench

import (
	"cmp"
	"fmt"
	"io"
	"maps"
	"math"
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
	fmt.Fprintf(w, "%s  %s\n", st.Bold("groma bench · "+s.Plugin), st.Dim(fmt.Sprintf("%s · %s", count(len(runs), "run"), duration(r.DurationSeconds))))
	fmt.Fprintln(w, st.Dim(fmt.Sprintf("main session %s · judge %s · Claude Code %s · about $%.2f at list price",
		cmp.Or(r.Suite.ModelOverride, "Claude Code's default model"), cmp.Or(judge, "claude plugin eval's default"), r.ClaudeVersion, r.CostUSD)))
	if r.Partial {
		fmt.Fprintln(w, st.Yellow(fmt.Sprintf("⚠ Partial: the run stopped early (%s), so fewer runs than planned were scored.", r.PartialReason)))
	}

	fmt.Fprintf(w, "\n%s\n", st.Bold("Agents"))
	var agentRows [][]style.Cell
	var notes []string
	for _, sc := range scores {
		p, pOK := sc.Precision()
		pLo, pHi := sc.PrecisionInterval()
		rc, rOK := sc.Recall()
		rLo, rHi := sc.RecallInterval()
		agentRows = append(agentRows, []style.Cell{
			{Text: short(sc.Agent), Styled: st.Bold(short(sc.Agent))},
			agentModel(st, s.Models[sc.Agent]),
			share(st, p, pOK, pLo, pHi), share(st, rc, rOK, rLo, rHi),
			work(st, sc.Checks), work(st, sc.Judge),
			style.Plain(fmt.Sprintf("%s · %s", count(sc.Scenarios(), "scenario"), count(sc.Expected+sc.Other, "run"))),
		})
		if len(sc.Instead) > 0 {
			notes = append(notes, st.Yellow(fmt.Sprintf("› %s was missed in %s: Claude picked %s", short(sc.Agent), count(sc.Expected-sc.Hit, "run"), listCounts(sc.Instead, short))))
		}
		if len(sc.WrongCases) > 0 {
			notes = append(notes, st.Red(fmt.Sprintf("› %s was picked wrongly in %s", short(sc.Agent), listCounts(sc.WrongCases, func(c string) string { return c }))))
		}
	}
	st.Table(w, []string{"Agent", "Model", "Precision", "Recall", "Work: checks", "Work: judge", "Based on"}, agentRows)
	for _, n := range notes {
		fmt.Fprintln(w, n)
	}

	fmt.Fprintf(w, "\n%s\n", st.Bold("Cases"))
	byCase := map[string][]Run{}
	for _, run := range runs {
		byCase[run.Case] = append(byCase[run.Case], run)
	}
	type caseRow struct {
		cells []style.Cell
		right float64
		name  string
	}
	var rows []caseRow
	for _, c := range s.Cases {
		if len(byCase[c.Name]) == 0 {
			continue
		}
		expects := make([]string, len(c.Expect))
		for i, a := range c.Expect {
			expects[i] = short(a)
		}
		result, others, right := outcomes(st, c, byCase[c.Name], short)
		rows = append(rows, caseRow{[]style.Cell{style.Plain(c.Name), style.Plain(cmp.Or(strings.Join(expects, ", "), "no agent")), result, others}, right, c.Name})
	}
	// Cases that went wrong come first: they are what the report is for.
	slices.SortStableFunc(rows, func(a, b caseRow) int { return cmp.Or(cmp.Compare(a.right, b.right), strings.Compare(a.name, b.name)) })
	caseRows := make([][]style.Cell, len(rows))
	for i, r := range rows {
		caseRows[i] = r.cells
	}
	st.Table(w, []string{"Case", "Expects", "Right", "Otherwise"}, caseRows)

	fmt.Fprintln(w)
	for _, line := range []string{
		"Precision: when Claude picked the agent, how often it was right. Recall: when the agent was needed, how often Claude picked it.",
		"The small range after each score is its 95% interval, wider when a scenario's rephrasings disagree.",
		"Work: the share of the case's checks passed when the agent was rightly picked; checks are exact, the judge is a model.",
	} {
		fmt.Fprintln(w, st.Dim(line))
	}
	if undecided > 0 {
		fmt.Fprintln(w, st.Dim(fmt.Sprintf("Left out, since they timed out before picking any agent: %s.", count(undecided, "run"))))
	}
	if failed > 0 {
		fmt.Fprintln(w, st.Dim(fmt.Sprintf("Ended in an error such as a timeout: %s. Timed-out runs are left out of work.", count(failed, "run"))))
	}
}

// agentModel names the model an agent runs on: the one its frontmatter sets,
// or the main session's when it sets none or asks to inherit it.
func agentModel(st style.Style, model string) style.Cell {
	if model == "" || model == "inherit" {
		return style.Cell{Text: "main session's", Styled: st.Dim("main session's")}
	}
	return style.Plain(model)
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

func percent(v float64) string {
	return fmt.Sprintf("%d%%", int(math.Round(v*100)))
}

// share prints a rate as a percentage, followed by its interval.
func share(st style.Style, value float64, ok bool, lo, hi float64) style.Cell {
	if !ok {
		return style.Cell{Text: "—", Styled: st.Dim("—")}
	}
	v := percent(value)
	interval := fmt.Sprintf("%d–%d", int(math.Round(lo*100)), int(math.Round(hi*100)))
	return style.Cell{Text: v + "  " + interval, Styled: grade(st, value)(v) + "  " + st.Dim(interval)}
}

func work(st style.Style, m Mean) style.Cell {
	v, n, ok := m.Value()
	if !ok {
		return style.Cell{Text: "—", Styled: st.Dim("—")}
	}
	basis := "of " + count(n, "run")
	return style.Cell{Text: percent(v) + "  " + basis, Styled: grade(st, v)(percent(v)) + "  " + st.Dim(basis)}
}

// outcomes sums up a case's runs: how many picked the right agent, and what
// the others did. right is the share of right runs, for sorting.
func outcomes(st style.Style, c Case, runs []Run, short func(string) string) (result, others style.Cell, right float64) {
	hits, decided := 0, 0
	otherwise := map[string]int{}
	for _, run := range runs {
		if run.TimedOut && len(run.Dispatched) == 0 {
			otherwise["timed out"]++
			continue
		}
		decided++
		names := make([]string, len(run.Dispatched))
		for i, a := range run.Dispatched {
			names[i] = short(a)
		}
		if len(c.Expect) == 0 && len(names) == 0 ||
			slices.ContainsFunc(run.Dispatched, func(a string) bool { return slices.Contains(c.Expect, a) }) {
			hits++
			continue
		}
		otherwise[cmp.Or(strings.Join(names, "+"), "no agent")]++
	}
	text := fmt.Sprintf("%d/%d", hits, decided)
	if decided > 0 {
		right = float64(hits) / float64(decided)
	}
	switch {
	case decided == 0:
		result = style.Cell{Text: "— " + text, Styled: st.Dim("— " + text)}
	case hits == decided:
		result = style.Cell{Text: "✔ " + text, Styled: st.Green("✔ " + text)}
	default:
		result = style.Cell{Text: "✖ " + text, Styled: st.Red("✖ " + text)}
	}
	if len(otherwise) == 0 {
		return result, style.Cell{Text: "—", Styled: st.Dim("—")}, right
	}
	list := listCounts(otherwise, func(k string) string { return k })
	return result, style.Cell{Text: list, Styled: st.Yellow(list)}, right
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
