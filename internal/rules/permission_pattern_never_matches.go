package rules

import (
	"strings"

	"github.com/Maxime-Quesnel/groma/internal/claudecode"
	"github.com/Maxime-Quesnel/groma/internal/component"
	"github.com/Maxime-Quesnel/groma/internal/rule"
)

var permissionPatternNeverMatches = Rule{
	Meta: rule.Meta{
		ID:    "permission-pattern-never-matches",
		Level: rule.RedFlag,
		Title: "Permission pattern that never matches",
		Description: "The :* prefix form only works at the end of a pattern: in Bash(git:* push) the colon is a literal character and matches no git command. " +
			"And allowed-tools only takes tool-name globs after a literal mcp__<server>__ prefix: *, B* or mcp__* are skipped and approve nothing. " +
			"The skill then prompts, or aborts its inline shell, for what its author meant to pre-approve.",
		Remediation: "Put :* last, as in Bash(git push:*), or use the space form, Bash(git push *). Name tools exactly, or glob only after mcp__<server>__.",
		FalsePositives: []string{
			"None known.",
		},
		References: []string{claudecode.PermissionsDocs},
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
			for _, entry := range f.Items() {
				inner, specified := strings.CutPrefix(entry, claudecode.Tool(entry)+"(")
				switch {
				case specified && strings.Contains(strings.TrimSuffix(strings.TrimSuffix(inner, ")"), ":*"), ":*"):
					evidence = append(evidence, at(f.Line, "%s lists %s, where :* isn't at the end", key, entry))
				case key == "allowed-tools" && !specified && strings.Contains(entry, "*") && !anchoredGlob(entry):
					evidence = append(evidence, at(f.Line, "allowed-tools lists %s, a tool-name glob that approves nothing", entry))
				}
			}
		}
		return evidence
	},
}

// anchoredGlob reports whether a tool-name glob starts with a literal
// mcp__<server>__ prefix, the only form allowed-tools accepts.
func anchoredGlob(entry string) bool {
	server, rest, ok := strings.Cut(strings.TrimPrefix(entry, "mcp__"), "__")
	return strings.HasPrefix(entry, "mcp__") && ok && server != "" && !strings.Contains(server, "*") && rest != ""
}
