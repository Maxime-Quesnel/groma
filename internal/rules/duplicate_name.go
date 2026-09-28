package rules

import (
	"fmt"

	"github.com/Maxime-Quesnel/groma/internal/claudecode"
	"github.com/Maxime-Quesnel/groma/internal/component"
	"github.com/Maxime-Quesnel/groma/internal/rule"
)

var duplicateName = Rule{
	Meta: rule.Meta{
		ID:    "duplicate-name",
		Level: rule.RedFlag,
		Title: "Two components share a name",
		Description: "When two agents of a project have the same name, or two agents of a plugin the same name in the same subfolder of agents/, " +
			"Claude Code loads only one of them, chosen by the order it reads files in. " +
			"A skill and a command, or two skills, with the same name both answer to the same slash command, so one hides the other.",
		Remediation: "Give each component its own name.",
		FalsePositives: []string{
			"Components of different plugins that groma checks together, such as a marketplace, when their plugins are never installed side by side.",
		},
		References: []string{claudecode.SubagentsDocs, claudecode.SkillsDocs},
	},
	Kinds: markdown,
	Check: func(c *component.Component, t *component.Tree) []string {
		var evidence []string
		for _, o := range t.Components {
			if o == c || o.Kind == component.Hooks || o.Kind == component.Plugin || (o.Kind == component.Agent) != (c.Kind == component.Agent) ||
				scope(o) != scope(c) || o.ID() != c.ID() {
				continue
			}
			if c.Kind == component.Agent {
				evidence = append(evidence, fmt.Sprintf("%s is also named %s; Claude Code loads only one of them", o.Path, c.Name()))
			} else {
				evidence = append(evidence, fmt.Sprintf("%s also answers to /%s", o.Path, c.Name()))
			}
		}
		return evidence
	},
}

// scope is where a component's name must be unique: its plugin, or its
// project's .claude directory.
func scope(c *component.Component) string {
	if c.InPlugin {
		return "plugin " + c.Plugin
	}
	return "project " + c.Project
}
