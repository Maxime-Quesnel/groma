package rules

import (
	"unicode/utf8"

	"github.com/Maxime-Quesnel/groma/internal/claudecode"
	"github.com/Maxime-Quesnel/groma/internal/component"
	"github.com/Maxime-Quesnel/groma/internal/rule"
)

var descriptionTooLong = Rule{
	Meta: rule.Meta{
		ID:    "description-too-long",
		Level: rule.Warning,
		Title: "Description too long",
		Description: "Descriptions sit in Claude's context in every conversation, next to every other component's. The Agent Skills specification caps them at 1,024 characters, " +
			"and Claude Code cuts a skill's description and when_to_use at 1,536 characters in its listing, so Claude never reads the end.",
		Remediation: "Say what the component does and when to use it, key use case first, and move the rest into the body.",
		FalsePositives: []string{
			"An agent description over 1,024 characters that Claude Code still loads whole: the limit is the skills specification's, applied to agents for the same context cost.",
		},
		References: []string{claudecode.SkillsDocs, claudecode.BestPractices},
	},
	Kinds: markdown,
	Check: func(c *component.Component, t *component.Tree) []string {
		if !headerReadable(c) {
			return nil
		}
		description := utf8.RuneCountInString(c.Header.Value("description"))
		whenToUse := utf8.RuneCountInString(c.Header.Value("when_to_use"))
		switch {
		case c.Kind == component.Skill && description+whenToUse > 1536:
			return []string{fieldAt(c, "description", "description and when_to_use make %d characters; Claude Code cuts them at 1,536", description+whenToUse)}
		case description > 1024:
			return []string{fieldAt(c, "description", "%d characters, over the 1,024 limit", description)}
		}
		return nil
	},
}
