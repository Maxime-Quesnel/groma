package rules

import (
	"fmt"
	"slices"

	"github.com/Maxime-Quesnel/groma/internal/claudecode"
	"github.com/Maxime-Quesnel/groma/internal/component"
	"github.com/Maxime-Quesnel/groma/internal/rule"
)

var unknownField = Rule{
	Meta: rule.Meta{
		ID:    "unknown-field",
		Level: rule.Warning,
		Title: "Field Claude Code's documentation doesn't list",
		Description: "Claude Code ignores frontmatter fields it doesn't know. A field such as version, author or tags does nothing, " +
			"and a reader of the file may take it for a setting that applies. The same goes for a hook field the hooks reference doesn't list.",
		Remediation: "In a skill or command, move your own data under metadata, the map Claude Code keeps for it. In an agent, remove the field. " +
			"In a hook, check the field's name against the hooks reference.",
		FalsePositives: []string{
			"A field added by a Claude Code release newer than groma's list of fields.",
			"A field read by your own tooling, which you choose to keep at the top level.",
		},
		References: []string{claudecode.SkillsDocs, claudecode.SubagentsDocs, claudecode.HooksDocs},
	},
	Kinds: everything,
	Check: func(c *component.Component, t *component.Tree) []string {
		if c.Kind == component.Hooks {
			return unknownHookFields(c)
		}
		if !headerReadable(c) {
			return nil
		}
		var evidence []string
		for _, f := range c.Header.Fields {
			if !slices.Contains(recognized(c.Kind), f.Key) && typoOf(c.Kind, f.Key) == "" {
				evidence = append(evidence, at(f.Line, "%s", f.Key))
			}
		}
		return evidence
	},
}

func unknownHookFields(c *component.Component) []string {
	var evidence []string
	for _, g := range c.Hooks.Groups {
		for _, key := range g.Keys {
			if key != "matcher" && key != "hooks" {
				evidence = append(evidence, fmt.Sprintf("%s: %s", g.Where(), key))
			}
		}
	}
	for _, h := range c.Hooks.Handlers {
		spec, known := claudecode.HandlerFields[h.Type]
		if !known {
			continue
		}
		for _, key := range h.Keys {
			if key != "matcher" && !slices.Contains(claudecode.CommonHandlerFields, key) && !slices.Contains(spec.Required, key) && !slices.Contains(spec.Optional, key) {
				evidence = append(evidence, fmt.Sprintf("%s: %s", h.Where(), key))
			}
		}
	}
	return evidence
}
