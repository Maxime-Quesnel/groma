package rules

import (
	"fmt"
	"slices"
	"strings"

	"github.com/Maxime-Quesnel/groma/internal/claudecode"
	"github.com/Maxime-Quesnel/groma/internal/component"
	"github.com/Maxime-Quesnel/groma/internal/rule"
)

var hookEnvironmentUnavailable = Rule{
	Meta: rule.Meta{
		ID:    "hook-environment-unavailable",
		Level: rule.RedFlag,
		Title: "Hook relies on something hooks don't have",
		Description: "Command hooks run in their own session with no controlling terminal, so they can't open /dev/tty. " +
			"And only SessionStart, Setup, CwdChanged and FileChanged hooks get CLAUDE_ENV_FILE. A hook that writes to either fails or does nothing.",
		Remediation: "To tell the user something, return a systemMessage or use terminalSequence. To set environment variables, do it from SessionStart.",
		FalsePositives: []string{
			"A script that checks whether /dev/tty or CLAUDE_ENV_FILE is available before using it.",
		},
		References: []string{claudecode.HooksDocs},
	},
	Kinds: hooks,
	Check: func(c *component.Component, t *component.Tree) []string {
		var evidence []string
		for _, h := range c.Hooks.Handlers {
			if h.Type != "command" {
				continue
			}
			code := hookCode(c, t, h)
			if strings.Contains(code, "/dev/tty") {
				evidence = append(evidence, fmt.Sprintf("%s uses /dev/tty", h.Where()))
			}
			if strings.Contains(code, "CLAUDE_ENV_FILE") && !slices.Contains(claudecode.EnvFileEvents, h.Event) {
				evidence = append(evidence, fmt.Sprintf("%s uses CLAUDE_ENV_FILE, which %s hooks don't get", h.Where(), h.Event))
			}
		}
		return evidence
	},
}
