package rules

import (
	"slices"

	"github.com/Maxime-Quesnel/groma/internal/claudecode"
	"github.com/Maxime-Quesnel/groma/internal/component"
	"github.com/Maxime-Quesnel/groma/internal/rule"
)

var toolUnavailableToAgents = Rule{
	Meta: rule.Meta{
		ID:    "tool-unavailable-to-agents",
		Level: rule.RedFlag,
		Title: "Agent lists a tool no subagent gets",
		Description: "Claude Code removes AskUserQuestion, EndConversation, EnterPlanMode, ScheduleWakeup, WaitForMcpServers and Workflow from every subagent, " +
			"and ExitPlanMode from any not in plan mode, even when its tools field lists them. An agent written to ask the user or to plan cannot, " +
			"and its prompt steers it towards a step that fails.",
		Remediation: "Remove the tool, and have the agent return its question or plan to the main conversation instead.",
		FalsePositives: []string{
			"An agent file also run as the main session agent, with --agent, where these tools stay available.",
		},
		References: []string{claudecode.SubagentsDocs},
	},
	Kinds: agents,
	Check: func(c *component.Component, t *component.Tree) []string {
		f, ok := c.Header.Field("tools")
		if !headerReadable(c) || !ok {
			return nil
		}
		var evidence, seen []string
		for _, entry := range f.Items() {
			name := claudecode.Tool(entry)
			if slices.Contains(seen, name) {
				continue
			}
			seen = append(seen, name)
			if slices.Contains(claudecode.UnavailableToSubagents, name) ||
				name == "ExitPlanMode" && c.Header.Value("permissionMode") != "plan" {
				evidence = append(evidence, at(f.Line, "tools lists %s, which Claude Code never gives this subagent", name))
			}
		}
		return evidence
	},
}
