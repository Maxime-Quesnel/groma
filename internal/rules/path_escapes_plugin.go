package rules

import (
	"fmt"
	"path"

	"github.com/Maxime-Quesnel/groma/internal/claudecode"
	"github.com/Maxime-Quesnel/groma/internal/component"
	"github.com/Maxime-Quesnel/groma/internal/rule"
)

var pathEscapesPlugin = Rule{
	Meta: rule.Meta{
		ID:    "path-escapes-plugin",
		Level: rule.RedFlag,
		Title: "Path that leaves the plugin",
		Description: "Claude Code copies an installed plugin into its cache and loads it from there, so a link or a ${CLAUDE_PLUGIN_ROOT}/.. path " +
			"that reaches outside the plugin's directory points at nothing once the plugin is installed. It works while the author tests from the repository, and breaks for everyone else.",
		Remediation: "Move the file inside the plugin, or link it through a symlink inside the plugin.",
		FalsePositives: []string{
			"A link meant for humans reading the repository, such as one to a contributing guide, that Claude never follows.",
		},
		References: []string{claudecode.PluginTroubleshooting},
	},
	Kinds: []component.Kind{component.Skill, component.Agent, component.Command, component.Hooks},
	Check: func(c *component.Component, t *component.Tree) []string {
		if !c.InPlugin || c.Plugin == "" {
			return nil
		}
		var evidence []string
		if c.Kind == component.Hooks {
			for _, cmd := range commands(c) {
				for _, s := range t.Scripts(cmd.run, c.Roots()) {
					if s.Path != "" && !within(s.Path, c.Plugin) {
						evidence = append(evidence, fmt.Sprintf("%s runs %s", cmd.where, s.Ref))
					}
				}
			}
			return evidence
		}
		for _, f := range markdownFiles(c, t) {
			for _, l := range links(c, t, f) {
				if !within(path.Clean(l.file), c.Plugin) {
					evidence = append(evidence, in(c, f, at(l.line, "%s", l.target)))
				}
			}
		}
		return evidence
	},
}
