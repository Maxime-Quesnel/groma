package main

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/Maxime-Quesnel/groma/internal/prompt"
)

var ansi = regexp.MustCompile(`\x1b\[[0-9;?]*[A-Za-z]`)

func testMarketplace(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	picks := "---\ntype: tool_used\ntool: Agent\ninput_match: '\"subagent_type\"\\s*:\\s*\"shop:rails\"'\nmin: 1\n---\n"
	err := os.CopyFS(root, fstest.MapFS{
		"plugins/shop/.claude-plugin/plugin.json":   {Data: []byte(`{"name": "shop", "version": "1.2.0"}`)},
		"plugins/shop/agents/rails.md":              {Data: []byte("---\nname: rails\n---\n")},
		"plugins/shop/agents/ruby.md":               {Data: []byte("---\nname: ruby\n---\n")},
		"plugins/shop/evals/slow-page/case.yaml":    {Data: []byte("name: slow-page\ncontext:\n  scaffold_script: scaffold.sh\n")},
		"plugins/shop/evals/slow-page/graders/p.md": {Data: []byte(picks)},
		"plugins/docs/.claude-plugin/plugin.json":   {Data: []byte(`{"name": "docs"}`)},
		"plugins/docs/skills/write/SKILL.md":        {Data: []byte("---\nname: write\n---\n")},
	})
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func TestFindPluginsListsThoseWithAgents(t *testing.T) {
	root := testMarketplace(t)

	plugins := findPlugins(root, filepath.Join(t.TempDir(), "no-config"), t.TempDir())

	if len(plugins) != 1 || plugins[0].suite.Plugin != "shop" || plugins[0].where != filepath.Join("plugins", "shop") {
		t.Errorf("got %+v", plugins)
	}
}

func TestAskBenchAsksOnlyForAgentsAndDepth(t *testing.T) {
	root := testMarketplace(t)
	plugins := findPlugins(root, filepath.Join(t.TempDir(), "no-config"), t.TempDir())
	keys := strings.Join([]string{
		"\r",             // no agent checked: asked again
		" \r",            // rails
		"\x1b[B\x1b[B\r", // precise
	}, "")
	var out strings.Builder

	cfg, err := askBench(prompt.New(strings.NewReader(keys), &out), plugins)

	if err != nil {
		t.Fatal(err)
	}
	if cfg.pluginDir != plugins[0].dir || !slices.Equal(cfg.agents, []string{"rails"}) || cfg.runs != 5 || cfg.concurrency != 2 ||
		cfg.model != "" || cfg.judge != "sonnet" || !cfg.scaffold || !slices.Equal(cfg.allowTools, []string{"Edit", "Write"}) || cfg.dryRun {
		t.Errorf("got %+v", cfg)
	}
	plain := ansi.ReplaceAllString(out.String(), "")
	for _, want := range []string{"✔ Plugin  shop 1.2.0", "Check at least one agent", "5 runs per case · 5 runs", "✔ Run  Precise"} {
		if !strings.Contains(plain, want) {
			t.Errorf("output lacks %q:\n%s", want, plain)
		}
	}
	if got := commandLine(cfg, root, t.TempDir()); got != "groma bench --agent rails --runs 5 --concurrency 2 --judge-model sonnet --scaffold --allow-tools Edit,Write plugins/shop" {
		t.Errorf("command line %q", got)
	}
}

func TestAskBenchShowsThePlanOrCancels(t *testing.T) {
	root := testMarketplace(t)
	plugins := findPlugins(root, filepath.Join(t.TempDir(), "no-config"), t.TempDir())

	cfg, err := askBench(prompt.New(strings.NewReader(" \r\x1b[A\x1b[A\r"), &strings.Builder{}), plugins)
	if err != nil || !cfg.dryRun || cfg.runs != 3 {
		t.Errorf("plan only: got %+v, %v", cfg, err)
	}

	_, err = askBench(prompt.New(strings.NewReader(" \r\x1b[A\r"), &strings.Builder{}), plugins)
	if !errors.Is(err, prompt.ErrCanceled) {
		t.Errorf("cancel: got %v", err)
	}
}

func TestFindPluginsFallsBackToInstalledOnes(t *testing.T) {
	marketplace := testMarketplace(t)
	config := t.TempDir()
	installed := filepath.Join(marketplace, "plugins", "shop")
	registry := `{"plugins": {"shop@m": [{"scope": "project", "projectPath": "/work/x", "installPath": "` + installed + `"}]}}`
	if err := os.MkdirAll(filepath.Join(config, "plugins"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(config, "plugins", "installed_plugins.json"), []byte(registry), 0o644); err != nil {
		t.Fatal(err)
	}

	plugins := findPlugins(t.TempDir(), config, "/home/me")

	if len(plugins) != 1 || plugins[0].where != "installed for /work/x" || plugins[0].version != "1.2.0" {
		t.Errorf("got %+v", plugins)
	}
}

func TestEstimate(t *testing.T) {
	for _, tc := range []struct {
		runs, concurrency int
		want              string
	}{{5, 1, "7 min"}, {108, 2, "1 h 12"}, {1, 8, "1 min"}} {
		if got := estimate(tc.runs, tc.concurrency); got != tc.want {
			t.Errorf("estimate(%d, %d) = %q, want %q", tc.runs, tc.concurrency, got, tc.want)
		}
	}
}
