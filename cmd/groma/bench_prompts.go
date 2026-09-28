package main

import (
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/Maxime-Quesnel/groma/internal/agent/claudecode"
	"github.com/Maxime-Quesnel/groma/internal/bench"
	"github.com/Maxime-Quesnel/groma/internal/prompt"
)

// secondsPerRun is what a run took on average in the first benchmarks, for
// the duration shown before starting.
const secondsPerRun = 80

func interactive() bool {
	for _, f := range []*os.File{os.Stdin, os.Stdout} {
		if info, err := f.Stat(); err != nil || info.Mode()&os.ModeCharDevice == 0 {
			return false
		}
	}
	return true
}

type pluginChoice struct {
	dir, where, version string
	suite               bench.Suite
}

func runBenchPrompts(stdout, stderr io.Writer) int {
	home, _ := os.UserHomeDir()
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(stderr, "groma: %v\n", err)
		return 2
	}
	plugins := findPlugins(cwd, cmp.Or(os.Getenv("CLAUDE_CONFIG_DIR"), filepath.Join(home, ".claude")), home)
	if len(plugins) == 0 {
		fmt.Fprintln(stderr, "groma: no plugin with agents here or installed. Run groma bench from a plugin or marketplace directory, or pass a path.")
		return 2
	}
	term, err := prompt.Open()
	if err != nil {
		fmt.Fprintf(stderr, "groma: %v\n", err)
		return 2
	}
	cfg, err := askBench(term, plugins)
	term.Close()
	if errors.Is(err, prompt.ErrCanceled) {
		fmt.Fprintln(stdout, "Canceled.")
		return 0
	}
	if err != nil {
		fmt.Fprintf(stderr, "groma: %v\n", err)
		return 2
	}
	fmt.Fprintf(stdout, "\nSame run next time, or in CI:\n  %s\n\n", commandLine(cfg, cwd, home))
	return executeBench(cfg, stdout, stderr)
}

// findPlugins lists the plugins groma bench can run from here: the one in the
// current directory or the ones below it, as in a marketplace repository, or
// the installed ones when there is none here. It skips plugins without agents.
func findPlugins(cwd, configDir, home string) []pluginChoice {
	if here := choices(findLocal(cwd)); len(here) > 0 {
		return here
	}
	var dirs []pluginChoice
	for _, l := range claudecode.Installed(configDir, cwd) {
		if _, err := os.Stat(filepath.Join(l.Path, ".claude-plugin", "plugin.json")); err != nil {
			continue
		}
		where := "installed"
		if l.For != "" {
			where = "installed for " + tilde(l.For, home)
		}
		dirs = append(dirs, pluginChoice{dir: l.Path, where: where})
	}
	return choices(dirs)
}

func findLocal(cwd string) []pluginChoice {
	var dirs []pluginChoice
	for _, pattern := range []string{"", "*", filepath.Join("plugins", "*")} {
		matches, _ := filepath.Glob(filepath.Join(cwd, pattern, ".claude-plugin", "plugin.json"))
		for _, m := range matches {
			dir := filepath.Dir(filepath.Dir(m))
			where, _ := filepath.Rel(cwd, dir)
			dirs = append(dirs, pluginChoice{dir: dir, where: cmp.Or(where, ".")})
		}
	}
	return dirs
}

// choices keeps the plugins that have agents, once each, with their version.
func choices(dirs []pluginChoice) []pluginChoice {
	var plugins []pluginChoice
	var seen []string
	for _, p := range dirs {
		real, err := filepath.EvalSymlinks(p.dir)
		if err != nil || slices.Contains(seen, real) {
			continue
		}
		seen = append(seen, real)
		if p.suite, err = loadSuite(p.dir, ""); err == nil && len(p.suite.Agents) > 0 {
			var manifest struct {
				Version string `json:"version"`
			}
			content, _ := os.ReadFile(filepath.Join(p.dir, ".claude-plugin", "plugin.json"))
			json.Unmarshal(content, &manifest)
			p.version = manifest.Version
			plugins = append(plugins, p)
		}
	}
	return plugins
}

func count(n int, noun string) string {
	if n == 1 {
		return fmt.Sprintf("1 %s", noun)
	}
	return fmt.Sprintf("%d %ss", n, noun)
}

