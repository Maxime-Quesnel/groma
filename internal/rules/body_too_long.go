package rules

import (
	"strings"

	"github.com/Maxime-Quesnel/groma/internal/claudecode"
	"github.com/Maxime-Quesnel/groma/internal/component"
	"github.com/Maxime-Quesnel/groma/internal/rule"
)

const maxBodyLines = 500

var bodyTooLong = Rule{
	Meta: rule.Meta{
		ID:    "body-too-long",
		Level: rule.Warning,
		Title: "Body over 500 lines",
		Description: "Once invoked, the whole body enters Claude's context and competes with the conversation. " +
			"Claude Code's documentation and Anthropic's guidance keep SKILL.md under 500 lines and move detail into files it links to, which Claude reads only when needed.",
		Remediation: "Keep the overview and the steps in the body; move reference material, long examples and edge cases into separate files linked from it.",
		FalsePositives: []string{
			"A long body made mostly of a table or data Claude needs on every run.",
		},
		References: []string{claudecode.SkillsDocs, claudecode.BestPractices},
	},
	Kinds: skillsAndCommands,
	Check: func(c *component.Component, t *component.Tree) []string {
		lines := strings.Count(strings.Trim(c.Header.Body, "\n"), "\n") + 1
		if lines <= maxBodyLines {
			return nil
		}
		return []string{at(c.Header.BodyLine, "the body runs %d lines", lines)}
	},
}
