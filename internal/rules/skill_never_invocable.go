package rules

import (
	"github.com/Maxime-Quesnel/groma/internal/claudecode"
	"github.com/Maxime-Quesnel/groma/internal/component"
	"github.com/Maxime-Quesnel/groma/internal/rule"
)

var skillNeverInvocable = Rule{
	Meta: rule.Meta{
		ID:    "skill-never-invocable",
		Level: rule.RedFlag,
		Title: "Nobody can run this skill",
		Description: "user-invocable: false hides the skill from the / menu and stops the user from running it; disable-model-invocation: true stops Claude " +
			"from invoking it and from preloading it into subagents. With both, nothing can ever start the skill.",
		Remediation: "Drop one of the two: keep user-invocable: false for background knowledge only Claude uses, or disable-model-invocation: true for a workflow only the user starts.",
		FalsePositives: []string{
			"A skill deliberately parked, kept for later.",
		},
		References: []string{claudecode.SkillsDocs},
	},
	Kinds: skillsAndCommands,
	Check: func(c *component.Component, t *component.Tree) []string {
		if !headerReadable(c) || !isTrue(c.Header.Value("disable-model-invocation")) || !isFalse(c.Header.Value("user-invocable")) {
			return nil
		}
		return []string{fieldAt(c, "user-invocable", "user-invocable: false with disable-model-invocation: true")}
	},
}
