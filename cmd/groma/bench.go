package main

import (
	"cmp"
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/Maxime-Quesnel/groma/internal/bench"
	"github.com/Maxime-Quesnel/groma/internal/style"
)

const benchUsage = `Usage: groma bench [flags] <plugin>

Measures, for each agent of a Claude Code plugin, how precisely Claude routes
work to it: precision, recall and what it confuses it with, with 95% intervals.
The ground truth is the plugin's own eval suite: every case grader that requires
or forbids an agent. groma copies the plugin, adds one dispatch check per agent
to each case, and runs the copy with claude plugin eval.

Unlike scan, bench runs Claude: through your Claude Code login, so it counts
against your plan's usage, with the model you chose in Claude Code unless
--model sets another. The installed plugin is never modified.

Run groma bench with no argument in a terminal to choose everything from lists.

Flags:
`

// benchConfig is one benchmark to run, from flags or from the prompts.
type benchConfig struct {
	pluginDir, extraCases string
	// agents are names with or without the plugin prefix; empty means all.
	agents            []string
	runs, concurrency int
	model, judge      string
	maxCost           float64
	scaffold          bool
	allowTools        []string
	dryRun            bool
	from              string
}

func runBench(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 && interactive() {
		return runBenchPrompts(stdout, stderr)
	}
	flags := flag.NewFlagSet("bench", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.Usage = func() { fmt.Fprint(stderr, benchUsage); flags.PrintDefaults() }
	agents := flags.String("agent", "", "score only these agents, comma-separated, running only the cases that expect or forbid them")
	cases := flags.String("cases", "", "an extra directory of eval cases, next to the plugin's own")
	runs := flags.Int("runs", 5, "runs per case")
	model := flags.String("model", "", "model under test; defaults to the one set in Claude Code. Set it to compare results over time")
	judge := flags.String("judge-model", "", "model that grades the cases' llm graders; claude plugin eval's default is a small fast one")
	maxCost := flags.Float64("max-cost-usd", 0, "stop once the runs' list-price estimate reaches this amount")
	concurrency := flags.Int("concurrency", 1, "runs at once, 1 to 8")
	scaffold := flags.Bool("scaffold", false, "run the cases' scaffold scripts, as you")
	allowTools := flags.String("allow-tools", "", "tools to grant beyond read-only ones, comma-separated, such as Edit,Write")
	dryRun := flags.Bool("dry-run", false, "show the ground truth and the planned runs, and run nothing")
	from := flags.String("from", "", "score an earlier result.json again instead of running, at no cost")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	// Flags may also follow the plugin path.
	if rest := flags.Args(); len(rest) > 1 {
		if err := flags.Parse(rest[1:]); err != nil {
			return 2
		}
		args = append([]string{rest[0]}, flags.Args()...)
	} else {
		args = rest
	}
	if len(args) != 1 {
		flags.Usage()
		return 2
	}
	return executeBench(benchConfig{
		pluginDir: args[0], extraCases: *cases, agents: split(*agents),
		runs: *runs, concurrency: *concurrency, model: *model, judge: *judge, maxCost: *maxCost,
		scaffold: *scaffold, allowTools: split(*allowTools), dryRun: *dryRun, from: *from,
	}, stdout, stderr)
}

func split(list string) []string {
	var items []string
	for _, item := range strings.Split(list, ",") {
		if item = strings.TrimSpace(item); item != "" {
			items = append(items, item)
		}
	}
	return items
}

func loadSuite(pluginDir, extraCases string) (bench.Suite, error) {
	caseDirs := []string{filepath.Join(pluginDir, "evals")}
	if extraCases != "" {
		caseDirs = append(caseDirs, extraCases)
	}
	return bench.Load(pluginDir, caseDirs...)
}

// selection returns the agents to score and the cases to run: every agent
// and case, or the named agents and the cases that expect or forbid one.
func selection(s bench.Suite, names []string) (agents []string, cases []bench.Case, err error) {
	if len(names) == 0 {
		return s.Agents, s.Cases, nil
	}
	for _, name := range names {
		if !strings.Contains(name, ":") {
			name = s.Plugin + ":" + name
		}
		if !slices.Contains(s.Agents, name) {
			return nil, nil, fmt.Errorf("%s has no agent %s", s.Plugin, name)
		}
		agents = append(agents, name)
	}
	for _, c := range s.Cases {
		if slices.ContainsFunc(agents, func(a string) bool { return slices.Contains(c.Expect, a) || slices.Contains(c.Avoid, a) }) {
			cases = append(cases, c)
		}
	}
	return agents, cases, nil
}

func executeBench(cfg benchConfig, stdout, stderr io.Writer) int {
	suite, err := loadSuite(cfg.pluginDir, cfg.extraCases)
	if err != nil {
		fmt.Fprintf(stderr, "groma: %v\n", err)
		return 2
	}
	agents, selected, err := selection(suite, cfg.agents)
	if err != nil {
		fmt.Fprintf(stderr, "groma: %v\n", err)
		return 2
	}
	if cfg.from != "" {
		return scoreResult(cfg.from, "", cfg.judge, suite, agents, stdout, stderr)
	}
	printPlan(stdout, suite, agents, selected, cfg.runs)
	if cfg.dryRun {
		return 0
	}
	work := filepath.Join("groma-bench", suite.Plugin+"-"+time.Now().Format("20060102-150405"))
	copyDir, err := bench.Prepare(suite, cfg.pluginDir, work, selected)
	if err != nil {
		fmt.Fprintf(stderr, "groma: %v\n", err)
		return 2
	}
	resultPath, _ := filepath.Abs(filepath.Join(work, "result.json"))
	fmt.Fprintf(stdout, "%s\n\n", style.For(stdout).Dim(fmt.Sprintf("Running on your Claude Code plan with %s · workspace %s", cmp.Or(cfg.model, "your default model"), work)))
	o := bench.Options{Runs: cfg.runs, Model: cfg.model, JudgeModel: cfg.judge, MaxCostUSD: cfg.maxCost,
		Concurrency: cfg.concurrency, Scaffold: cfg.scaffold, AllowTools: cfg.allowTools}
	if err := bench.Eval(context.Background(), copyDir, resultPath, o, stdout, stderr); err != nil {
		fmt.Fprintf(stderr, "groma: claude plugin eval: %v\n", err)
		return 2
	}
	return scoreResult(resultPath, filepath.Join(work, "report.txt"), cfg.judge, suite, agents, stdout, stderr)
}

// scoreResult reports on a claude plugin eval result, and saves the report
// to reportPath when it is set.
func scoreResult(resultPath, reportPath, judge string, suite bench.Suite, agents []string, stdout, stderr io.Writer) int {
	result, err := bench.ReadResult(resultPath)
	if err != nil {
		fmt.Fprintf(stderr, "groma: %v\n", err)
		return 2
	}
	runs := result.Runs(suite)
	var report strings.Builder
	bench.Text(io.MultiWriter(stdout, &report), suite, result, judge, runs, bench.ScoreAgents(suite, agents, runs))
	if reportPath == "" {
		return 0
	}
	if err := os.WriteFile(reportPath, []byte(report.String()), 0o644); err != nil {
		fmt.Fprintf(stderr, "groma: %v\n", err)
		return 2
	}
	return 0
}

func printPlan(w io.Writer, s bench.Suite, agents []string, cases []bench.Case, runs int) {
	st := style.For(w)
	short := func(names []string) string {
		out := make([]string, len(names))
		for i, n := range names {
			out[i] = strings.TrimPrefix(n, s.Plugin+":")
		}
		return strings.Join(out, ", ")
	}
	scenarios := map[string]bool{}
	for _, c := range cases {
		scenarios[c.Scenario] = true
	}
	fmt.Fprintf(w, "%s  %s\n\n", st.Bold("groma bench · "+s.Plugin),
		st.Dim(fmt.Sprintf("%s in %s · %s each · %s", count(len(cases), "case"), count(len(scenarios), "scenario"), count(runs, "run"), count(len(cases)*runs, "run"))))
	fmt.Fprintln(w, st.Dim(fmt.Sprintf("%-52s %-28s %s", "CASE", "EXPECTS", "FORBIDS")))
	for _, c := range cases {
		fmt.Fprintf(w, "%-52s %s %s\n", c.Name, style.Pad(cmp.Or(short(c.Expect), "no agent"), 28, st.Green), st.Red(short(c.Avoid)))
	}
	var uncovered []string
	for _, a := range agents {
		if !slices.ContainsFunc(s.Cases, func(c bench.Case) bool { return slices.Contains(c.Expect, a) }) {
			uncovered = append(uncovered, a)
		}
	}
	if len(uncovered) > 0 {
		fmt.Fprintf(w, "\n%s\n", st.Yellow("⚠ No case expects "+short(uncovered)+", so recall can't be measured for it."))
	}
	fmt.Fprintln(w)
}
