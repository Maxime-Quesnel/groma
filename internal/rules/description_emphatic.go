package rules

import (
	"regexp"
	"strings"

	"github.com/Maxime-Quesnel/groma/internal/claudecode"
	"github.com/Maxime-Quesnel/groma/internal/component"
	"github.com/Maxime-Quesnel/groma/internal/rule"
)

var descriptionEmphatic = Rule{
	Meta: rule.Meta{
		ID:    "description-emphatic",
		Level: rule.Warning,
		Title: "Description that shouts to get picked",
		Description: "Words such as MUST, CRITICAL or ALWAYS in capitals were a way to stop older models from under-using a tool or skill. " +
			"Anthropic's prompting guidance for current models says they now over-trigger on such wording: " +
			"where you might have said \"CRITICAL: You MUST use this tool when…\", use \"Use this tool when…\". " +
			"The description is in context in every conversation, so Claude may reach for the component when it shouldn't.",
		Remediation: "Say when to use it in plain words, such as \"Use when …\" or \"Use proactively after …\", and keep the capitals out.",
		FalsePositives: []string{
			"An acronym written in capitals that happens to be one of these words.",
			"A description tuned against a measured under-triggering problem on the models it runs on.",
		},
		References: []string{claudecode.PromptingDocs},
	},
	Kinds: markdown,
	Check: func(c *component.Component, t *component.Tree) []string {
		if !headerReadable(c) {
			return nil
		}
		var evidence []string
		for _, key := range []string{"description", "when_to_use"} {
			if words := emphatic.FindAllString(c.Header.Value(key), -1); len(words) > 0 {
				evidence = append(evidence, fieldAt(c, key, "%s uses %s", key, strings.Join(dedupe(words), ", ")))
			}
		}
		return evidence
	},
}

var emphatic = regexp.MustCompile(`\b(?:MUST(?: BE USED)?|CRITICAL|ALWAYS|NEVER|IMPORTANT|REQUIRED)\b`)

func dedupe(words []string) []string {
	var unique []string
	for _, w := range words {
		if !strings.Contains(strings.Join(unique, "\n"), w) {
			unique = append(unique, w)
		}
	}
	return unique
}
