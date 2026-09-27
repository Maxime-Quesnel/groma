package claudecode

import (
	"reflect"
	"testing"

	"github.com/Maxime-Quesnel/groma/internal/plugin"
)

func TestHooksReadsEveryDeclaration(t *testing.T) {
	files := map[string]string{
		"plugins/fmt/hooks/hooks.json": `{"hooks": {"SessionStart": [{"hooks": [
			{"type": "command", "command": "\"${CLAUDE_PLUGIN_ROOT}\"/scripts/start.sh --quiet"},
			{"type": "http", "url": "https://audit.example.com"}]}]}}`,
		"plugins/fmt/scripts/start.sh":           "echo start",
		"plugins/fmt/.claude-plugin/plugin.json": `{"name": "fmt", "hooks": ["./config/extra.json", {"Stop": [{"hooks": [{"command": "bash", "args": ["-c", "echo done"]}]}]}]}`,
		"plugins/fmt/config/extra.json":          `{"hooks": {"PreToolUse": [{"matcher": "Bash", "hooks": [{"type": "command", "command": "$CLAUDE_PLUGIN_ROOT/missing.sh"}]}]}}`,
		".claude/settings.json":                  `{"permissions": {}, "hooks": {"UserPromptSubmit": [{"hooks": [{"type": "command", "command": "sh ${CLAUDE_PROJECT_DIR}/.claude/hooks/check.sh"}]}]}}`,
		".claude/hooks/check.sh":                 "echo check",
		"plugins/broken/hooks/hooks.json":        `{"hooks": `,
	}
	var p plugin.Plugin
	for path, content := range files {
		p.Files = append(p.Files, plugin.File{Path: path, Content: []byte(content)})
	}

	got := map[string]plugin.Hook{}
	for _, h := range Hooks(p) {
		got[h.Source+" "+h.Event] = h
	}

	want := map[string]plugin.Hook{
		"plugins/fmt/hooks/hooks.json SessionStart": {Event: "SessionStart", Command: `"${CLAUDE_PLUGIN_ROOT}"/scripts/start.sh --quiet`,
			Source: "plugins/fmt/hooks/hooks.json", Scripts: []string{"plugins/fmt/scripts/start.sh"}},
		"plugins/fmt/.claude-plugin/plugin.json Stop": {Event: "Stop", Command: "bash -c echo done",
			Source: "plugins/fmt/.claude-plugin/plugin.json"},
		"plugins/fmt/config/extra.json PreToolUse": {Event: "PreToolUse", Command: "$CLAUDE_PLUGIN_ROOT/missing.sh",
			Source: "plugins/fmt/config/extra.json"},
		".claude/settings.json UserPromptSubmit": {Event: "UserPromptSubmit", Command: "sh ${CLAUDE_PROJECT_DIR}/.claude/hooks/check.sh",
			Source: ".claude/settings.json", Scripts: []string{".claude/hooks/check.sh"}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got\n%+v\nwant\n%+v", got, want)
	}
}
