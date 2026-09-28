package rules

import (
	"regexp"

	"github.com/Maxime-Quesnel/groma/internal/claudecode"
	"github.com/Maxime-Quesnel/groma/internal/component"
	"github.com/Maxime-Quesnel/groma/internal/rule"
)

var timeSensitiveText = Rule{
	Meta: rule.Meta{
		ID:    "time-sensitive-text",
		Level: rule.Warning,
		Title: "Instruction that depends on today's date",
		Description: "\"Before August 2025, use the old API\" becomes wrong on its own, and Claude has no reliable way to know the date it reads it. " +
			"Anthropic's skill guidance is to describe the current way, and keep the old one in a clearly marked old-patterns section.",
		Remediation: "State the current behaviour, and move history into an \"Old patterns\" section, dated if needed.",
		FalsePositives: []string{
			"A date that is history, such as a changelog entry, rather than a condition to act on.",
		},
		References: []string{claudecode.BestPractices},
	},
	Kinds: markdown,
	Check: func(c *component.Component, t *component.Tree) []string {
		var evidence []string
		bodyLines(c, func(n int, line string) {
			if m := dated.FindString(line); m != "" {
				evidence = append(evidence, at(n, "%s", m))
			}
		})
		return evidence
	},
}

var dated = regexp.MustCompile(`(?i)\b(?:before|after|until|as of)\s+(?:(?:jan(?:uary)?|feb(?:ruary)?|mar(?:ch)?|apr(?:il)?|may|june?|july?|aug(?:ust)?|sept?(?:ember)?|oct(?:ober)?|nov(?:ember)?|dec(?:ember)?)\s+(?:\d{1,2},?\s+)?|q[1-4]\s+)?20\d\d\b`)
