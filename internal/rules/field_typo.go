package rules

import (
	"slices"
	"strings"

	"github.com/Maxime-Quesnel/groma/internal/claudecode"
	"github.com/Maxime-Quesnel/groma/internal/component"
	"github.com/Maxime-Quesnel/groma/internal/rule"
)

var fieldTypo = Rule{
	Meta: rule.Meta{
		ID:    "field-typo",
		Level: rule.RedFlag,
		Title: "Misspelled frontmatter field",
		Description: "Claude Code ignores frontmatter fields it doesn't know, without a word. A field that is one letter, a dash or a case away from a real one, " +
			"such as allowed_tools or allowedTools for allowed-tools, or tools in a skill, is almost always meant as the real one: " +
			"the setting it carries, such as the tools the component may use, silently doesn't apply.",
		Remediation: "Rename the field to the one Claude Code reads.",
		FalsePositives: []string{
			"A field of your own that happens to sit close to a Claude Code field. In a skill, keep your own data under metadata instead.",
		},
		References: []string{claudecode.SkillsDocs, claudecode.SubagentsDocs},
	},
	Kinds: markdown,
	Check: func(c *component.Component, t *component.Tree) []string {
		if !headerReadable(c) {
			return nil
		}
		var evidence []string
		for _, f := range c.Header.Fields {
			if meant := typoOf(c.Kind, f.Key); meant != "" {
				evidence = append(evidence, at(f.Line, "%s looks like %s; as written, Claude Code ignores it", f.Key, meant))
			}
		}
		return evidence
	},
}

// recognized lists the fields Claude Code reads for a kind of component.
// A command's name and paths are known but ignored: field-ignored reports
// them.
func recognized(k component.Kind) []string {
	if k == component.Agent {
		return claudecode.AgentFields
	}
	return claudecode.SkillFields
}

// typoOf returns the field key seems meant to be, or "" when key is a field
// Claude Code reads or nothing like one.
func typoOf(k component.Kind, key string) string {
	known := recognized(k)
	if slices.Contains(known, key) {
		return ""
	}
	norm := strings.ToLower(strings.NewReplacer("-", "", "_", "").Replace(key))
	switch {
	case k == component.Agent && norm == "allowedtools":
		return "tools"
	case k != component.Agent && norm == "tools":
		return "allowed-tools"
	}
	return suggest(key, known)
}