func askBench(term *prompt.Terminal, plugins []pluginChoice) (benchConfig, error) {
	var cfg benchConfig
	options := make([]prompt.Option, len(plugins))
	for i, p := range plugins {
		options[i] = prompt.Option{Label: strings.TrimSpace(p.suite.Plugin + " " + p.version),
			Hint: fmt.Sprintf("%s, %s · %s", count(len(p.suite.Agents), "agent"), count(len(p.suite.Cases), "case"), p.where)}
	}
	i, err := term.Select("Which plugin?", options, 0)
	if err != nil {
		return cfg, err
	}
	p := plugins[i]
	cfg.pluginDir = p.dir

	agents := make([]prompt.Option, len(p.suite.Agents))
	for i, a := range p.suite.Agents {
		agents[i] = prompt.Option{Label: strings.TrimPrefix(a, p.suite.Plugin+":"), Hint: agentHint(p.suite, a)}
	}
	for len(cfg.agents) == 0 {
		checked, err := term.Check("Which agents should the benchmark score?", agents)
		if err != nil {
			return cfg, err
		}
		for _, a := range checked {
			if a.Checked {
				cfg.agents = append(cfg.agents, a.Label)
			}
		}
		if len(cfg.agents) == 0 {
			term.Say("  Check at least one agent with the space bar.")
		}
	}
	_, cases, err := selection(p.suite, cfg.agents)
	if err != nil {
		return cfg, err
	}

	runs := []int{1, 3, 5, 10}
	i, err = term.Select("How many runs per case?", []prompt.Option{
		{Label: "1 run", Hint: "a quick look"},
		{Label: "3 runs", Hint: "recommended"},
		{Label: "5 runs", Hint: "tighter intervals"},
		{Label: "10 runs", Hint: "a baseline to compare against later"},
	}, 1)
	if err != nil {
		return cfg, err
	}
	cfg.runs = runs[i]

	concurrency := []int{1, 2, 4}
	i, err = term.Select("How many runs at once?", []prompt.Option{
		{Label: "1 at a time", Hint: "slowest"},
		{Label: "2 at a time", Hint: "recommended"},
		{Label: "4 at a time", Hint: "faster, but runs can slow down or time out on your plan's rate limit"},
	}, 1)
	if err != nil {
		return cfg, err
	}
	cfg.concurrency = concurrency[i]

	models := []string{"", "sonnet", "opus", "haiku"}
	i, err = term.Select("Which model should Claude use?", []prompt.Option{
		{Label: "Your Claude Code default", Hint: "recommended: the model you work with"},
		{Label: "sonnet"}, {Label: "opus"}, {Label: "haiku"},
	}, 0)
	if err != nil {
		return cfg, err
	}
	cfg.model = models[i]

	checked, err := term.Check("Options", []prompt.Option{
		{Label: "Run the cases' scaffold scripts", Hint: "they set up each case's workspace, and run as you", Checked: slices.ContainsFunc(cases, hasScaffold)},
		{Label: "Let Claude edit files (Edit, Write)", Hint: "most cases grade the files Claude writes", Checked: true},
		{Label: "Judge with Sonnet", Hint: "the default judge is small and can refuse right work phrased another way", Checked: true},
	})
	if err != nil {
		return cfg, err
	}
	cfg.scaffold = checked[0].Checked
	if checked[1].Checked {
		cfg.allowTools = []string{"Edit", "Write"}
	}
	if checked[2].Checked {
		cfg.judge = "sonnet"
	}

	total := len(cases) * cfg.runs
	term.Say("  %s · %s × %s = %s · about %s with %d at a time",
		strings.Join(cfg.agents, ", "), count(len(cases), "case"), count(cfg.runs, "run"), count(total, "run"), estimate(total, cfg.concurrency), cfg.concurrency)
	term.Say("  Claude runs through your Claude Code login: this counts against your plan.")
	i, err = term.Select("Start?", []prompt.Option{
		{Label: "Start"},
		{Label: "Only show the plan", Hint: "a dry run, free"},
		{Label: "Cancel"},
	}, 0)
	if err != nil {
		return cfg, err
	}
	switch i {
	case 1:
		cfg.dryRun = true
	case 2:
		return cfg, prompt.ErrCanceled
	}
	return cfg, nil
}

func agentHint(s bench.Suite, agent string) string {
	about, expected := 0, 0
	for _, c := range s.Cases {
		if slices.Contains(c.Expect, agent) {
			expected++
		}
		if slices.Contains(c.Expect, agent) || slices.Contains(c.Avoid, agent) {
			about++
		}
	}
	switch {
	case about == 0:
		return "no case: nothing to measure yet"
	case expected == 0:
		return count(about, "case") + ", none expects it: precision only"
	}
	return count(about, "case")
}

func hasScaffold(c bench.Case) bool {
	content, err := os.ReadFile(filepath.Join(c.Dir, "case.yaml"))
	return err == nil && strings.Contains(string(content), "scaffold_script")
}

func estimate(runs, concurrency int) string {
	d := time.Duration(runs*secondsPerRun/max(concurrency, 1)) * time.Second
	if d < time.Hour {
		return fmt.Sprintf("%d min", max(1, int(d.Round(time.Minute).Minutes())))
	}
	return fmt.Sprintf("%d h %02d", int(d.Hours()), int(d.Minutes())%60)
}

// commandLine spells out cfg as flags, so a run chosen from the prompts can
// be repeated without them.
func commandLine(cfg benchConfig, cwd, home string) string {
	args := []string{"groma bench", "--agent " + strings.Join(cfg.agents, ","), fmt.Sprintf("--runs %d", cfg.runs), fmt.Sprintf("--concurrency %d", cfg.concurrency)}
	if cfg.model != "" {
		args = append(args, "--model "+cfg.model)
	}
	if cfg.judge != "" {
		args = append(args, "--judge-model "+cfg.judge)
	}
	if cfg.scaffold {
		args = append(args, "--scaffold")
	}
	if len(cfg.allowTools) > 0 {
		args = append(args, "--allow-tools "+strings.Join(cfg.allowTools, ","))
	}
	if cfg.dryRun {
		args = append(args, "--dry-run")
	}
	dir := tilde(cfg.pluginDir, home)
	if rel, err := filepath.Rel(cwd, cfg.pluginDir); err == nil && !strings.HasPrefix(rel, "..") {
		dir = rel
	}
	return strings.Join(append(args, dir), " ")
}
