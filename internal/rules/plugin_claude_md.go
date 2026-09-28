package rules

import (
	"path"

	"github.com/Maxime-Quesnel/groma/internal/claudecode"
	"github.com/Maxime-Quesnel/groma/internal/component"
	"github.com/Maxime-Quesnel/groma/internal/rule"
)

var pluginClaudeMd = Rule{
	Meta: rule.Meta{
		ID:    "plugin-claude-md",
		Level: rule.RedFlag,
		Title: "CLAUDE.md at the plugin's root, which Claude Code never loads",
		Description: "A CLAUDE.md in a project gives Claude standing instructions, but Claude Code doesn't load one at a plugin's root. " +
			"The instructions it holds never reach Claude.",
		Remediation: "Write the instructions as a skill, whose description says when they apply.",
		FalsePositives: []string{
			"A CLAUDE.md kept for developing the plugin itself, when the plugin's root is also the repository's root and the author opens Claude Code there.",
		},
		References: []string{claudecode.PluginComponentsDocs},
	},
	Kinds: []component.Kind{component.Plugin},
	Check: func(c *component.Component, t *component.Tree) []string {
		if _, ok := t.File(path.Join(c.Dir, "CLAUDE.md")); ok {
			return []string{path.Join(c.Dir, "CLAUDE.md") + " sits at the plugin's root"}
		}
		return nil
	},
}
