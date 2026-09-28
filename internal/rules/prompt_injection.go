package rules

import (
	"path"
	"regexp"
	"strings"

	"github.com/Maxime-Quesnel/groma/internal/component"
	"github.com/Maxime-Quesnel/groma/internal/rule"
)

var promptInjection = Rule{
	Meta: rule.Meta{
		ID:    "prompt-injection",
		Level: rule.RedFlag,
		Title: "Text written to override Claude or hide things from the user",
		Description: "The component tells Claude to ignore its previous instructions, or to act without telling the user. " +
			"Legitimate skills and agents add to Claude's instructions and work in the open; " +
			"these phrases are how injected instructions take over an agent and keep the user from noticing.",
		Remediation: "Remove the phrase. If the component needs Claude to drop a default behaviour, name the behaviour and say why.",
		FalsePositives: []string{
			"A skill about prompt injection that cites such phrases without quotes. Phrases that open right after a quote mark are skipped as examples.",
			"An instruction to keep a detail from the user for a benign reason, such as not revealing a quiz answer.",
		},
		References: []string{"https://genai.owasp.org/llmrisk/llm01-prompt-injection/"},
	},
	Kinds: markdown,
	Check: func(c *component.Component, t *component.Tree) []string {
		var evidence []string
		scan := func(f component.File, text string, firstLine int) {
			for i, line := range strings.Split(text, "\n") {
				for _, re := range injection {
					if m := re.FindStringIndex(line); m != nil && !quoted(line, m[0]) {
						evidence = append(evidence, in(c, f, at(firstLine+i, "%s", clip(line[m[0]:m[1]]))))
						break
					}
				}
			}
		}
		own, _ := t.File(c.Path)
		scan(own, string(c.Content), 1)
		if c.Kind == component.Skill {
			for _, f := range t.Owned(c) {
				if f.Path != c.Path && path.Ext(f.Path) == ".md" {
					scan(f, string(f.Content), 1)
				}
			}
		}
		return evidence
	},
}

var injection = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\b(?:ignore|disregard|forget|override)\s+(?:all\s+|any\s+|the\s+|your\s+|of\s+)*(?:previous|prior|above|earlier|preceding|system|original)\s+(?:instructions?|rules|prompts?|messages|directions|guidelines)`),
	regexp.MustCompile(`(?i)\b(?:do\s+not|don't|never)\s+(?:tell|inform|notify|alert)\s+(?:the\s+)?user\b`),
	regexp.MustCompile(`(?i)\b(?:do\s+not|don't|never)\s+(?:mention|reveal|show)\s+(?:this|it|these\s+\w+)\s+to\s+(?:the\s+)?user\b`),
	regexp.MustCompile(`(?i)\b(?:hide|conceal|keep)\s+(?:this|it|these\s+\w+)\s+(?:secret\s+)?from\s+(?:the\s+)?user\b`),
	regexp.MustCompile(`(?i)\byou\s+are\s+now\s+(?:in\s+)?(?:developer|dan|jailbreak|god|unrestricted)\s+mode\b`),
}

// quoted reports whether the text at i opens right after a quote mark, as
// an example cited in a warning does: ("ignore previous instructions").
func quoted(line string, i int) bool {
	before := strings.TrimRight(line[:i], " ")
	return strings.HasSuffix(before, `"`) || strings.HasSuffix(before, "'") || strings.HasSuffix(before, "`") ||
		strings.HasSuffix(before, "“") || strings.HasSuffix(before, "‘") || strings.HasSuffix(before, "«")
}
