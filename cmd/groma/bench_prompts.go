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
	"github.com/Maxime-Quesnel/groma/internal/style"
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
	model, label := userModel()
	cfg, err := askBench(term, plugins, model, label)
	term.Close()
	if errors.Is(err, prompt.ErrCanceled) {
		fmt.Fprintln(stdout, "Canceled.")
		return 0
	}
	if err != nil {
		fmt.Fprintf(stderr, "groma: %v\n", err)
		return 2
	}
	st := style.For(stdout)
	fmt.Fprintf(stdout, "\n%s\n  %s\n\n", st.Dim("Same run next time, or in CI:"), st.Cyan(commandLine(cfg, cwd, home)))
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

// askBench asks only what changes from one benchmark to the next: the
// plugin when there is more than one, the agents, the model, and how
// thorough to be. The model question starts on detected, the model groma
// found for the user, labelled as label. The rest takes the defaults the
// first benchmarks settled on; flags change it, and the command printed
// afterwards shows them.
func askBench(term *prompt.Terminal, plugins []pluginChoice, detected, label string) (benchConfig, error) {
	cfg := benchConfig{concurrency: 2, judge: "sonnet", allowTools: []string{"Edit", "Write"}}
	st := style.On()
	p := plugins[0]
	if len(plugins) == 1 {
		term.Done("Plugin", pluginLabel(p))
	} else {
		options := make([]prompt.Option, len(plugins))
		for i, p := range plugins {
			options[i] = prompt.Option{Label: pluginLabel(p),
				Hint: fmt.Sprintf("%s, %s · %s", count(len(p.suite.Agents), "agent"), count(len(p.suite.Cases), "case"), p.where)}
		}
		i, err := term.Select("Plugin", options, 0)
		if err != nil {
			return cfg, err
		}
		p = plugins[i]
	}
	cfg.pluginDir = p.dir

	agents := make([]prompt.Option, len(p.suite.Agents))
	for i, a := range p.suite.Agents {
		agents[i] = prompt.Option{Label: strings.TrimPrefix(a, p.suite.Plugin+":"), Hint: agentHint(p.suite, a)}
	}
	for len(cfg.agents) == 0 {
		checked, err := term.Check("Agents", agents)
		if err != nil {
			return cfg, err
		}
		for _, a := range checked {
			if a.Checked {
				cfg.agents = append(cfg.agents, a.Label)
			}
		}
		if len(cfg.agents) == 0 {
			term.Say("  %s", st.Yellow("Check at least one agent with the space bar."))
		}
	}
	_, cases, err := selection(p.suite, cfg.agents)
	if err != nil {
		return cfg, err
	}
	cfg.scaffold = slices.ContainsFunc(cases, hasScaffold)

	models, options := modelChoices(detected, label)
	i, err := term.Select("Model", options, 0)
	if err != nil {
		return cfg, err
	}
	cfg.model = models[i]

	depth := func(runs int) string {
		total := len(cases) * runs
		return fmt.Sprintf("%s per case · %s · about %s", count(runs, "run"), count(total, "run"), estimate(total, cfg.concurrency))
	}
	term.Say("  %s", st.Dim("Claude runs on your Claude Code plan. Other settings: groma bench -h"))
	i, err = term.Select("Run", []prompt.Option{
		{Label: "Standard", Hint: depth(3)},
		{Label: "Quick look", Hint: depth(1)},
		{Label: "Precise", Hint: depth(5)},
		{Label: "Show the plan only", Hint: "free"},
		{Label: "Cancel"},
	}, 0)
	if err != nil {
		return cfg, err
	}
	cfg.runs = []int{3, 1, 5, 3, 0}[i]
	switch i {
	case 3:
		cfg.dryRun = true
	case 4:
		return cfg, prompt.ErrCanceled
	}
	return cfg, nil
}

// modelChoices lists the models the main session can run on: the one groma
// detected first, then Claude Code's aliases. The main session is the one
// that decides which agent to call; agents keep their own model.
func modelChoices(detected, label string) ([]string, []prompt.Option) {
	models := []string{detected}
	options := []prompt.Option{{Label: cmp.Or(detected, "default"), Hint: strings.TrimPrefix(label, detected+", ")}}
	for _, alias := range []struct{ name, hint string }{
		{"opus", "Opus 5.5"}, {"sonnet", "Sonnet 5"}, {"fable", "Fable 5.1"}, {"haiku", "the latest Haiku"},
	} {
		if alias.name != detected {
			models = append(models, alias.name)
			options = append(options, prompt.Option{Label: alias.name, Hint: alias.hint})
		}
	}
	return models, options
}

func pluginLabel(p pluginChoice) string {
	return strings.TrimSpace(p.suite.Plugin + " " + p.version)
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
