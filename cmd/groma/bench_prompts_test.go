package main

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/Maxime-Quesnel/groma/internal/prompt"
)

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

func TestAskBenchBuildsTheConfigFromKeys(t *testing.T) {
	root := testMarketplace(t)
	plugins := findPlugins(root, filepath.Join(t.TempDir(), "no-config"), t.TempDir())
	keys := strings.Join([]string{
		"\r",        // the only plugin
		"\r",        // no agent checked: asked again
		" \r",       // rails
		"\x1b[B\r",  // 5 runs
		"\r",        // 2 at a time
		"\x1b[B\r",  // sonnet
		"\x1b[B \r", // uncheck editing
		"\x1b[B\r",  // only show the plan
	}, "")
	var out strings.Builder

	cfg, err := askBench(prompt.New(strings.NewReader(keys), &out), plugins)

	if err != nil {
		t.Fatal(err)
	}
	if cfg.pluginDir != plugins[0].dir || !slices.Equal(cfg.agents, []string{"rails"}) || cfg.runs != 5 || cfg.concurrency != 2 ||
		cfg.model != "sonnet" || cfg.judge != "sonnet" || !cfg.scaffold || cfg.allowTools != nil || !cfg.dryRun {
		t.Errorf("got %+v", cfg)
	}
	if !strings.Contains(out.String(), "Check at least one agent") || !strings.Contains(out.String(), "1 case × 5 runs = 5 runs") || !strings.Contains(out.String(), "shop 1.2.0") {
		t.Errorf("output:\n%s", out.String())
	}

	if got := commandLine(cfg, root, t.TempDir()); got != "groma bench --agent rails --runs 5 --concurrency 2 --model sonnet --judge-model sonnet --scaffold --dry-run plugins/shop" {
		t.Errorf("command line %q", got)
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
