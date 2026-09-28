package rules

import (
	"slices"
	"strings"

	"github.com/Maxime-Quesnel/groma/internal/claudecode"
	"github.com/Maxime-Quesnel/groma/internal/component"
	"github.com/Maxime-Quesnel/groma/internal/rule"
)

var toolAllowedAndDenied = Rule{
	Meta: rule.Meta{
		ID:    "tool-allowed-and-denied",
		Level: rule.RedFlag,
		Title: "Tool both granted and removed",
		Description: "Claude Code applies an agent's disallowedTools first, then resolves tools against what's left, so a tool listed in both is removed. " +
			"The agent doesn't get a tool its tools field grants.",
		Remediation: "Remove the tool from one of the two lists.",
		FalsePositives: []string{
			"None known.",
		},
		References: []string{claudecode.SubagentsDocs},
	},
	Kinds: agents,
	Check: func(c *component.Component, t *component.Tree) []string {
		tools, granted := c.Header.Field("tools")
		denied, removed := c.Header.Field("disallowedTools")
		if !headerReadable(c) || !granted || !removed {
			return nil
		}
		var gone []string
		for _, entry := range denied.Items() {
			// An entry with a specifier is disallowed-tool-specifier's to report.
			if !strings.Contains(entry, "(") {
				gone = append(gone, entry)
			}
		}
		var evidence []string
		for _, entry := range tools.Items() {
			name := claudecode.Tool(entry)
			if slices.Contains(gone, name) || strings.HasPrefix(name, "mcp__") && slices.Contains(gone, "mcp__*") {
				evidence = append(evidence, at(tools.Line, "tools lists %s, which disallowedTools removes", name))
			}
		}
		return evidence
	},
}
