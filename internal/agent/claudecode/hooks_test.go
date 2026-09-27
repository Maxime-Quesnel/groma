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
		got[h.Source+" "+h.Trigger] = h
	}

	want := map[string]plugin.Hook{
		"plugins/fmt/hooks/hooks.json SessionStart hook": {Trigger: "SessionStart hook", Command: `"${CLAUDE_PLUGIN_ROOT}"/scripts/start.sh --quiet`,
			Source: "plugins/fmt/hooks/hooks.json", Scripts: []string{"plugins/fmt/scripts/start.sh"}},
		"plugins/fmt/.claude-plugin/plugin.json Stop hook": {Trigger: "Stop hook", Command: "bash -c echo done",
			Source: "plugins/fmt/.claude-plugin/plugin.json"},
		"plugins/fmt/config/extra.json PreToolUse hook": {Trigger: "PreToolUse hook", Command: "$CLAUDE_PLUGIN_ROOT/missing.sh",
			Source: "plugins/fmt/config/extra.json"},
		".claude/settings.json UserPromptSubmit hook": {Trigger: "UserPromptSubmit hook", Command: "sh ${CLAUDE_PROJECT_DIR}/.claude/hooks/check.sh",
			Source: ".claude/settings.json", Scripts: []string{".claude/hooks/check.sh"}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got\n%+v\nwant\n%+v", got, want)
	}
}

func TestHooksReadsTheInlineShellOfSkillsAndCommands(t *testing.T) {
	files := map[string]string{
		"plugins/fmt/skills/pr/SKILL.md":   "---\nname: pr\n---\n\n- Diff: !`gh pr diff`\n- Not run: KEY=!`date`\n\n```!\nbash \"${CLAUDE_SKILL_DIR}/collect.sh\"\n```\n",
		"plugins/fmt/skills/pr/collect.sh": "echo collect",
		"plugins/fmt/commands/sync.md":     "Sync first: !`${CLAUDE_PLUGIN_ROOT}/scripts/sync.sh`",
		"plugins/fmt/scripts/sync.sh":      "echo sync",
		"plugins/fmt/README.md":            "Run !`make docs` to build the docs.",
	}
	var p plugin.Plugin
	for path, content := range files {
		p.Files = append(p.Files, plugin.File{Path: path, Content: []byte(content)})
	}

	got := map[string]plugin.Hook{}
	for _, h := range Hooks(p) {
		got[h.Command] = h
	}

	want := map[string]plugin.Hook{
		"gh pr diff": {Trigger: "inline shell", Command: "gh pr diff", Source: "plugins/fmt/skills/pr/SKILL.md"},
		`bash "${CLAUDE_SKILL_DIR}/collect.sh"`: {Trigger: "inline shell", Command: `bash "${CLAUDE_SKILL_DIR}/collect.sh"`,
			Source: "plugins/fmt/skills/pr/SKILL.md", Scripts: []string{"plugins/fmt/skills/pr/collect.sh"}},
		"${CLAUDE_PLUGIN_ROOT}/scripts/sync.sh": {Trigger: "inline shell", Command: "${CLAUDE_PLUGIN_ROOT}/scripts/sync.sh",
			Source: "plugins/fmt/commands/sync.md", Scripts: []string{"plugins/fmt/scripts/sync.sh"}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got\n%+v\nwant\n%+v", got, want)
	}
}
