package rules

import (
	"regexp"
	"strings"

	"github.com/Maxime-Quesnel/groma/internal/claudecode"
	"github.com/Maxime-Quesnel/groma/internal/component"
	"github.com/Maxime-Quesnel/groma/internal/rule"
)

var descriptionNoTrigger = Rule{
	Meta: rule.Meta{
		ID:    "description-no-trigger",
		Level: rule.Warning,
		Title: "Description doesn't say when to use it",
		Description: "Claude chooses among skills and agents by their descriptions. One that only says what the component is, such as \"Helps with documents\", " +
			"gives Claude nothing to match a request against. Anthropic's guidance is to state both what it does and when to use it: " +
			"\"Use when working with PDF files\", \"Use after writing or modifying code\".",
		Remediation: "Add the situations that should trigger it: \"Use when …\", \"Use after …\", or \"Use proactively when …\" for an agent.",
		FalsePositives: []string{
			"A description that names its triggers without words such as when, after, before, if, proactively or dispatched by.",
		},
		References: []string{claudecode.BestPractices, claudecode.SubagentsDocs},
	},
	Kinds: skillsAndAgents,
	Check: func(c *component.Component, t *component.Tree) []string {
		text := strings.TrimSpace(c.Header.Value("description"))
		// when_to_use says when by definition, and Claude never picks a skill
		// the user alone may start.
		if !headerReadable(c) || text == "" || c.Header.Value("when_to_use") != "" ||
			strings.EqualFold(c.Header.Value("disable-model-invocation"), "true") || trigger.MatchString(text) {
			return nil
		}
		return []string{fieldAt(c, "description", "%s", clip(text))}
	},
}

var trigger = regexp.MustCompile(`(?i)\b(?:when|whenever|after|before|proactively|must be used|if (?:the )?user|if you|in response to|for (?:any|every)|on (?:any|every)|dispatch(?:ed|es)?|(?:invoked|called|spawned|used) by)\b|\buse (?:it |this(?: skill| agent)? )?(?:for|to|on|with)\b`)
