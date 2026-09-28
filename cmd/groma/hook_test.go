package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

func hookInput(file string) string {
	b, _ := json.Marshal(map[string]any{
		"hook_event_name": "PostToolUse",
		"tool_name":       "Write",
		"cwd":             filepath.Dir(file),
		"tool_input":      map[string]string{"file_path": file},
	})
	return string(b)
}

func TestHookTellsClaudeWhatToFix(t *testing.T) {
	dir := t.TempDir()
	err := os.CopyFS(dir, fstest.MapFS{
		".claude-plugin/plugin.json":     {Data: []byte(`{"name": "shop"}`)},
		"agents/rails.md":                {Data: []byte("---\nname: shop:rails\ndescription: Reviews Rails code. Use after a change.\ntools: Read\n---\n\nYou review Rails code and report each bug.\n")},
		"skills/pdf/SKILL.md":            {Data: []byte("---\nname: pdf\ndescription: Handles PDFs.\n---\n\nSee [forms](references/forms.md).\n")},
		"skills/pdf/references/forms.md": {Data: []byte("# Forms\n")},
		"skills/pdf/scripts/fill.py":     {Data: []byte("print('fill')\n")},
		"src/app.rb":                     {Data: []byte("puts 1\n")},
		"notes/README.md":                {Data: []byte("# Notes\n")},
	})
	if err != nil {
		t.Fatal(err)
	}

	for _, tt := range []struct {
		file, want string
	}{
		{"agents/rails.md", `"decision":"block"`},
		{"skills/pdf/SKILL.md", `"hookEventName":"PostToolUse","additionalContext":"groma check found 1 warning`},
		{"skills/pdf/references/forms.md", "description-no-trigger"},
		{"src/app.rb", ""},
		{"notes/README.md", ""},
	} {
		var stdout strings.Builder
		code := run([]string{"hook"}, strings.NewReader(hookInput(filepath.Join(dir, tt.file))), &stdout, &strings.Builder{})

		if code != 0 || tt.want == "" && stdout.Len() > 0 || !strings.Contains(stdout.String(), tt.want) {
			t.Errorf("%s: exit %d, output %s", tt.file, code, stdout.String())
		}
	}
}

func TestHookReasonExplainsTheRedFlag(t *testing.T) {
	dir := t.TempDir()
	agent := filepath.Join(dir, "agents", "rails.md")
	os.MkdirAll(filepath.Dir(agent), 0o755)
	os.WriteFile(agent, []byte("---\nname: shop:rails\ndescription: Reviews Rails code. Use after a change.\ntools: Read\n---\n\nYou review Rails code and report each bug.\n"), 0o644)
	var stdout strings.Builder

	run([]string{"hook"}, strings.NewReader(hookInput(agent)), &stdout, &strings.Builder{})

	var out struct{ Decision, Reason string }
	if err := json.Unmarshal([]byte(stdout.String()), &out); err != nil {
		t.Fatalf("%v: %s", err, stdout.String())
	}
	for _, want := range []string{"1 red flag", "red flag, agent-not-loaded: Claude Code skips this agent.", "name: shop:rails starts with - or contains :", "Fix: Give the agent a name", "Fix the red flags before moving on"} {
		if !strings.Contains(out.Reason, want) {
			t.Errorf("reason lacks %q:\n%s", want, out.Reason)
		}
	}
}

func TestHookStaysQuietOnWhatItCantRead(t *testing.T) {
	for _, input := range []string{"", "not json", `{"tool_input": {}}`, hookInput("/nonexistent/agents/x.md")} {
		var stdout strings.Builder
		if code := run([]string{"hook"}, strings.NewReader(input), &stdout, &strings.Builder{}); code != 0 || stdout.Len() > 0 {
			t.Errorf("%q: exit %d, output %q", input, code, stdout.String())
		}
	}
}
