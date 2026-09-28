package rules

import (
	"regexp"
	"strings"

	"github.com/Maxime-Quesnel/groma/internal/claudecode"
	"github.com/Maxime-Quesnel/groma/internal/component"
	"github.com/Maxime-Quesnel/groma/internal/rule"
)

var triggerInBody = Rule{
	Meta: rule.Meta{
		ID:    "trigger-in-body",
		Level: rule.Warning,
		Title: "When to use it is written in the body",
		Description: "Claude decides to use a skill from its description alone; the body loads only once the skill is chosen. " +
			"A \"When to use\" section in the body can't help Claude choose, and Anthropic's skill-creator puts all of it in the description.",
		Remediation: "Move the triggers into the description or when_to_use, and keep the body for what to do once the skill runs.",
		FalsePositives: []string{
			"A section about when to use a tool or command inside the skill, rather than the skill itself.",
		},
		References: []string{claudecode.SkillCreator, claudecode.SkillsDocs},
	},
	Kinds: skills,
	Check: func(c *component.Component, t *component.Tree) []string {
		if !headerReadable(c) || isTrue(c.Header.Value("disable-model-invocation")) {
			return nil
		}
		var evidence []string
		bodyLines(c, func(n int, line string) {
			if triggerHeading.MatchString(line) {
				evidence = append(evidence, at(n, "%s", strings.TrimSpace(line)))
			}
		})
		return evidence
	},
}

var triggerHeading = regexp.MustCompile(`(?i)^#{1,6}\s*(?:when to (?:use|invoke|trigger)(?: this skill)?|triggers?|use this skill when)\s*:?\s*$`)
