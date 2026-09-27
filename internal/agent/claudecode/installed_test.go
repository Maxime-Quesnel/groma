package claudecode

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
	"testing/fstest"
)

func TestInstalledListsWhatClaudeCodeLoads(t *testing.T) {
	root := t.TempDir()
	config, project := filepath.Join(root, "config"), filepath.Join(root, "project")
	active := filepath.Join(config, "plugins", "cache", "shop", "fmt", "2.0.0")
	older := filepath.Join(config, "plugins", "cache", "shop", "fmt", "1.0.0")
	registry := `{"version": 2, "plugins": {"fmt@shop": [{"scope": "user", "installPath": "` + active + `"},
		{"scope": "project", "projectPath": "/work/legacy", "installPath": "` + older + `"},
		{"scope": "project", "projectPath": "/work/new", "installPath": "` + active + `"}],
		"gone@shop": [{"scope": "user", "installPath": "` + filepath.Join(root, "deleted") + `"}]}}`
	err := os.CopyFS(root, fstest.MapFS{
		"config/settings.json":                          {Data: []byte("{}")},
		"config/skills/tidy/SKILL.md":                   {Data: []byte("tidy")},
		"config/plugins/installed_plugins.json":         {Data: []byte(registry)},
		"config/plugins/cache/shop/fmt/2.0.0/README.md": {Data: []byte("active")},
		"config/plugins/cache/shop/fmt/1.0.0/README.md": {Data: []byte("older")},
		"config/plugins/cache/shop/fmt/0.9.0/README.md": {Data: []byte("stale")},
		"config/plugins/marketplaces/shop/README.md":    {Data: []byte("not installed")},
		"project/.claude/settings.json":                 {Data: []byte("{}")},
		"project/CLAUDE.md":                             {Data: []byte("rules")},
	})
	if err != nil {
		t.Fatal(err)
	}

	got := Installed(config, project)

	want := []Location{
		{Path: filepath.Join(config, "settings.json")},
		{Path: filepath.Join(config, "skills")},
		{Path: active},
		{Path: older, For: "/work/legacy"},
		{Path: filepath.Join(project, ".claude")},
		{Path: filepath.Join(project, "CLAUDE.md")},
	}
	if !slices.Equal(got, want) {
		t.Errorf("got\n%+v\nwant\n%+v", got, want)
	}
}
