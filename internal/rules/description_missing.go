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
		Description: "Claude picks skills and commands by their description, and decides to delegate to an agent by its description. " +
			"Without one, Claude Code falls back to the first line of a skill's body, which rarely says what it does and when to use it, " +
			"and a plugin agent gets a stock description; Claude uses them at the wrong time or not at all.",
		Remediation: "Add a description that says what the component does and when to use it, in the third person.",
		FalsePositives: []string{
			"A command only its author invokes by name, with disable-model-invocation: true.",
		},
		References: []string{claudecode.SkillsDocs, claudecode.BestPractices},
	},
	Kinds: markdown,
	Check: func(c *component.Component, t *component.Tree) []string {
		// Outside a plugin, an agent without a description isn't loaded:
		// agent-not-loaded reports it.
		if !headerReadable(c) || strings.TrimSpace(c.Header.Value("description")+c.Header.Value("when_to_use")) != "" ||
			c.Kind == component.Agent && !c.InPlugin {
			return nil
		}
		if c.Kind == component.Agent {
			return []string{"no description, so Claude has nothing to decide when to delegate on"}
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
