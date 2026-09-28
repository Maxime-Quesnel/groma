package rules

import (
	"fmt"
	"strings"

	"github.com/Maxime-Quesnel/groma/internal/claudecode"
	"github.com/Maxime-Quesnel/groma/internal/component"
	"github.com/Maxime-Quesnel/groma/internal/rule"
)

var hookUserConfigInShell = Rule{
	Meta: rule.Meta{
		ID:    "hook-user-config-in-shell",
		Level: rule.RedFlag,
		Title: "Plugin option substituted into a shell command",
		Description: "Claude Code refuses ${user_config.…} in a shell-form hook command, since the user's option value would be parsed by the shell, " +
			"and the hook fails with an error every time it fires.",
		Remediation: "Use the exec form, with the option in args, or read the $CLAUDE_PLUGIN_OPTION_<KEY> environment variable inside the script.",
		FalsePositives: []string{
			"None known.",
		},
		References: []string{claudecode.HooksDocs},
	},
	Kinds: hooks,
	Check: func(c *component.Component, t *component.Tree) []string {
		var evidence []string
		for _, h := range c.Hooks.Handlers {
			if !h.Exec && strings.Contains(h.Command, "${user_config.") {
				evidence = append(evidence, fmt.Sprintf("%s runs %s", h.Where(), clip(h.Command)))
			}
		}
		return evidence
	},
}
