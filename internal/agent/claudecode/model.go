package claudecode

import (
	"bufio"
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
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

// SessionModel returns the model of a running Claude Code session: the one
// its last answer came from, as its transcript records it. Claude Code gives
// the commands it runs the session's ID but not its model, and a model picked
// with /model for this session only is saved nowhere else.
func SessionModel(configDir, sessionID string) string {
	transcripts, _ := filepath.Glob(filepath.Join(configDir, "projects", "*", sessionID+".jsonl"))
	if sessionID == "" || len(transcripts) == 0 {
		return ""
	}
	f, err := os.Open(transcripts[0])
	if err != nil {
		return ""
	}
	defer f.Close()
	model := ""
	lines := bufio.NewReader(f)
	for {
		line, err := lines.ReadBytes('\n')
		if bytes.Contains(line, []byte(`"assistant"`)) {
			var entry struct {
				Type    string `json:"type"`
				Message struct {
					Model string `json:"model"`
				} `json:"message"`
			}
			// Claude Code writes errors and interruptions as <synthetic>
			// answers, which no model gave.
			if json.Unmarshal(line, &entry) == nil && entry.Type == "assistant" &&
				entry.Message.Model != "" && !strings.HasPrefix(entry.Message.Model, "<") {
				model = entry.Message.Model
			}
		}
		if err != nil {
			return model
		}
	}
}
