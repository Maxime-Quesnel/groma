package rules

import (
	"fmt"
	"path"
	"strings"

	"github.com/Maxime-Quesnel/groma/internal/claudecode"
	"github.com/Maxime-Quesnel/groma/internal/component"
	"github.com/Maxime-Quesnel/groma/internal/rule"
)

var hookRelativeScript = Rule{
	Meta: rule.Meta{
		ID:    "hook-relative-script",
		Level: rule.RedFlag,
		Title: "Plugin hook runs its script by a relative path",
		Description: "Hooks run in the session's current directory, which is the user's project, not the plugin. A plugin hook that runs scripts/check.sh " +
			"or ./check.sh finds nothing there once the plugin is installed, and fails every time it fires.",
		Remediation: `Reach the plugin's files through its root: "${CLAUDE_PLUGIN_ROOT}/scripts/check.sh".`,
		FalsePositives: []string{
			"A hook meant to run a file of the user's project that happens to share a path with a file of the plugin.",
		},
		References: []string{claudecode.HooksDocs, claudecode.HooksGuide},
	},
	Kinds: hooks,
	Check: func(c *component.Component, t *component.Tree) []string {
		if !c.InPlugin || c.Plugin == "" {
			return nil
		}
		var evidence []string
		for _, cmd := range commands(c) {
			for _, word := range strings.Fields(strings.NewReplacer(`"`, " ", `'`, " ").Replace(cmd.run)) {
				if strings.ContainsAny(word[:1], "$/~-") || !strings.Contains(word, "/") && !strings.HasPrefix(word, ".") {
					continue
				}
				if _, ok := t.File(path.Join(c.Plugin, word)); ok {
					evidence = append(evidence, fmt.Sprintf("%s runs %s, which resolves against the user's project", cmd.where, word))
				}
			}
		}
		return evidence
	},
}
