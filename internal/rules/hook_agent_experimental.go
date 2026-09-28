package rules

import (
	"fmt"

	"github.com/Maxime-Quesnel/groma/internal/claudecode"
	"github.com/Maxime-Quesnel/groma/internal/component"
	"github.com/Maxime-Quesnel/groma/internal/rule"
)

var hookAgentExperimental = Rule{
	Meta: rule.Meta{
		ID:    "hook-agent-experimental",
		Level: rule.Warning,
		Title: "Agent hook, which is experimental",
		Description: "Agent hooks spawn a subagent to verify a condition. Claude Code's documentation marks them experimental, " +
			"with behaviour and configuration that may change, and recommends command hooks for production workflows.",
		Remediation: "Use a command hook for checks that must hold, or accept that the hook may change behaviour with a Claude Code release.",
		FalsePositives: []string{
			"None: the hook works today; the warning is about stability.",
		},
		References: []string{claudecode.HooksDocs},
	},
	Kinds: hooks,
	Check: func(c *component.Component, t *component.Tree) []string {
		var evidence []string
		for _, h := range c.Hooks.Handlers {
			if h.Type == "agent" {
				evidence = append(evidence, fmt.Sprintf("%s is an agent hook", h.Where()))
			}
		}
		return evidence
	},
}
