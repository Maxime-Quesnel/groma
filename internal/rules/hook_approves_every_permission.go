package rules

import (
	"fmt"
	"regexp"

	"github.com/Maxime-Quesnel/groma/internal/claudecode"
	"github.com/Maxime-Quesnel/groma/internal/component"
	"github.com/Maxime-Quesnel/groma/internal/rule"
)

var hookApprovesEveryPermission = Rule{
	Meta: rule.Meta{
		ID:    "hook-approves-every-permission",
		Level: rule.RedFlag,
		Title: "Hook that approves every permission prompt",
		Description: "A PermissionRequest hook with an empty matcher, * or .* that answers allow approves every permission prompt, including file writes and shell commands, " +
			"and a hook that switches the session to bypassPermissions turns prompts off altogether. Whatever steers Claude then acts unchecked.",
		Remediation: "Match only the tools the hook is meant to approve, as narrowly as possible, and never switch the session to bypassPermissions from a hook.",
		FalsePositives: []string{
			"A hook whose script only answers allow for requests it checks itself; groma doesn't follow the script's branches.",
		},
		References: []string{claudecode.HooksGuide, claudecode.HooksDocs},
	},
	Kinds: hooks,
	Check: func(c *component.Component, t *component.Tree) []string {
		var evidence []string
		for _, h := range c.Hooks.Handlers {
			if h.Type != "command" {
				continue
			}
			code := hookCode(c, t, h)
			broad := h.Matcher == "" || h.Matcher == "*" || h.Matcher == ".*"
			if h.Event == "PermissionRequest" && broad && allows.MatchString(code) {
				evidence = append(evidence, fmt.Sprintf("%s answers allow to every permission request", h.Where()))
			}
			if bypass.MatchString(code) {
				evidence = append(evidence, fmt.Sprintf("%s switches the session to bypassPermissions", h.Where()))
			}
		}
		return evidence
	},
}

var (
	allows = regexp.MustCompile(`["']?behavior["']?\s*(?::|=>|=)\s*["']allow["']`)
	bypass = regexp.MustCompile(`setMode[\s\S]{0,80}bypassPermissions`)
)
