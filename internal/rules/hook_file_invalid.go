package rules

import (
	"github.com/Maxime-Quesnel/groma/internal/claudecode"
	"github.com/Maxime-Quesnel/groma/internal/component"
	"github.com/Maxime-Quesnel/groma/internal/rule"
)

var hookFileInvalid = Rule{
	Meta: rule.Meta{
		ID:    "hook-file-invalid",
		Level: rule.RedFlag,
		Title: "Claude Code can't read the hooks",
		Description: "The hooks file isn't valid JSON, or doesn't have the shape Claude Code reads: " +
			`{"hooks": {"<Event>": [{"matcher": "...", "hooks": [{"type": "command", ...}]}]}}. ` +
			"Claude Code then skips the file or the malformed part, and the hooks in it never run.",
		Remediation: "Fix the JSON so each event maps to a list of matcher groups, each with a hooks list of handler objects.",
		FalsePositives: []string{
			"None known: Claude Code reads hooks files with the same structure.",
		},
		References: []string{claudecode.HooksDocs},
	},
	Kinds: hooks,
	Check: func(c *component.Component, t *component.Tree) []string {
		return c.Hooks.Problems
	},
}
