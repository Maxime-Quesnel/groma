package claudecode

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSelectedModelFollowsClaudeCodePrecedence(t *testing.T) {
	config, project := t.TempDir(), t.TempDir()
	write := func(path, content string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("ANTHROPIC_MODEL", "")
	t.Setenv("ANTHROPIC_DEFAULT_MODEL", "")

	if model, source := SelectedModel(config, project); model != "" || source != "" {
		t.Errorf("nothing set: got %q from %q", model, source)
	}

	t.Setenv("ANTHROPIC_DEFAULT_MODEL", "haiku")
	if model, _ := SelectedModel(config, project); model != "haiku" {
		t.Errorf("default env: got %q", model)
	}

	userSettings := filepath.Join(config, "settings.json")
	write(userSettings, `{"model": "opus", "effortLevel": "high"}`)
	if model, source := SelectedModel(config, project); model != "opus" || source != userSettings {
		t.Errorf("/model choice: got %q from %q", model, source)
	}

	write(filepath.Join(project, ".claude", "settings.json"), `{"model": "sonnet"}`)
	write(filepath.Join(project, ".claude", "settings.local.json"), `{"model": "fable"}`)
	if model, _ := SelectedModel(config, project); model != "fable" {
		t.Errorf("local over project over user: got %q", model)
	}

	t.Setenv("ANTHROPIC_MODEL", "claude-opus-5-5")
	if model, source := SelectedModel(config, project); model != "claude-opus-5-5" || source != "ANTHROPIC_MODEL" {
		t.Errorf("env over settings: got %q from %q", model, source)
	}
}
