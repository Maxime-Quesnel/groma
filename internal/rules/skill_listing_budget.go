package rules

import (
	"fmt"
	"unicode/utf8"

	"github.com/Maxime-Quesnel/groma/internal/claudecode"
	"github.com/Maxime-Quesnel/groma/internal/component"
	"github.com/Maxime-Quesnel/groma/internal/rule"
)

const (
	listingBudget  = 8000
	listingPerItem = 1536
)

var skillListingBudget = Rule{
	Meta: rule.Meta{
		ID:    "skill-listing-budget",
		Level: rule.Warning,
		Title: "Skill descriptions that overflow Claude Code's listing on their own",
		Description: "Claude Code lists every skill's name and description in context, within a budget of 1% of the context window, 8,000 characters when it can't tell. " +
			"Past it, it drops descriptions, starting with the skills used least, and Claude loses the words it matches requests against. " +
			"This plugin's descriptions alone fill that budget, before any other plugin's or the user's own.",
		Remediation: "Shorten the descriptions, key use case first, or set disable-model-invocation: true on skills only the user starts, which leaves the listing.",
		FalsePositives: []string{
			"Users on a model with a large context window, whose budget is larger.",
		},
		References: []string{claudecode.SkillsDocs},
	},
	Kinds: []component.Kind{component.Plugin},
	Check: func(c *component.Component, t *component.Tree) []string {
		total, skills := 0, 0
		for _, s := range t.Components {
			if s.Kind != component.Skill && s.Kind != component.Command || !s.InPlugin || s.Plugin != c.Dir ||
				isTrue(s.Header.Value("disable-model-invocation")) {
				continue
			}
			skills++
			described := utf8.RuneCountInString(s.Header.Value("description")) + utf8.RuneCountInString(s.Header.Value("when_to_use"))
			total += utf8.RuneCountInString(s.Name()) + min(described, listingPerItem)
		}
		if total <= listingBudget {
			return nil
		}
		return []string{fmt.Sprintf("%d skills and commands Claude can invoke list %d characters of names and descriptions", skills, total)}
	},
}
