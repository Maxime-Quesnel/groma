package bench

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"testing/fstest"
)

func testPlugin(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	picks := "---\ntype: tool_used\ntool: Agent\ninput_match: '\"subagent_type\"\\s*:\\s*\"shop:rails\"'\nmin: 1\nweight: 1\n---\n"
	avoids := "---\ntype: tool_used\ntool: Agent\ninput_match: '\"subagent_type\"\\s*:\\s*\"shop:rails\"'\nmin: 0\nmax: 0\n---\n"
	err := os.CopyFS(dir, fstest.MapFS{
		".claude-plugin/plugin.json":              {Data: []byte(`{"name": "shop"}`)},
		"agents/rails.md":                         {Data: []byte("---\nname: rails\nmodel: sonnet\n---\n")},
		"agents/ruby.md":                          {Data: []byte("---\nname: ruby\n---\n")},
		"evals/slow-page/prompt.md":               {Data: []byte("Make the page fast.")},
		"evals/slow-page/graders/picks.md":        {Data: []byte(picks)},
		"evals/slow-page/graders/eager.md":        {Data: []byte("---\ntype: regex\npattern: includes\n---\n")},
		"evals/hot-loop/case.yaml":                {Data: []byte("name: hot-loop")},
		"evals/hot-loop/graders/avoids-rails.md":  {Data: []byte(avoids)},
		"evals/results/old/aggregate-result.json": {Data: []byte("{}")},
		"evals/README.md":                         {Data: []byte("suite")},
	})
	if err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestLoadTakesGroundTruthFromTheCasesGraders(t *testing.T) {
	dir := testPlugin(t)

	s, err := Load(dir, filepath.Join(dir, "evals"))
	if err != nil {
		t.Fatal(err)
	}

	if s.Plugin != "shop" || !slices.Equal(s.Agents, []string{"shop:rails", "shop:ruby"}) {
		t.Errorf("plugin %q agents %v", s.Plugin, s.Agents)
	}
	if !reflect.DeepEqual(s.Models, map[string]string{"shop:rails": "sonnet"}) {
		t.Errorf("models %v", s.Models)
	}
	got := map[string][2][]string{}
	for _, c := range s.Cases {
		got[c.Name] = [2][]string{c.Expect, c.Avoid}
	}
	want := map[string][2][]string{
		"hot-loop":  {nil, {"shop:rails"}},
		"slow-page": {{"shop:rails"}, nil},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestLoadAcceptsAPluginWithoutEvals(t *testing.T) {
	dir := testPlugin(t)
	extra := filepath.Join(dir, "evals")
	if err := os.Rename(extra, filepath.Join(t.TempDir(), "moved")); err != nil {
		t.Fatal(err)
	}

	s, err := Load(dir, filepath.Join(dir, "evals"))

	if err != nil || len(s.Cases) != 0 || len(s.Agents) != 2 {
		t.Errorf("got %+v, %v", s, err)
	}
}

func TestPrepareLeavesThePluginAloneAndAddsDispatchGraders(t *testing.T) {
	dir := testPlugin(t)
	s, err := Load(dir, filepath.Join(dir, "evals"))
	if err != nil {
		t.Fatal(err)
	}
	work := t.TempDir()

	copyDir, err := Prepare(s, dir, work, s.About("shop:rails")[:1])
	if err != nil {
		t.Fatal(err)
	}

	entries, _ := os.ReadDir(filepath.Join(copyDir, "evals"))
	if len(entries) != 1 {
		t.Errorf("copied cases %v, want only the selected one", entries)
	}
	grader, err := os.ReadFile(filepath.Join(copyDir, "evals", s.About("shop:rails")[0].Name, "graders", "groma-called-shop--ruby.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(grader), `input_match: '"subagent_type"\s*:\s*"shop:ruby"'`) || !strings.Contains(string(grader), "weight: 0.001") {
		t.Errorf("grader:\n%s", grader)
	}
	if _, err := os.Stat(filepath.Join(dir, "evals", "slow-page", "graders", "groma-called-shop--ruby.md")); err == nil {
		t.Error("wrote into the plugin itself")
	}
}

func TestScoreCountsHitsMissesAndWrongDispatches(t *testing.T) {
	s := Suite{Plugin: "shop", Agents: []string{"shop:rails", "shop:ruby"}, Cases: []Case{
		{Name: "slow-page", Expect: []string{"shop:rails"}},
		{Name: "hot-loop", Expect: []string{"shop:ruby"}, Avoid: []string{"shop:rails"}},
	}}
	runs := []Run{
		{Case: "slow-page", Dispatched: []string{"shop:rails"}},
		{Case: "slow-page", Dispatched: []string{"shop:rails"}},
		{Case: "slow-page", Dispatched: []string{"shop:ruby"}},
		{Case: "slow-page"},
		{Case: "hot-loop", Dispatched: []string{"shop:ruby"}},
		{Case: "hot-loop", Dispatched: []string{"shop:rails"}},
	}

	sc := ScoreAgents(s, []string{"shop:rails"}, runs)[0]

	if sc.Expected != 4 || sc.Hit != 2 || sc.Other != 2 || sc.Wrong != 1 {
		t.Errorf("got %+v", sc)
	}
	if p, _ := sc.Precision(); math.Abs(p-2.0/3) > 1e-9 {
		t.Errorf("precision %v", p)
	}
	if r, _ := sc.Recall(); r != 0.5 {
		t.Errorf("recall %v", r)
	}
	if !reflect.DeepEqual(sc.Instead, map[string]int{"shop:ruby": 1, "": 1}) || !reflect.DeepEqual(sc.WrongCases, map[string]int{"hot-loop": 1}) {
		t.Errorf("instead %v wrong %v", sc.Instead, sc.WrongCases)
	}
}

func TestWilsonMatchesKnownIntervals(t *testing.T) {
	for _, tc := range []struct {
		k, n   int
		lo, hi float64
	}{
		{8, 10, 0.4902, 0.9433},
		{10, 10, 0.7225, 1},
		{0, 10, 0, 0.2775},
		{45, 50, 0.7864, 0.9565},
	} {
		lo, hi := wilson(tc.k, tc.n)
		if math.Abs(lo-tc.lo) > 1e-3 || math.Abs(hi-tc.hi) > 1e-3 {
			t.Errorf("wilson(%d, %d) = %.4f–%.4f, want %.4f–%.4f", tc.k, tc.n, lo, hi, tc.lo, tc.hi)
		}
	}
}

func TestRunsReadWhichAgentsWereDispatched(t *testing.T) {
	s := Suite{Plugin: "botyglot-dev", Agents: []string{"botyglot-dev:rails-expert", "botyglot-dev:ruby-expert"}}
	r, err := ReadResult("testdata/result.json")
	if err != nil {
		t.Fatal(err)
	}

	runs := r.Runs(s)

	want := []Run{{Case: "probe-index", Dispatched: []string{"botyglot-dev:rails-expert"}}}
	if !reflect.DeepEqual(runs, want) || r.Suite.ModelOverride != "claude-sonnet-5" || r.ClaudeVersion != "2.1.280" {
		t.Errorf("got %+v from %+v", runs, r)
	}
}

func TestIntervalsWidenWhenScenariosDisagree(t *testing.T) {
	s := Suite{Plugin: "shop", Agents: []string{"shop:rails"}, Cases: []Case{
		{Name: "index", Scenario: "index", Expect: []string{"shop:rails"}},
		{Name: "index--fr", Scenario: "index", Expect: []string{"shop:rails"}},
		{Name: "cache", Scenario: "cache", Expect: []string{"shop:rails"}},
	}}
	var runs []Run
	for range 5 {
		runs = append(runs, Run{Case: "index", Dispatched: []string{"shop:rails"}}, Run{Case: "index--fr", Dispatched: []string{"shop:rails"}}, Run{Case: "cache"})
	}

	sc := ScoreAgents(s, s.Agents, runs)[0]

	wLo, wHi := wilson(10, 15)
	lo, hi := sc.RecallInterval()
	if sc.Scenarios() != 2 || lo >= wLo || hi <= wHi {
		t.Errorf("scenarios %d, interval %.2f–%.2f, want wider than the per-run %.2f–%.2f", sc.Scenarios(), lo, hi, wLo, wHi)
	}
}

func TestIntervalsKeepWilsonWhenScenariosAgree(t *testing.T) {
	var s Suite
	var runs []Run
	for _, name := range []string{"a", "b", "c", "d"} {
		s.Cases = append(s.Cases, Case{Name: name, Scenario: name, Expect: []string{"shop:rails"}})
		for range 5 {
			runs = append(runs, Run{Case: name, Dispatched: []string{"shop:rails"}})
		}
	}

	sc := ScoreAgents(s, []string{"shop:rails"}, runs)[0]

	wLo, wHi := wilson(20, 20)
	if lo, hi := sc.RecallInterval(); lo != wLo || hi != wHi {
		t.Errorf("interval %.4f–%.4f, want Wilson %.4f–%.4f", lo, hi, wLo, wHi)
	}
}

func TestRunsScoreWorkGradersOnly(t *testing.T) {
	var r Result
	err := json.Unmarshal([]byte(`{"cases": [{"name": "slow-page",
		"graders": [{"name": "picks", "type": "tool_used", "config": {"tool": "Agent"}},
			{"name": "eager", "type": "regex", "config": {}}, {"name": "no-edit", "type": "tool_used", "config": {"tool": "Edit"}},
			{"name": "fast", "type": "llm", "config": {}}, {"name": "groma-called-shop--rails", "type": "tool_used", "config": {"tool": "Agent"}}],
		"arms": {"with": [{"error": null, "graders": [{"name": "picks", "passed": true, "weight": 1},
			{"name": "eager", "passed": true, "weight": 3}, {"name": "no-edit", "passed": false, "weight": 1},
			{"name": "fast", "passed": false, "weight": 3}, {"name": "groma-called-shop--rails", "passed": true, "weight": 0.001}]}]}}]}`), &r)
	if err != nil {
		t.Fatal(err)
	}

	runs := r.Runs(Suite{Plugin: "shop", Agents: []string{"shop:rails"}})

	checks, hasChecks := runs[0].Checks.share()
	judge, hasJudge := runs[0].Judge.share()
	if len(runs) != 1 || !hasChecks || checks != 0.75 || !hasJudge || judge != 0 || !slices.Equal(runs[0].Dispatched, []string{"shop:rails"}) {
		t.Errorf("got %+v", runs)
	}
}

func TestEvalLeavesTheModelAndSpendToTheUser(t *testing.T) {
	plain := strings.Join(evalArgs("/w/shop", "/w/result.json", Options{Runs: 3}), " ")
	pinned := strings.Join(evalArgs("/w/shop", "/w/result.json", Options{Runs: 3, Model: "claude-sonnet-5", JudgeModel: "sonnet", MaxCostUSD: 20, Scaffold: true, AllowTools: []string{"Edit", "Write"}}), " ")

	if strings.Contains(plain, "--model") || strings.Contains(plain, "--judge-model") || strings.Contains(plain, "--max-cost-usd") || !strings.Contains(plain, "--no-publish") {
		t.Errorf("default args: %s", plain)
	}
	if !strings.Contains(pinned, "--model claude-sonnet-5") || !strings.Contains(pinned, "--judge-model sonnet") || !strings.Contains(pinned, "--max-cost-usd 20") ||
		!strings.HasSuffix(pinned, "--scaffold --allow-tools Edit Write") {
		t.Errorf("pinned args: %s", pinned)
	}
}

func TestCheckClaudeNamesAClaudeWithoutPluginEval(t *testing.T) {
	tests := []struct {
		name, evalHelp, want string
	}{
		{"current release", "Usage: claude plugin eval [options] [target]\n  --ablation <mode>", ""},
		{"older release", "Usage: claude plugin|plugins [options] [command]", "claude is Claude Code 2.1.81, which has no plugin eval"},
		{"not installed", "", "claude not found on PATH"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bin := t.TempDir()
			if tt.evalHelp != "" {
				script := "#!/bin/sh\nif [ \"$1\" = --version ]; then echo '2.1.81 (Claude Code)'; else echo '" + tt.evalHelp + "'; fi\n"
				if err := os.WriteFile(filepath.Join(bin, "claude"), []byte(script), 0o755); err != nil {
					t.Fatal(err)
				}
			}
			t.Setenv("PATH", bin)

			err := CheckClaude(t.Context())

			if tt.want == "" && err != nil {
				t.Errorf("got %v", err)
			}
			if tt.want != "" && (err == nil || !strings.Contains(err.Error(), tt.want)) {
				t.Errorf("got %v, want %q", err, tt.want)
			}
		})
	}
}

func TestTimedOutRunsWithoutDispatchDecideNothing(t *testing.T) {
	s := Suite{Plugin: "shop", Agents: []string{"shop:rails"}, Cases: []Case{
		{Name: "slow-page", Scenario: "slow-page", Expect: []string{"shop:rails"}},
		{Name: "hot-loop", Scenario: "hot-loop", Expect: []string{"shop:ruby"}},
	}}
	runs := []Run{
		{Case: "slow-page", Dispatched: []string{"shop:rails"}, Checks: graded{passed: 2, total: 2}},
		{Case: "slow-page", Dispatched: []string{"shop:rails"}, Checks: graded{total: 2}, TimedOut: true, Failed: true},
		{Case: "slow-page", TimedOut: true, Failed: true},
		{Case: "hot-loop", TimedOut: true, Failed: true},
	}

	sc := ScoreAgents(s, s.Agents, runs)[0]

	checks, n, _ := sc.Checks.Value()
	if _, _, judged := sc.Judge.Value(); sc.Expected != 2 || sc.Hit != 2 || sc.Other != 0 || sc.Undecided != 2 || n != 1 || checks != 1 || judged {
		t.Errorf("got %+v", sc)
	}
}

func TestTextPutsTheCasesThatWentWrongFirst(t *testing.T) {
	s := Suite{Plugin: "shop", Agents: []string{"shop:rails", "shop:ruby"}, Models: map[string]string{"shop:rails": "sonnet"}, Cases: []Case{
		{Name: "a-fine-case", Scenario: "a", Expect: []string{"shop:rails"}},
		{Name: "b-broken-case", Scenario: "b", Expect: []string{"shop:rails"}},
	}}
	runs := []Run{
		{Case: "a-fine-case", Dispatched: []string{"shop:rails"}},
		{Case: "b-broken-case", Dispatched: []string{"shop:ruby"}},
		{Case: "b-broken-case", Dispatched: []string{"shop:rails"}},
		{Case: "b-broken-case", TimedOut: true, Failed: true},
	}
	var out strings.Builder

	Text(&out, s, Result{}, "sonnet", runs, ScoreAgents(s, []string{"shop:rails"}, runs))

	report := out.String()
	for _, want := range []string{
		"│ rails │ sonnet │ 100%  ",
		"main session Claude Code's default model · judge sonnet",
		"│ 67%  ",
		"› rails was missed in 1 run: Claude picked ruby ×1",
		"│ b-broken-case │ rails   │ ✖ 1/2 │ ruby ×1, timed out ×1 │",
		"│ a-fine-case   │ rails   │ ✔ 1/1 │ —                     │",
	} {
		if !strings.Contains(report, want) {
			t.Errorf("report lacks %q:\n%s", want, report)
		}
	}
	if strings.Index(report, "b-broken-case") > strings.Index(report, "a-fine-case") {
		t.Errorf("the broken case should come first:\n%s", report)
	}
}
