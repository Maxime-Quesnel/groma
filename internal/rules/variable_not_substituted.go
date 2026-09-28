package rules

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/Maxime-Quesnel/groma/internal/claudecode"
	"github.com/Maxime-Quesnel/groma/internal/component"
	"github.com/Maxime-Quesnel/groma/internal/rule"
)

var variableNotSubstituted = Rule{
	Meta: rule.Meta{
		ID:    "variable-not-substituted",
		Level: rule.RedFlag,
		Title: "Placeholder Claude Code doesn't replace here",
		Description: "Claude Code replaces ${CLAUDE_PLUGIN_ROOT}, ${CLAUDE_PLUGIN_DATA} and ${CLAUDE_SKILL_DIR} in Markdown only in their braced form, the first two only in plugin components, " +
			"and ${CLAUDE_SKILL_DIR} only in skills; the variables aren't in the environment of commands Claude runs through the Bash tool either. " +
			"Elsewhere, and for $ARGUMENTS.0 or $IF(, which Claude Code doesn't have, Claude gets the text as written, and a command built on it runs with an empty or literal path.",
		Remediation: "Write ${CLAUDE_PLUGIN_ROOT} with braces in plugin components, ${CLAUDE_SKILL_DIR} only in skills, $ARGUMENTS[0] for the first argument, and plain instructions instead of $IF(.",
		FalsePositives: []string{
			"A skill about writing hooks that shows a hook command, where the variable is set. Fenced code blocks are skipped for the unbraced form.",
		},
		References: []string{claudecode.PluginManifestDocs, claudecode.SkillsDocs},
	},
	Kinds: everything,
	Check: func(c *component.Component, t *component.Tree) []string {
		var evidence []string
		if c.Kind == component.Hooks {
			for _, cmd := range commands(c) {
				if strings.Contains(cmd.run, "CLAUDE_SKILL_DIR") {
					evidence = append(evidence, fmt.Sprintf("%s uses CLAUDE_SKILL_DIR, which hooks don't get", cmd.where))
				}
			}
			return evidence
		}
		bodyLines(c, func(n int, line string) {
			if m := unbraced.FindString(line); m != "" {
				evidence = append(evidence, at(n, "%s is written without braces", strings.TrimSuffix(m, "/")))
			}
		})
		for i, line := range strings.Split(c.Header.Body, "\n") {
			n := c.Header.BodyLine + i
			switch {
			case !c.InPlugin && pluginVariable.MatchString(line):
				evidence = append(evidence, at(n, "%s is only replaced in plugin components", pluginVariable.FindString(line)))
			case c.Kind == component.Agent && strings.Contains(line, "${CLAUDE_SKILL_DIR}"):
				evidence = append(evidence, at(n, "${CLAUDE_SKILL_DIR} is only replaced in skills"))
			}
			if m := notClaudeCode.FindString(line); m != "" {
				evidence = append(evidence, at(n, "%s isn't Claude Code syntax", m))
			}
		}
		return evidence
	},
}

var (
	unbraced       = regexp.MustCompile(`\$CLAUDE_(?:PLUGIN_ROOT|PLUGIN_DATA|SKILL_DIR)/`)
	pluginVariable = regexp.MustCompile(`\$\{CLAUDE_PLUGIN_(?:ROOT|DATA)\}`)
	notClaudeCode  = regexp.MustCompile(`\$ARGUMENTS\.\d+|\$IF\(`)
)
