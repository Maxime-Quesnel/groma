package rules

import (
	"strings"

	"github.com/Maxime-Quesnel/groma/internal/claudecode"
	"github.com/Maxime-Quesnel/groma/internal/component"
	"github.com/Maxime-Quesnel/groma/internal/rule"
)

var reservedName = Rule{
	Meta: rule.Meta{
		ID:    "reserved-name",
		Level: rule.RedFlag,
		Title: "Name Claude Code reserves for synced skills",
		Description: "Outside a plugin, Claude Code doesn't load a skill folder or command file named anthropic-skills or starting with anthropic-skills:, " +
			"and skips a skill folder named synced in any capitalization. Both names belong to skills synced from the user's claude.ai account.",
		Remediation: "Rename the skill's folder or the command's file.",
		FalsePositives: []string{
			"None known. Plugin components are not affected and aren't flagged.",
		},
		References: []string{claudecode.SkillsDocs},
	},
	Kinds: skillsAndCommands,
	Check: func(c *component.Component, t *component.Tree) []string {
		if c.InPlugin {
			return nil
		}
		name := c.Name()
		if c.Kind == component.Skill {
			name = c.DirName()
			if strings.EqualFold(name, "synced") {
				return []string{"the skill's folder is named " + name}
			}
		}
		if name == "anthropic-skills" || strings.HasPrefix(name, "anthropic-skills:") {
			return []string{"named " + name}
		}
		return nil
	},
}
