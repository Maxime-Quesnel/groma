package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var rules = []string{"description-emphatic", "reference-no-toc", "broken-link"}

func TestFindReadsTheNearestFileAbove(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, "plugins", "shop", "agents"), 0o755)
	os.WriteFile(filepath.Join(root, FileName), []byte(`# house style
disable:
  - description-emphatic
exclude: [plugins/legacy/**]
reference-no-toc:
  - plugins/shop/skills/**
`), 0o644)
	agent := filepath.Join(root, "plugins", "shop", "agents", "rails.md")
	os.WriteFile(agent, []byte("---\nname: rails\n---\n"), 0o644)

	c, err := Find(agent, rules)
	if err != nil || c.Path != filepath.Join(root, FileName) {
		t.Fatalf("got %+v, %v", c, err)
	}

	for _, tt := range []struct {
		rule, path string
		want       bool
	}{
		{"description-emphatic", "plugins/shop/agents/rails.md", false},
		{"broken-link", "plugins/legacy/skills/old/SKILL.md", false},
		{"broken-link", "plugins/legacy-v2/skills/new/SKILL.md", true},
		{"reference-no-toc", "plugins/shop/skills/pdf/SKILL.md", false},
		{"reference-no-toc", "plugins/other/skills/pdf/SKILL.md", true},
		{"broken-link", "plugins/shop/skills/pdf/SKILL.md", true},
	} {
		if got := c.Allows(tt.rule, filepath.Join(root, tt.path)); got != tt.want {
			t.Errorf("Allows(%s, %s) = %v", tt.rule, tt.path, got)
		}
	}
}

func TestFindWithoutFileAllowsEverything(t *testing.T) {
	c, err := Find(t.TempDir(), rules)

	if err != nil || c.Path != "" || !c.Allows("broken-link", "/anything") {
		t.Errorf("got %+v, %v", c, err)
	}
}

func TestFindRejectsWhatItCantUse(t *testing.T) {
	for content, want := range map[string]string{
		"disable: [brokn-link]\n": "line 1: groma has no rule brokn-link",
		"refrence-no-toc: [x]\n":  "line 1: groma has no rule refrence-no-toc",
		"disable: [broken-link\n": "the [ opening disable's list is never closed",
	} {
		dir := t.TempDir()
		os.WriteFile(filepath.Join(dir, FileName), []byte(content), 0o644)

		if _, err := Find(dir, rules); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%q: got %v, want %q", content, err, want)
		}
	}
}
