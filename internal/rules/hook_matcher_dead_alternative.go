package rules

import (
	"fmt"
	"strings"

	"github.com/Maxime-Quesnel/groma/internal/claudecode"
	"github.com/Maxime-Quesnel/groma/internal/component"
	"github.com/Maxime-Quesnel/groma/internal/fix"
	"github.com/Maxime-Quesnel/groma/internal/rule"
	"slices"
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
	Fix: fixDeadAlternatives,
}

// fixDeadAlternatives removes the values of a matcher that name a tool
// Claude Code no longer has, which match nothing today. With unsafe, it also
// corrects a tool name's case, which makes the hook fire on that tool.
func fixDeadAlternatives(c *component.Component, t *component.Tree, unsafe bool) []fix.Edit {
	var edits []fix.Edit
	for _, g := range c.Hooks.Groups {
		alternatives, exact := claudecode.MatcherAlternatives(g.Matcher)
		if !exact || !slices.Contains(claudecode.ToolEvents, g.Event) {
			continue
		}
		var kept []string
		dead := 0
		for _, a := range alternatives {
			if deadMatcher(g.Event, a) != "" {
				dead++
			}
			switch correct := toolCase(a); {
			case claudecode.FormerTools[a] != "":
			case unsafe && correct != "" && correct != a:
				kept = append(kept, correct)
			default:
				kept = append(kept, a)
			}
		}
		if dead == len(alternatives) || len(kept) == len(alternatives) && strings.Join(kept, "") == strings.Join(alternatives, "") {
			continue
		}
		edits = append(edits, jsonValueEdits(c, "matcher", jsonString(g.Matcher), jsonString(joinMatcher(g.Matcher, kept)))...)
	}
	return edits
}

// toolCase returns the Claude Code tool name matching name in any case.
func toolCase(name string) string {
	if i := slices.IndexFunc(claudecode.Tools, func(tool string) bool { return strings.EqualFold(tool, name) }); i >= 0 {
		return claudecode.Tools[i]
	}
	return ""
}

// joinMatcher writes matcher values with the separator the original used.
func joinMatcher(original string, values []string) string {
	if strings.Contains(original, "|") {
		return strings.Join(values, "|")
	}
	return strings.Join(values, ", ")
}
