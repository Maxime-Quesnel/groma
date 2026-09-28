package rules

import (
	"regexp"
	"strings"

	"github.com/Maxime-Quesnel/groma/internal/claudecode"
	"github.com/Maxime-Quesnel/groma/internal/component"
	"github.com/Maxime-Quesnel/groma/internal/rule"
)

var descriptionVoice = Rule{
	Meta: rule.Meta{
		ID:    "description-voice",
		Level: rule.Warning,
		Title: "Description in the first or second person",
		Description: "Descriptions are injected into Claude's system prompt. Anthropic's guidance is to write them in the third person, " +
			"\"Processes Excel files\", not \"I can help you process Excel files\" or \"You can use this to…\": a shifting point of view makes Claude's choice less reliable.",
		Remediation: "Rewrite it in the third person, starting with what the component does.",
		FalsePositives: []string{
			"A first sentence that quotes what a user might say. groma only reads the first sentence, so later examples don't count.",
		},
		References: []string{claudecode.BestPractices},
	},
	Kinds: markdown,
	Check: func(c *component.Component, t *component.Tree) []string {
		if !headerReadable(c) {
			return nil
		}
		// Only the opening counts: descriptions often quote a dialogue as an
		// example after it, where the assistant rightly says "I'll".
		if m := person.FindString(firstSentence(c.Header.Value("description"))); m != "" {
			return []string{fieldAt(c, "description", "%q", m)}
		}
		return nil
	},
}

var person = regexp.MustCompile(`(?i)^\s*(?:i|i'm|i'll|i've|we|we're|we'll|my|our|you|your)\b|\b(?:i can|i will|i'll|let me|we can|we will|you can use (?:this|me|it)|helps you|lets you|allows you)\b`)

func firstSentence(s string) string {
	if i := strings.IndexAny(s, ".!?\n"); i >= 0 {
		return s[:i]
	}
	return s
}
