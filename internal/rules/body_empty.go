package rules

import (
	"strings"

	"github.com/Maxime-Quesnel/groma/internal/claudecode"
	"github.com/Maxime-Quesnel/groma/internal/component"
	"github.com/Maxime-Quesnel/groma/internal/rule"
)

var bodyEmpty = Rule{
	Meta: rule.Meta{
		ID:    "body-empty",
		Level: rule.Warning,
		Title: "Nothing after the frontmatter",
		Description: "A skill's or command's body is what Claude follows once it's invoked, and an agent's body is its system prompt: " +
			"subagents get only that prompt and basic environment details, not Claude Code's system prompt. An empty body leaves Claude with a name and a description to go on.",
		Remediation: "Write the instructions: for an agent, its role, how it works and what it returns; for a skill or command, the steps to follow.",
		FalsePositives: []string{
			"A placeholder kept on purpose while the component is being written.",
		},
		References: []string{claudecode.SubagentsDocs, claudecode.SkillsDocs},
	},
	Kinds: markdown,
	Check: func(c *component.Component, t *component.Tree) []string {
		if !headerReadable(c) || !c.Header.Found || strings.TrimSpace(c.Header.Body) != "" {
			return nil
		}
		return []string{at(c.Header.BodyLine-1, "the file ends with its frontmatter")}
	},
}
