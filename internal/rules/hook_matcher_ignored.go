package rules

import (
	"fmt"
	"slices"

	"github.com/Maxime-Quesnel/groma/internal/claudecode"
	"github.com/Maxime-Quesnel/groma/internal/component"
	"github.com/Maxime-Quesnel/groma/internal/rule"
)

var hookMatcherIgnored = Rule{
	Meta: rule.Meta{
		ID:    "hook-matcher-ignored",
		Level: rule.RedFlag,
		Title: "Matcher on an event that ignores it",
		Description: "Some events, such as Stop, UserPromptSubmit and TaskCompleted, fire on every occurrence and ignore a matcher. " +
			"A hook meant to run only in some cases runs every time.",
		Remediation: "Remove the matcher, and filter inside the hook's command on the JSON Claude Code passes it.",
		FalsePositives: []string{
			"None known: an empty matcher and * are not flagged.",
		},
		References: []string{claudecode.HooksDocs},
	},
	Kinds: hooks,
	Check: func(c *component.Component, t *component.Tree) []string {
		var evidence []string
		for _, g := range c.Hooks.Groups {
			if g.HasMatcher && g.Matcher != "" && g.Matcher != "*" && slices.Contains(claudecode.EventsWithoutMatcher, g.Event) {
				evidence = append(evidence, fmt.Sprintf("%s: %s ignores matchers, so this group runs on every %s", g.Where(), g.Event, g.Event))
			}
		}
		return evidence
	},
}
