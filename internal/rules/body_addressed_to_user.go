package rules

import (
	"regexp"
	"strings"

	"github.com/Maxime-Quesnel/groma/internal/claudecode"
	"github.com/Maxime-Quesnel/groma/internal/component"
	"github.com/Maxime-Quesnel/groma/internal/rule"
)

var bodyAddressedToUser = Rule{
	Meta: rule.Meta{
		ID:    "body-addressed-to-user",
		Level: rule.Warning,
		Title: "Body written to the user instead of Claude",
		Description: "The body of a command or skill is the prompt Claude receives when it's invoked. One that opens with \"This command will…\" " +
			"describes the command to a reader instead of telling Claude what to do; Anthropic's command guidance is to write instructions for Claude.",
		Remediation: "Open with what Claude should do: \"Review the staged changes for …\", not \"This command will review …\".",
		FalsePositives: []string{
			"A body that opens with a sentence of context before its instructions.",
		},
		References: []string{claudecode.PluginDevCommands},
	},
	Kinds: skillsAndCommands,
	Check: func(c *component.Component, t *component.Tree) []string {
		var evidence []string
		done := false
		bodyLines(c, func(n int, line string) {
			line = strings.TrimSpace(line)
			if done || line == "" || strings.HasPrefix(line, "#") {
				return
			}
			done = true
			if m := toTheUser.FindString(line); m != "" {
				evidence = append(evidence, at(n, "%s", clip(line)))
			}
		})
		return evidence
	},
}

var toTheUser = regexp.MustCompile(`(?i)^(?:this (?:command|skill) (?:will|lets you|helps you|allows you)|you(?:'ll| will) (?:get|receive|see))\b`)
