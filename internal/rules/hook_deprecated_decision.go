package rules

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/Maxime-Quesnel/groma/internal/claudecode"
	"github.com/Maxime-Quesnel/groma/internal/component"
	"github.com/Maxime-Quesnel/groma/internal/rule"
)

var hookDeprecatedDecision = Rule{
	Meta: rule.Meta{
		ID:    "hook-deprecated-decision",
		Level: rule.Warning,
		Title: "PreToolUse hook with the deprecated decision field",
		Description: "PreToolUse used to take a top-level decision of \"approve\" or \"block\". Both are deprecated for this event: Claude Code still maps them to allow and deny, " +
			"but the current form, hookSpecificOutput.permissionDecision, also offers ask and defer and a reason Claude sees.",
		Remediation: `Print {"hookSpecificOutput": {"hookEventName": "PreToolUse", "permissionDecision": "deny", "permissionDecisionReason": "..."}}.`,
		FalsePositives: []string{
			"A script shared with a PostToolUse or Stop hook, where the top-level decision is current.",
		},
		References: []string{claudecode.HooksDocs},
	},
	Kinds: hooks,
	Check: func(c *component.Component, t *component.Tree) []string {
		var evidence []string
		for _, h := range c.Hooks.Handlers {
			if h.Event != "PreToolUse" || h.Type != "command" {
				continue
			}
			code := hookCode(c, t, h)
			if m := oldDecision.FindString(code); m != "" && !strings.Contains(code, "permissionDecision") {
				evidence = append(evidence, fmt.Sprintf("%s returns %s", h.Where(), m))
			}
		}
		return evidence
	},
}

var oldDecision = regexp.MustCompile(`["']?decision["']?\s*(?::|=>|=)\s*["'](?:approve|block)["']`)
