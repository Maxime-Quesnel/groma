package rules

import (
	"github.com/Maxime-Quesnel/groma/internal/claudecode"
	"github.com/Maxime-Quesnel/groma/internal/component"
	"github.com/Maxime-Quesnel/groma/internal/rule"
)

var fieldIgnored = Rule{
	Meta: rule.Meta{
		ID:    "field-ignored",
		Level: rule.RedFlag,
		Title: "Frontmatter field Claude Code ignores here",
		Description: "The field is real, but not where it stands. Claude Code ignores hooks, mcpServers and permissionMode on an agent that ships in a plugin, " +
			"name and paths on a command, and agent and background on a skill that doesn't run with context: fork. " +
			"A permissionMode: plan or a guard hook that never applies leaves the component with more power than its author gave it.",
		Remediation: "Move the setting where Claude Code reads it: plugin hooks go in hooks/hooks.json, plugin MCP servers in .mcp.json, and a command " +
			"that needs a name or paths becomes a skill. For an agent's permissions, restrict its tools instead of its permission mode.",
		FalsePositives: []string{
			"An agent file shared between a plugin and a .claude/agents directory, where the fields do apply.",
		},
		References: []string{claudecode.SubagentsDocs, claudecode.SkillsDocs},
	},
	Kinds: markdown,
	Check: func(c *component.Component, t *component.Tree) []string {
		if !headerReadable(c) {
			return nil
		}
		var evidence []string
		ignored := func(key, why string) {
			if c.Header.Has(key) {
				evidence = append(evidence, fieldAt(c, key, "%s %s", key, why))
			}
		}
		switch c.Kind {
		case component.Agent:
			if c.InPlugin {
				for _, key := range claudecode.PluginIgnoredAgentFields {
					ignored(key, "is ignored on an agent that ships in a plugin")
				}
			}
		case component.Command:
			ignored("name", "is ignored: a command takes its name from its file")
			ignored("paths", "only works on skills")
		}
		if c.Kind != component.Agent && c.Header.Value("context") != "fork" {
			ignored("agent", "only applies with context: fork")
			ignored("background", "only applies with context: fork")
		}
		return evidence
	},
}
