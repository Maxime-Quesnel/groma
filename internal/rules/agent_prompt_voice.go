package rules

import (
	"regexp"
	"strings"

	"github.com/Maxime-Quesnel/groma/internal/claudecode"
	"github.com/Maxime-Quesnel/groma/internal/component"
	"github.com/Maxime-Quesnel/groma/internal/rule"
)

var agentPromptVoice = Rule{
	Meta: rule.Meta{
		ID:    "agent-prompt-voice",
		Level: rule.Warning,
		Title: "Agent prompt written in the first person",
		Description: "An agent's body is its system prompt, addressed to the model that plays the agent. " +
			"Anthropic's agent guidance is to write it in the second person, \"You are a security reviewer…\", not \"I am…\" or \"I will…\".",
		Remediation: "Open the prompt with the agent's role in the second person: \"You are …\".",
		FalsePositives: []string{
			"A prompt that opens with a quoted example of what the agent might say.",
		},
		References: []string{claudecode.PluginDevAgents, claudecode.SubagentsDocs},
	},
	Kinds: agents,
	Check: func(c *component.Component, t *component.Tree) []string {
		var evidence []string
		done := false
		bodyLines(c, func(n int, line string) {
			line = strings.TrimSpace(line)
			if done || line == "" || strings.HasPrefix(line, "#") {
				return
			}
			done = true
			if firstPerson.MatchString(line) {
				evidence = append(evidence, at(n, "%s", clip(line)))
			}
		})
		return evidence
	},
}

var firstPerson = regexp.MustCompile(`(?i)^(?:I am|I'm|I will|I'll|I can|My (?:role|job|goal))\b`)
