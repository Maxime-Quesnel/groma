package rules

import (
	"strings"

	"github.com/Maxime-Quesnel/groma/internal/claudecode"
	"github.com/Maxime-Quesnel/groma/internal/component"
	"github.com/Maxime-Quesnel/groma/internal/rule"
)

var agentNotLoaded = Rule{
	Meta: rule.Meta{
		ID:    "agent-not-loaded",
		Level: rule.RedFlag,
		Title: "Claude Code skips this agent",
		Description: "Claude Code skips an agent file without a name, takes one with a name but no description for a broken agent and skips it too, " +
			"and refuses a name that starts with - or contains :, which is reserved for plugin-scoped names. " +
			"It reports none of this in the session, only in the debug log, so the agent silently never runs.",
		Remediation: "Give the agent a name in lowercase letters, digits and hyphens, and a description that says when Claude should delegate to it.",
		FalsePositives: []string{
			"A Markdown file with a frontmatter header kept in agents/ as documentation rather than as an agent.",
		},
		References: []string{claudecode.SubagentsDocs},
	},
	Kinds: agents,
	Check: func(c *component.Component, t *component.Tree) []string {
		if !headerReadable(c) || !c.Header.Found {
			return nil
		}
		name, named := c.Header.Field("name")
		switch {
		case !named || strings.TrimSpace(name.Value) == "":
			return []string{at(1, "no name, so Claude Code takes the file for documentation kept beside the agents")}
		case strings.HasPrefix(name.Value, "-") || strings.Contains(name.Value, ":"):
			return []string{at(name.Line, "name: %s starts with - or contains :, which Claude Code refuses", name.Value)}
		case strings.TrimSpace(c.Header.Value("description")) == "":
			return []string{at(name.Line, "a name but no description, so Claude Code skips the file")}
		}
		return nil
	},
}
