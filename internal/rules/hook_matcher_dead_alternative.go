package rules

import (
	"fmt"
	"strings"

	"github.com/Maxime-Quesnel/groma/internal/claudecode"
	"github.com/Maxime-Quesnel/groma/internal/component"
	"github.com/Maxime-Quesnel/groma/internal/rule"
)

var hookMatcherDeadAlternative = Rule{
	Meta: rule.Meta{
		ID:    "hook-matcher-dead-alternative",
		Level: rule.Warning,
		Title: "Matcher value that can never match",
		Description: "Part of the matcher can never match: a tool Claude Code no longer has, such as MultiEdit, a tool name in the wrong case, " +
			"or a value the event doesn't use. The hook still runs on the other values, but the dead one suggests a case the author thinks is covered.",
		Remediation: "Remove the value, or correct it to one the event matches.",
		FalsePositives: []string{
			"A value kept for users of an older Claude Code release that still has the tool.",
		},
		References: []string{claudecode.HooksDocs},
	},
	Kinds: hooks,
	Check: func(c *component.Component, t *component.Tree) []string {
		var evidence []string
		for _, g := range c.Hooks.Groups {
			alternatives, exact := claudecode.MatcherAlternatives(g.Matcher)
			if !exact {
				continue
			}
			var reasons []string
			for _, a := range alternatives {
				if why := deadMatcher(g.Event, a); why != "" {
					reasons = append(reasons, why)
				}
			}
			// When nothing in the matcher can match, hook-never-runs reports it.
			if len(reasons) > 0 && len(reasons) < len(alternatives) {
				evidence = append(evidence, fmt.Sprintf("%s: %s", g.Where(), strings.Join(reasons, "; ")))
			}
		}
		return evidence
	},
}
