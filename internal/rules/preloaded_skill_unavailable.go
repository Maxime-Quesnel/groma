package rules

import (
	"strings"

	"github.com/Maxime-Quesnel/groma/internal/claudecode"
	"github.com/Maxime-Quesnel/groma/internal/component"
	"github.com/Maxime-Quesnel/groma/internal/rule"
)

var preloadedSkillUnavailable = Rule{
	Meta: rule.Meta{
		ID:    "preloaded-skill-unavailable",
		Level: rule.RedFlag,
		Title: "Agent preloads a skill it can't get",
		Description: "An agent's skills field preloads each skill's full content at startup, but only from the skills Claude can invoke: " +
			"a skill with disable-model-invocation: true can't be preloaded, and neither can one that doesn't exist. " +
			"Claude Code skips it and only logs a warning to the debug log, so the agent runs without the knowledge its prompt relies on.",
		Remediation: "Drop disable-model-invocation from the skill, or remove it from the agent's skills and have the agent read it another way. Fix the name of a skill that doesn't exist.",
		FalsePositives: []string{
			"A skill from another plugin or the user's own skills, which groma doesn't see. Only names prefixed with the agent's own plugin are checked for existence.",
		},
		References: []string{claudecode.SubagentsDocs, claudecode.SkillsDocs},
	},
	Kinds: agents,
	Check: func(c *component.Component, t *component.Tree) []string {
		f, ok := c.Header.Field("skills")
		if !headerReadable(c) || !ok {
			return nil
		}
		var evidence []string
		for _, name := range f.Items() {
			bare, prefixed := strings.CutPrefix(name, c.PluginName+":")
			prefixed = prefixed && c.PluginName != ""
			var found *component.Component
			for _, s := range t.Components {
				if s.Kind == component.Skill && scope(s) == scope(c) && s.Name() == bare {
					found = s
				}
			}
			switch {
			case found != nil && isTrue(found.Header.Value("disable-model-invocation")):
				evidence = append(evidence, at(f.Line, "skills lists %s, which sets disable-model-invocation: true", name))
			case found == nil && prefixed:
				evidence = append(evidence, at(f.Line, "skills lists %s, which %s doesn't have", name, c.PluginName))
			}
		}
		return evidence
	},
}
