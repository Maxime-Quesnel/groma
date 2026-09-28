package rules

import (
	"fmt"

	"github.com/Maxime-Quesnel/groma/internal/claudecode"
	"github.com/Maxime-Quesnel/groma/internal/component"
	"github.com/Maxime-Quesnel/groma/internal/rule"
)

var missingScript = Rule{
	Meta: rule.Meta{
		ID:    "missing-script",
		Level: rule.RedFlag,
		Title: "Runs a script that isn't there or can't run",
		Description: "A hook or an inline shell command runs a file of the plugin or project through ${CLAUDE_PLUGIN_ROOT}, ${CLAUDE_PROJECT_DIR} or ${CLAUDE_SKILL_DIR}, " +
			"and the file doesn't exist, or it runs the file directly and the file isn't executable. The hook fails every time it fires, " +
			"with a non-blocking error the user rarely sees, and a skill's invocation aborts.",
		Remediation: "Ship the script at that path and make it executable (chmod +x, then commit the mode), or call it through its interpreter, such as bash or python3.",
		FalsePositives: []string{
			"A script the plugin generates at install or first run.",
			"A file whose executable bit git lost on a system that doesn't track it, such as a checkout on Windows.",
		},
		References: []string{claudecode.HooksDocs},
	},
	Kinds: []component.Kind{component.Hooks, component.Skill, component.Command},
	Check: func(c *component.Component, t *component.Tree) []string {
		var evidence []string
		for _, cmd := range commands(c) {
			for _, s := range t.Scripts(cmd.run, c.Roots()) {
				f, isFile := t.File(s.Path)
				switch {
				case s.Path == "":
				case !s.Found:
					evidence = append(evidence, fmt.Sprintf("%s runs %s, which doesn't exist", cmd.where, s.Ref))
				case isFile && s.Direct && !f.Executable:
					evidence = append(evidence, fmt.Sprintf("%s runs %s directly, but the file isn't executable", cmd.where, s.Ref))
				}
			}
		}
		return evidence
	},
}
