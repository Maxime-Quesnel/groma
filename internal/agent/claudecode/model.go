package claudecode

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
)

// SelectedModel returns the model Claude Code starts new sessions with for a
// user working in project, and where that choice comes from, following
// Claude Code's own precedence: ANTHROPIC_MODEL, then the model field of the
// managed, local, project and user settings, which is where /model saves the
// user's choice, then ANTHROPIC_DEFAULT_MODEL. It returns an empty model
// when nothing is set, which leaves Claude Code's default for the account.
func SelectedModel(configDir, project string) (model, source string) {
	if m := os.Getenv("ANTHROPIC_MODEL"); m != "" {
		return m, "ANTHROPIC_MODEL"
	}
	for _, file := range []string{
		managedSettings(),
		filepath.Join(project, ".claude", "settings.local.json"),
		filepath.Join(project, ".claude", "settings.json"),
		filepath.Join(configDir, "settings.json"),
	} {
		content, err := os.ReadFile(file)
		if err != nil {
			continue
		}
		var settings struct {
			Model string `json:"model"`
		}
		if json.Unmarshal(content, &settings) == nil && settings.Model != "" {
			return settings.Model, file
		}
	}
	if m := os.Getenv("ANTHROPIC_DEFAULT_MODEL"); m != "" {
		return m, "ANTHROPIC_DEFAULT_MODEL"
	}
	return "", ""
}

func managedSettings() string {
	if runtime.GOOS == "darwin" {
		return "/Library/Application Support/ClaudeCode/managed-settings.json"
	}
	return "/etc/claude-code/managed-settings.json"
}
