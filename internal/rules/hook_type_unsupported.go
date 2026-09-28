package rules

import (
	"fmt"
	"slices"
	"strings"

	"github.com/Maxime-Quesnel/groma/internal/claudecode"
	"github.com/Maxime-Quesnel/groma/internal/component"
	"github.com/Maxime-Quesnel/groma/internal/rule"
)

var hookTypeUnsupported = Rule{
	Meta: rule.Meta{
		ID:    "hook-type-unsupported",
		Level: rule.RedFlag,
		Title: "Hook type the event doesn't run",
		Description: "Not every event runs every hook type: SessionStart and Setup only run command and mcp_tool hooks, eighteen events such as Notification, " +
			"SessionEnd and SubagentStart don't run prompt or agent hooks, and PermissionRequest doesn't run agent hooks. Claude Code skips a hook of the wrong type.",
		Remediation: "Use a command hook, which every event runs, or move the check to an event that supports the type.",
		FalsePositives: []string{
			"An event given a new hook type by a Claude Code release newer than groma's table.",
		},
		References: []string{claudecode.HooksDocs},
	},
	Kinds: hooks,
	Check: func(c *component.Component, t *component.Tree) []string {
		var evidence []string
		for _, h := range c.Hooks.Handlers {
			if _, known := claudecode.HandlerFields[h.Type]; !known || !slices.Contains(claudecode.HookEvents, h.Event) {
				continue
			}
			if types := claudecode.HookTypes(h.Event); !slices.Contains(types, h.Type) {
				evidence = append(evidence, fmt.Sprintf("%s is a %s hook, and %s only runs %s", h.Where(), h.Type, h.Event, strings.Join(types, ", ")))
			}
		}
		return evidence
	},
}
