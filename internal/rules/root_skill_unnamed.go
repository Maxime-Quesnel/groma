package rules

import (
	"github.com/Maxime-Quesnel/groma/internal/claudecode"
	"github.com/Maxime-Quesnel/groma/internal/component"
	"github.com/Maxime-Quesnel/groma/internal/rule"
)

var rootSkillUnnamed = Rule{
	Meta: rule.Meta{
		ID:    "root-skill-unnamed",
		Level: rule.Warning,
		Title: "Skill at the plugin's root without a name",
		Description: "A SKILL.md at a plugin's root has no folder of its own to take a name from. Without a name field, a marketplace install names the skill " +
			"after its cache directory, so its command differs from one installation to the next.",
		Remediation: "Set name in the skill's frontmatter.",
		FalsePositives: []string{
			"None known.",
		},
		References: []string{claudecode.PluginComponentsDocs},
	},
	Kinds: skills,
	Check: func(c *component.Component, t *component.Tree) []string {
		if !headerReadable(c) || !c.InPlugin || c.Dir != c.Plugin || c.Header.Value("name") != "" {
			return nil
		}
		return []string{"SKILL.md sits at the plugin's root and sets no name"}
	},
}
