package rules

import (
	"fmt"
	"regexp"

	"github.com/Maxime-Quesnel/groma/internal/claudecode"
	"github.com/Maxime-Quesnel/groma/internal/component"
	"github.com/Maxime-Quesnel/groma/internal/rule"
)

const maxEmphasized = 4

var emphasisOverused = Rule{
	Meta: rule.Meta{
		ID:    "emphasis-overused",
		Level: rule.Warning,
		Title: "Too many lines in capitals",
		Description: "Anthropic's guidance for current Claude models is to dial back aggressive language: MUST, NEVER, ALWAYS or CRITICAL in capitals make Claude over-apply an instruction, " +
			"and when many lines are emphasized, none of them stands out. Explaining why a rule matters works better than shouting it.",
		Remediation: "Keep capitals for the one or two rules Claude keeps missing, and give the reason for the others in plain words.",
		FalsePositives: []string{
			"Capitals that quote a specification, such as the MUST and SHOULD of an RFC.",
		},
		References: []string{claudecode.PromptingDocs, claudecode.SkillCreator},
	},
	Kinds: markdown,
	Check: func(c *component.Component, t *component.Tree) []string {
		var lines []int
		bodyLines(c, func(n int, line string) {
			if shouting.MatchString(line) {
				lines = append(lines, n)
			}
		})
		if len(lines) <= maxEmphasized {
			return nil
		}
		return []string{fmt.Sprintf("%d lines use MUST, NEVER, ALWAYS, CRITICAL, IMPORTANT, REQUIRED or DO NOT in capitals, starting on line %d", len(lines), lines[0])}
	},
}

var shouting = regexp.MustCompile(`\b(?:MUST|NEVER|ALWAYS|CRITICAL|IMPORTANT|REQUIRED|DO NOT)\b`)
