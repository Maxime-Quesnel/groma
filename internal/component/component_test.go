package component

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func writeTree(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for name, content := range files {
		p := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		mode := os.FileMode(0o644)
		if filepath.Ext(name) == ".sh" {
			mode = 0o755
		}
		if err := os.WriteFile(p, []byte(content), mode); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

type found struct {
	Kind            Kind
	Name            string
	InPlugin        bool
	Plugin, Project string
}

func TestLoadFindsComponentsWhereClaudeCodeLooks(t *testing.T) {
	root := writeTree(t, map[string]string{
		"plugins/shop/.claude-plugin/plugin.json":        `{"name": "shop", "hooks": "./config/extra.json"}`,
		"plugins/shop/skills/pdf/SKILL.md":               "---\nname: pdf-tools\ndescription: d\n---\n",
		"plugins/shop/skills/pdf/references/agents/x.md": "---\nname: not-an-agent\n---\n",
		"plugins/shop/skills/form/SKILL.md":              "Fill forms.\n",
		"plugins/shop/agents/review/rails.md":            "---\nname: rails\ndescription: d\n---\n",
		"plugins/shop/agents/README.md":                  "# Agents\n",
		"plugins/shop/commands/db/migrate.md":            "Migrate.\n",
		"plugins/shop/hooks/hooks.json":                  `{"hooks": {}}`,
		"plugins/shop/config/extra.json":                 `{"hooks": {}}`,
		"plugins/shop/evals/case/prompt.md":              "Do it.\n",
		"project/.claude/settings.json":                  `{"hooks": {"Stop": []}}`,
		"project/.claude/settings.local.json":            `{"permissions": {}}`,
		"project/.claude/agents/local.md":                "---\nname: local\n---\n",
		"project/.claude/skills/notes/SKILL.md":          "---\nname: notes\n---\n",
	})

	tree, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}

	got := map[string]found{}
	for _, c := range tree.Components {
		if c.Kind != Plugin {
			got[c.Path] = found{c.Kind, c.Name(), c.InPlugin, c.Plugin, c.Project}
		}
	}
	want := map[string]found{
		"plugins/shop/skills/pdf/SKILL.md":      {Skill, "pdf-tools", true, "plugins/shop", ""},
		"plugins/shop/skills/form/SKILL.md":     {Skill, "form", true, "plugins/shop", ""},
		"plugins/shop/agents/review/rails.md":   {Agent, "rails", true, "plugins/shop", ""},
		"plugins/shop/commands/db/migrate.md":   {Command, "db:migrate", true, "plugins/shop", ""},
		"plugins/shop/hooks/hooks.json":         {Hooks, "hooks", true, "plugins/shop", ""},
		"plugins/shop/config/extra.json":        {Hooks, "extra", true, "plugins/shop", ""},
		"project/.claude/settings.json":         {Hooks, "settings", false, "", "project"},
		"project/.claude/agents/local.md":       {Agent, "local", false, "", "project"},
		"project/.claude/skills/notes/SKILL.md": {Skill, "notes", false, "", "project"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got\n%+v\nwant\n%+v", got, want)
	}
}

func TestLoadOneFileReadsItsPlugin(t *testing.T) {
	root := writeTree(t, map[string]string{
		".claude-plugin/plugin.json": `{"name": "shop"}`,
		"agents/rails.md":            "---\nname: rails\n---\n",
		"agents/ruby.md":             "---\nname: ruby\n---\n",
		"scripts/check.sh":           "echo ok",
	})

	tree, err := Load(filepath.Join(root, "agents", "rails.md"))
	if err != nil {
		t.Fatal(err)
	}

	if len(tree.Targets) != 1 || tree.Targets[0].Path != "agents/rails.md" || !tree.Targets[0].InPlugin || tree.Targets[0].PluginName != "shop" {
		t.Fatalf("got %+v", tree.Targets)
	}
	if len(tree.Components) != 3 {
		t.Errorf("the plugin's other components weren't read: %d", len(tree.Components))
	}
	if _, ok := tree.File("scripts/check.sh"); !ok {
		t.Error("the plugin around the file wasn't read")
	}
}

func TestLoadGuessesAFileOutsideAnyComponentDirectory(t *testing.T) {
	root := writeTree(t, map[string]string{
		"reviewer.md": "---\nname: reviewer\ndescription: d\ntools: Read\n---\n",
		"deploy.md":   "---\ndescription: Deploy\n---\n",
		"notes.txt":   "notes",
	})

	for file, kind := range map[string]Kind{"reviewer.md": Agent, "deploy.md": Command} {
		tree, err := Load(filepath.Join(root, file))
		if err != nil || len(tree.Targets) != 1 || tree.Targets[0].Kind != kind || !tree.Targets[0].Guessed {
			t.Errorf("%s: got %+v, %v", file, tree, err)
		}
	}
	if _, err := Load(filepath.Join(root, "notes.txt")); err == nil {
		t.Error("a text file was accepted")
	}
}

func TestHooksReadEveryDeclarationForm(t *testing.T) {
	root := writeTree(t, map[string]string{
		".claude-plugin/plugin.json": `{"name": "fmt", "hooks": [{"Stop": [{"hooks": [{"type": "command", "command": "bash", "args": ["-c", "echo done"]}]}]}]}`,
		"hooks/hooks.json": `{"description": "x", "hooks": {"PreToolUse": [
			{"matcher": "Bash", "hooks": [{"type": "command", "command": "\"${CLAUDE_PLUGIN_ROOT}\"/scripts/check.sh --quiet"}, {"type": "http", "url": "https://audit.example.com"}]},
			{"hooks": "none"}], "Stop": {}}}`,
		"scripts/check.sh": "echo ok",
	})

	tree, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}

	byPath := map[string]*Component{}
	for _, c := range tree.Components {
		if c.Kind != Plugin {
			byPath[c.Path] = c
		}
	}
	hooks := byPath["hooks/hooks.json"].Hooks
	if len(hooks.Handlers) != 2 || hooks.Handlers[0].Where() != `PreToolUse (matcher "Bash"), hook 1` || hooks.Handlers[1].Type != "http" {
		t.Errorf("handlers: %+v", hooks.Handlers)
	}
	wantProblems := []string{
		`PreToolUse has no "hooks" list, so it runs nothing`,
		`Stop must be a list of matcher groups, [{"matcher": ..., "hooks": [...]}]`,
	}
	if !reflect.DeepEqual(hooks.Problems, wantProblems) {
		t.Errorf("problems: %q", hooks.Problems)
	}
	scripts := tree.Scripts(hooks.Handlers[0].Run(), byPath["hooks/hooks.json"].Roots())
	if !reflect.DeepEqual(scripts, []Script{{Ref: "${CLAUDE_PLUGIN_ROOT}/scripts/check.sh", Path: "scripts/check.sh", Found: true, Direct: true}}) {
		t.Errorf("scripts: %+v", scripts)
	}
	inline := byPath[".claude-plugin/plugin.json"].Hooks
	if len(inline.Handlers) != 1 || !inline.Handlers[0].Exec || inline.Handlers[0].Run() != "bash -c echo done" {
		t.Errorf("manifest: %+v", inline)
	}
}

func TestInlineShell(t *testing.T) {
	root := writeTree(t, map[string]string{
		"skills/pr/SKILL.md": "---\nname: pr\n---\n\n- Diff: !`gh pr diff`\n- Not run: KEY=!`date`\n\n```!\nbash \"${CLAUDE_SKILL_DIR}/collect.sh\"\n```\n",
	})
	tree, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}

	got := tree.Components[0].InlineShell()

	want := []Shell{{5, "gh pr diff"}, {9, `bash "${CLAUDE_SKILL_DIR}/collect.sh"`}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v", got)
	}
}
