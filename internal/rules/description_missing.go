package rules

import (
	"strings"

	"github.com/Maxime-Quesnel/groma/internal/claudecode"
	"github.com/Maxime-Quesnel/groma/internal/component"
	"github.com/Maxime-Quesnel/groma/internal/rule"
)

var descriptionMissing = Rule{
	Meta: rule.Meta{
		ID:    "description-missing",
		Level: rule.Warning,
		Title: "No description",
		Description: "Claude picks skills and commands by their description. Without one, Claude Code falls back to the first line of the body, " +
			"which rarely says what the component does and when to use it, so Claude uses it at the wrong time or not at all.",
		Remediation: "Add a description that says what the component does and when to use it, in the third person.",
		FalsePositives: []string{
			"A command only its author invokes by name, with disable-model-invocation: true.",
		},
		References: []string{claudecode.SkillsDocs, claudecode.BestPractices},
	},
	Kinds: skillsAndCommands,
	Check: func(c *component.Component, t *component.Tree) []string {
		if !headerReadable(c) || strings.TrimSpace(c.Header.Value("description")+c.Header.Value("when_to_use")) != "" {
			return nil
		}
		first := ""
		for _, line := range strings.Split(c.Header.Body, "\n") {
			if first = strings.TrimSpace(line); first != "" {
				break
			}
		}
		if first == "" {
			return []string{"no description, and no text to fall back on"}
		}
		return []string{"no description, so Claude reads the first line instead: " + clip(first)}
	},
}
