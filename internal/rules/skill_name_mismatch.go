package rules

import (
	"strings"

	"github.com/Maxime-Quesnel/groma/internal/claudecode"
	"github.com/Maxime-Quesnel/groma/internal/component"
	"github.com/Maxime-Quesnel/groma/internal/rule"
)

var skillNameMismatch = Rule{
	Meta: rule.Meta{
		ID:    "skill-name-mismatch",
		Level: rule.Warning,
		Title: "Skill named differently from its folder",
		Description: "The Agent Skills specification requires a skill's name to match its folder. In a plugin, Claude Code uses the name for the skill's command, " +
			"so references written with the folder's name, in other skills, commands or docs, point at nothing, and readers look for the skill in the wrong place.",
		Remediation: "Rename the folder to the skill's name, or the name to the folder's.",
		FalsePositives: []string{
			"A folder with an ordering prefix, such as 01-, kept on purpose while the name drops it.",
		},
		References: []string{"https://agentskills.io/specification", claudecode.SkillsDocs},
	},
	Kinds: skills,
	Check: func(c *component.Component, t *component.Tree) []string {
		name := c.Header.Value("name")
		// A skill at a plugin's root has no folder of its own.
		if !headerReadable(c) || name == "" || c.InPlugin && c.Dir == c.Plugin {
			return nil
		}
		if _, last, found := strings.Cut(name, ":"); found {
			name = last
		}
		if name == c.DirName() {
			return nil
		}
		return []string{fieldAt(c, "name", "name: %s, in the folder %s", c.Header.Value("name"), c.DirName())}
	},
}
