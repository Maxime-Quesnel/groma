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

Flags:
`

func runBench(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("bench", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.Usage = func() { fmt.Fprint(stderr, benchUsage); flags.PrintDefaults() }
	agent := flags.String("agent", "", "score only this agent, running only the cases that expect or forbid it")
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
	pluginDir := args[0]

	caseDirs := []string{filepath.Join(pluginDir, "evals")}
	if *cases != "" {
		caseDirs = append(caseDirs, *cases)
	}
	suite, err := bench.Load(pluginDir, caseDirs...)
	if err != nil {
		fmt.Fprintf(stderr, "groma: %v\n", err)
		return 2
	}
	agents, selected := suite.Agents, suite.Cases
	if *agent != "" {
		name := *agent
		if !strings.Contains(name, ":") {
			name = suite.Plugin + ":" + name
		}
		if !slices.Contains(suite.Agents, name) {
			fmt.Fprintf(stderr, "groma: %s has no agent %s\n", suite.Plugin, name)
			return 2
		}
		agents, selected = []string{name}, suite.About(name)
	}

	if *from != "" {
		return scoreResult(*from, "", *judge, suite, agents, stdout, stderr)
	}
	printPlan(stdout, suite, agents, selected, *runs)
	if *dryRun {
		return 0
	}
	work := filepath.Join("groma-bench", suite.Plugin+"-"+time.Now().Format("20060102-150405"))
	copyDir, err := bench.Prepare(suite, pluginDir, work, selected)
	if err != nil {
		fmt.Fprintf(stderr, "groma: %v\n", err)
		return 2
	}
	resultPath, _ := filepath.Abs(filepath.Join(work, "result.json"))
	fmt.Fprintf(stdout, "Running with %s, through your Claude Code plan. Workspace: %s\n\n", cmp.Or(*model, "your Claude Code default model"), work)
	o := bench.Options{Runs: *runs, Model: *model, JudgeModel: *judge, MaxCostUSD: *maxCost, Concurrency: *concurrency, Scaffold: *scaffold}
	if *allowTools != "" {
		o.AllowTools = strings.Split(*allowTools, ",")
	}
	if err := bench.Eval(context.Background(), copyDir, resultPath, o, stdout, stderr); err != nil {
		fmt.Fprintf(stderr, "groma: claude plugin eval: %v\n", err)
		return 2
	}
	return scoreResult(resultPath, filepath.Join(work, "report.txt"), *judge, suite, agents, stdout, stderr)
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
	fmt.Fprintf(w, "%s: %d agents, %d cases in %d scenarios selected, %d runs each: %d runs.\n\n",
		s.Plugin, len(s.Agents), len(cases), len(scenarios), runs, len(cases)*runs)
	fmt.Fprintf(w, "%-52s %-28s %s\n", "CASE", "EXPECTS", "FORBIDS")
	for _, c := range cases {
		fmt.Fprintf(w, "%-52s %-28s %s\n", c.Name, cmp.Or(short(c.Expect), "no agent named"), short(c.Avoid))
	}
	var uncovered []string
	for _, a := range agents {
		if !slices.ContainsFunc(s.Cases, func(c bench.Case) bool { return slices.Contains(c.Expect, a) }) {
			uncovered = append(uncovered, a)
		}
	}
	if len(uncovered) > 0 {
		fmt.Fprintf(w, "\nNo case expects these agents, so their recall can't be measured: %s\n", short(uncovered))
	}
	fmt.Fprintln(w)
}
