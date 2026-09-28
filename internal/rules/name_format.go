package rules

import (
	"regexp"
	"strings"

	"github.com/Maxime-Quesnel/groma/internal/claudecode"
	"github.com/Maxime-Quesnel/groma/internal/component"
	"github.com/Maxime-Quesnel/groma/internal/rule"
)

var nameFormat = Rule{
	Meta: rule.Meta{
		ID:    "name-format",
		Level: rule.Warning,
		Title: "Name isn't lowercase with hyphens",
		Description: "Skill, agent and command names are typed as slash commands and read by Claude when it picks one. " +
			"The Agent Skills specification and Claude Code's documentation use lowercase letters, digits and single hyphens, up to 64 characters; " +
			"spaces break the slash command, and other characters make names hard to type and to tell apart.",
		Remediation: "Rename it in lowercase with hyphens, such as pdf-processing or code-reviewer.",
		FalsePositives: []string{
			"A name kept for compatibility with commands users already type.",
		},
		References: []string{claudecode.BestPractices, claudecode.SkillsDocs},
	},
	Kinds: markdown,
	Check: func(c *component.Component, t *component.Tree) []string {
		if !headerReadable(c) {
			return nil
		}
		name := c.Name()
		// A plugin skill's name may carry the plugin's prefix, and a
		// command's holds the directories it sits in.
		segments := strings.Split(name, ":")
		if c.Kind == component.Agent {
			if strings.HasPrefix(name, "-") || strings.Contains(name, ":") {
				return nil
			}
			segments = []string{name}
		}
		for _, s := range segments {
			if len(name) > 64 || !kebab.MatchString(s) {
				if f, ok := c.Header.Field("name"); ok && c.Kind != component.Command {
					return []string{at(f.Line, "name: %s", f.Value)}
				}
				return []string{"named " + name + " after its " + origin(c)}
			}
		}
		return nil
	},
}

var kebab = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

func origin(c *component.Component) string {
	if c.Kind == component.Skill {
		return "directory"
	}
	return "file"
}
