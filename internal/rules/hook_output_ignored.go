package rules

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/Maxime-Quesnel/groma/internal/claudecode"
	"github.com/Maxime-Quesnel/groma/internal/component"
	"github.com/Maxime-Quesnel/groma/internal/rule"
)

var hookOutputIgnored = Rule{
	Meta: rule.Meta{
		ID:    "hook-output-ignored",
		Level: rule.RedFlag,
		Title: "Hook output Claude Code ignores",
		Description: "Claude Code reads permissionDecision, additionalContext and updatedInput only inside hookSpecificOutput, which itself needs hookEventName; " +
			"at the top level they parse and are silently ignored. And outside PreToolUse, the only value of decision is \"block\": \"approve\" does nothing. " +
			"The decision or context the hook computes never reaches Claude.",
		Remediation: `Print {"hookSpecificOutput": {"hookEventName": "<event>", "permissionDecision": "deny", ...}}, or omit decision to let the action through.`,
		FalsePositives: []string{
			"A script that builds the object in a way groma's text search misses, such as keys assembled from variables.",
			"An async hook, whose top-level additionalContext Claude Code delivers on the next turn; async hooks aren't checked.",
		},
		References: []string{claudecode.HooksDocs, claudecode.HooksGuide},
	},
	Kinds: hooks,
	Check: func(c *component.Component, t *component.Tree) []string {
		var evidence []string
		for _, h := range c.Hooks.Handlers {
			var async bool
			json.Unmarshal(h.Fields["async"], &async)
			if h.Type != "command" || async {
				continue
			}
			code := hookCode(c, t, h)
			nested := strings.Contains(code, "hookSpecificOutput")
			switch {
			case nested && !strings.Contains(code, "hookEventName"):
				evidence = append(evidence, fmt.Sprintf("%s writes hookSpecificOutput without hookEventName", h.Where()))
			case !nested && specificField.MatchString(code):
				evidence = append(evidence, fmt.Sprintf("%s writes %s outside hookSpecificOutput", h.Where(), specificField.FindString(code)))
			}
			if h.Event != "PreToolUse" && approve.MatchString(code) {
				evidence = append(evidence, fmt.Sprintf("%s returns the decision \"approve\", which %s doesn't have", h.Where(), h.Event))
			}
		}
		return evidence
	},
}

var (
	specificField = regexp.MustCompile(`permissionDecision|additionalContext|updatedInput`)
	approve       = regexp.MustCompile(`["']?decision["']?\s*(?::|=>|=)\s*["']approve["']`)
)
