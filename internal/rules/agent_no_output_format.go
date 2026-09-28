package rules

import (
	"regexp"
	"strings"

	"github.com/Maxime-Quesnel/groma/internal/claudecode"
	"github.com/Maxime-Quesnel/groma/internal/component"
	"github.com/Maxime-Quesnel/groma/internal/rule"
)

var agentNoOutputFormat = Rule{
	Meta: rule.Meta{
		ID:    "agent-no-output-format",
		Level: rule.Warning,
		Title: "Agent prompt never says what to return",
		Description: "A subagent works in its own context and hands only its final message back to the conversation that called it. " +
			"Claude Code's examples specify \"exactly what to look for and how to format output\", and Anthropic's agent guidance warns against leaving the output undefined: " +
			"without it, the caller gets whatever shape the agent chooses.",
		Remediation: "End the prompt with what to return and in what form, such as \"Return the findings as a list: file, line, problem, fix\".",
		FalsePositives: []string{
			"A prompt that describes its output without the words return, report, output, respond, format, summary, result or deliver.",
		},
		References: []string{claudecode.SubagentsDocs, claudecode.PluginDevAgents},
	},
	Kinds: agents,
	Check: func(c *component.Component, t *component.Tree) []string {
		body := strings.TrimSpace(c.Header.Body)
		if !headerReadable(c) || body == "" || outputWords.MatchString(body) {
			return nil
		}
		return []string{at(c.Header.BodyLine, "the prompt never mentions what to return or report")}
	},
}

var outputWords = regexp.MustCompile(`(?i)\b(?:return|report|output|respond|response|format|summar|result|deliver|reply|hand back)`)
