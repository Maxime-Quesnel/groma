package rules

import (
	"github.com/Maxime-Quesnel/groma/internal/claudecode"
	"github.com/Maxime-Quesnel/groma/internal/component"
	"github.com/Maxime-Quesnel/groma/internal/rule"
)

var agentToolsUnrestricted = Rule{
	Meta: rule.Meta{
		ID:    "agent-tools-unrestricted",
		Level: rule.Warning,
		Title: "Agent can use every tool",
		Description: "Without tools or disallowedTools, the agent inherits every tool of the session: the shell, file writes, web requests and every MCP server. " +
			"Claude Code's documentation recommends giving each subagent only the tools its job needs, which limits what a mistake or an injected instruction can do.",
		Remediation: "List the tools the agent needs in tools, such as tools: Read, Grep, Glob for a reviewer, or deny the ones it must not use in disallowedTools.",
		FalsePositives: []string{
			"A general-purpose agent that genuinely needs the whole toolset.",
		},
		References: []string{claudecode.SubagentsDocs},
	},
	Kinds: agents,
	Check: func(c *component.Component, t *component.Tree) []string {
		if !headerReadable(c) || !c.Header.Found || c.Header.Has("tools") || c.Header.Has("disallowedTools") {
			return nil
		}
		return []string{at(1, "no tools or disallowedTools field")}
	},
}
