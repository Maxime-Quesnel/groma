package rules

import (
	"fmt"

	"github.com/Maxime-Quesnel/groma/internal/claudecode"
	"github.com/Maxime-Quesnel/groma/internal/component"
	"github.com/Maxime-Quesnel/groma/internal/rule"
)

var unknownTool = Rule{
	Meta: rule.Meta{
		ID:    "unknown-tool",
		Level: rule.RedFlag,
		Title: "Tool Claude Code doesn't have",
		Description: "A tool listed in tools, disallowedTools, allowed-tools or disallowed-tools matches no Claude Code tool: a typo, a lowercase name, " +
			"or a tool Claude Code has since renamed or removed, such as MultiEdit. Claude Code drops the entry silently, so a restriction doesn't apply " +
			"or a tool the component relies on isn't granted. When no entry of an agent's tools resolves, Claude Code refuses to launch the agent.",
		Remediation: "Use the tool's exact name, as the Claude Code tools reference spells it.",
		FalsePositives: []string{
			"A tool added by a Claude Code release newer than groma's list. MCP tools, named mcp__<server>__<tool>, are never flagged.",
		},
		References: []string{claudecode.ToolsDocs},
	},
	Kinds: markdown,
	Check: func(c *component.Component, t *component.Tree) []string {
		if !headerReadable(c) {
			return nil
		}
		keys := []string{"allowed-tools", "disallowed-tools"}
		if c.Kind == component.Agent {
			keys = []string{"tools", "disallowedTools"}
		}
		var evidence []string
		for _, key := range keys {
			f, ok := c.Header.Field(key)
			if !ok {
				continue
			}
			entries, unknown := f.Items(), 0
			for _, entry := range entries {
				name := claudecode.Tool(entry)
				if name == "" || claudecode.KnownTool(name) {
					continue
				}
				unknown++
				evidence = append(evidence, at(f.Line, "%s lists %s, %s", key, name, whyUnknown(name)))
			}
			if key == "tools" && unknown > 0 && unknown == len(entries) {
				evidence = append(evidence, at(f.Line, "no entry of tools resolves, so Claude Code refuses to launch the agent"))
			}
		}
		return evidence
	},
}

func whyUnknown(name string) string {
	if replaced, ok := claudecode.FormerTools[name]; ok {
		return fmt.Sprintf("which Claude Code no longer has; use %s", replaced)
	}
	if s := suggest(name, claudecode.Tools); s != "" {
		return fmt.Sprintf("which isn't a tool; did you mean %s?", s)
	}
	return "which no Claude Code tool is called"
}
