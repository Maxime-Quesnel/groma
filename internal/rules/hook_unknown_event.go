package rules

import (
	"fmt"
	"slices"

	"github.com/Maxime-Quesnel/groma/internal/claudecode"
	"github.com/Maxime-Quesnel/groma/internal/component"
	"github.com/Maxime-Quesnel/groma/internal/rule"
)

var hookUnknownEvent = Rule{
	Meta: rule.Meta{
		ID:    "hook-unknown-event",
		Level: rule.RedFlag,
		Title: "Hook on an event Claude Code doesn't have",
		Description: "Event names are case-sensitive, and Claude Code ignores hooks under a name it doesn't know, such as preToolUse or BeforeToolUse. " +
			"The hooks under it never run, including any guard meant to block a dangerous tool call.",
		Remediation: "Use one of the event names of the hooks reference, spelled exactly.",
		FalsePositives: []string{
			"An event added by a Claude Code release newer than groma's list.",
		},
		References: []string{claudecode.HooksDocs},
	},
	Kinds: hooks,
	Check: func(c *component.Component, t *component.Tree) []string {
		var evidence []string
		var seen []string
		for _, g := range c.Hooks.Groups {
			if slices.Contains(claudecode.HookEvents, g.Event) || slices.Contains(seen, g.Event) {
				continue
			}
			seen = append(seen, g.Event)
			line := fmt.Sprintf("%s isn't a hook event", g.Event)
			if s := suggest(g.Event, claudecode.HookEvents); s != "" {
				line += fmt.Sprintf("; did you mean %s?", s)
			}
			evidence = append(evidence, line)
		}
		return evidence
	},
}
