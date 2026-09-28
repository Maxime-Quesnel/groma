package rules

import (
	"github.com/Maxime-Quesnel/groma/internal/claudecode"
	"github.com/Maxime-Quesnel/groma/internal/component"
	"github.com/Maxime-Quesnel/groma/internal/rule"
)

var bypassPermissions = Rule{
	Meta: rule.Meta{
		ID:    "bypass-permissions",
		Level: rule.RedFlag,
		Title: "Agent runs every tool without asking",
		Description: "With permissionMode: bypassPermissions the agent runs shell commands, edits and web requests with no permission prompt. " +
			"Whatever steers it, such as a prompt injection in a file or page it reads, acts on the user's machine unchecked.",
		Remediation: "Use acceptEdits or default, and pre-approve in settings only the commands the agent needs.",
		FalsePositives: []string{
			"An agent meant to run only inside a disposable sandbox or container.",
		},
		References: []string{claudecode.SubagentsDocs},
	},
	Kinds: agents,
	Check: func(c *component.Component, t *component.Tree) []string {
		// A plugin agent's permissionMode is ignored; field-ignored says so.
		if !headerReadable(c) || c.InPlugin || c.Header.Value("permissionMode") != "bypassPermissions" {
			return nil
		}
		return []string{fieldAt(c, "permissionMode", "permissionMode: bypassPermissions")}
	},
}
