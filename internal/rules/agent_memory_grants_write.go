package rules

import (
	"slices"

	"github.com/Maxime-Quesnel/groma/internal/claudecode"
	"github.com/Maxime-Quesnel/groma/internal/component"
	"github.com/Maxime-Quesnel/groma/internal/rule"
)

var agentMemoryGrantsWrite = Rule{
	Meta: rule.Meta{
		ID:    "agent-memory-grants-write",
		Level: rule.Warning,
		Title: "Memory gives a read-only agent Write and Edit",
		Description: "Setting memory on an agent automatically enables Read, Write and Edit so it can keep its memory files. " +
			"An agent whose tools were limited to reading, such as a reviewer, can then write and edit files anywhere its permissions allow.",
		Remediation: "Remove memory from a read-only agent, or accept that it can write and review its prompt with that in mind.",
		FalsePositives: []string{
			"An agent trusted to write, whose tools list simply omits Write and Edit.",
		},
		References: []string{claudecode.SubagentsDocs},
	},
	Kinds: agents,
	Check: func(c *component.Component, t *component.Tree) []string {
		tools, restricted := c.Header.Field("tools")
		if !headerReadable(c) || !restricted || c.Header.Value("memory") == "" {
			return nil
		}
		names := make([]string, 0)
		for _, entry := range tools.Items() {
			names = append(names, claudecode.Tool(entry))
		}
		if slices.Contains(names, "Write") || slices.Contains(names, "Edit") {
			return nil
		}
		return []string{fieldAt(c, "memory", "memory: %s, on an agent whose tools don't include Write or Edit", c.Header.Value("memory"))}
	},
}
