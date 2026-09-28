package rules

import (
	"encoding/json"
	"fmt"

	"github.com/Maxime-Quesnel/groma/internal/claudecode"
	"github.com/Maxime-Quesnel/groma/internal/component"
	"github.com/Maxime-Quesnel/groma/internal/rule"
)

var fieldWithoutEffect = Rule{
	Meta: rule.Meta{
		ID:    "field-without-effect",
		Level: rule.Warning,
		Title: "Setting that has no effect as the component is written",
		Description: "when_to_use and paths only steer Claude's choice, and a skill with disable-model-invocation: true is never Claude's to choose: " +
			"its description isn't even in context. And Claude Code doesn't enforce a timeout on an async hook, which runs in the background.",
		Remediation: "Remove the setting, or drop what makes it moot.",
		FalsePositives: []string{
			"A setting kept on purpose for when the other one changes.",
		},
		References: []string{claudecode.SkillsDocs, claudecode.HooksDocs},
	},
	Kinds: []component.Kind{component.Skill, component.Command, component.Hooks},
	Check: func(c *component.Component, t *component.Tree) []string {
		var evidence []string
		if c.Kind == component.Hooks {
			for _, h := range c.Hooks.Handlers {
				var async bool
				json.Unmarshal(h.Fields["async"], &async)
				if _, timed := h.Fields["timeout"]; async && timed {
					evidence = append(evidence, fmt.Sprintf("%s sets a timeout on an async hook", h.Where()))
				}
			}
			return evidence
		}
		if !headerReadable(c) || !isTrue(c.Header.Value("disable-model-invocation")) {
			return nil
		}
		for _, key := range []string{"when_to_use", "paths"} {
			if c.Header.Has(key) {
				evidence = append(evidence, fieldAt(c, key, "%s, on a skill Claude can't invoke", key))
			}
		}
		return evidence
	},
}
