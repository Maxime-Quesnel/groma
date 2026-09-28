package rules

import (
	"strings"

	"github.com/Maxime-Quesnel/groma/internal/claudecode"
	"github.com/Maxime-Quesnel/groma/internal/component"
	"github.com/Maxime-Quesnel/groma/internal/rule"
)

var disallowedToolSpecifier = Rule{
	Meta: rule.Meta{
		ID:    "disallowed-tool-specifier",
		Level: rule.RedFlag,
		Title: "disallowedTools entry that removes the whole tool",
		Description: "An agent's disallowedTools entry with a specifier, such as Bash(git push *), doesn't block only the matching commands: " +
			"it removes the whole tool from the agent. An agent meant to run tests but never push can't run anything in the shell.",
		Remediation: "List the bare tool in disallowedTools only to remove it entirely. To keep Bash and block some commands, add a deny rule such as Bash(git push *) to permissions.deny in settings.",
		FalsePositives: []string{
			"None known: the specifier never narrows the removal.",
		},
		References: []string{claudecode.SubagentsDocs},
	},
	Kinds: agents,
	Check: func(c *component.Component, t *component.Tree) []string {
		f, ok := c.Header.Field("disallowedTools")
		if !headerReadable(c) || !ok {
			return nil
		}
		var evidence []string
		for _, entry := range f.Items() {
			if strings.Contains(entry, "(") {
				evidence = append(evidence, at(f.Line, "disallowedTools lists %s, which removes all of %s", entry, claudecode.Tool(entry)))
			}
		}
		return evidence
	},
}
