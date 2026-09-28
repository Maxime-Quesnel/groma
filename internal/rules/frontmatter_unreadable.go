package rules

import (
	"strings"

	"github.com/Maxime-Quesnel/groma/internal/claudecode"
	"github.com/Maxime-Quesnel/groma/internal/component"
	"github.com/Maxime-Quesnel/groma/internal/rule"
)

var frontmatterUnreadable = Rule{
	Meta: rule.Meta{
		ID:    "frontmatter-unreadable",
		Level: rule.RedFlag,
		Title: "Claude Code can't read the frontmatter",
		Description: "When the YAML header doesn't parse, or doesn't open on line 1, Claude Code loads the component with none of its fields: " +
			"no name, no description to route on, no allowed-tools, no model. Nothing in the session says so.",
		Remediation: "Open the file with --- on line 1 and close the header with ---. Indent with spaces, write each field as key: value, " +
			"and quote a value that starts with @, `, *, !, [ or {, or that holds a colon followed by a space.",
		FalsePositives: []string{
			"groma follows YAML 1.2 on what it flags; a construct at the edge of the spec may pass or fail differently in Claude Code's parser.",
		},
		References: []string{claudecode.SkillsDocs, claudecode.SubagentsDocs},
	},
	Kinds: markdown,
	Check: func(c *component.Component, t *component.Tree) []string {
		var evidence []string
		if c.Header.Misplaced {
			for i, line := range strings.Split(string(c.Content), "\n") {
				if strings.TrimSpace(line) != "" {
					evidence = append(evidence, at(i+1, "the --- that opens the header isn't on line 1, so Claude Code reads the header as text"))
					break
				}
			}
		}
		for _, p := range c.Header.Problems {
			evidence = append(evidence, at(p.Line, "%s", p.Text))
		}
		return evidence
	},
}
