package rules

import (
	"fmt"
	"slices"
	"strings"

	"github.com/Maxime-Quesnel/groma/internal/claudecode"
	"github.com/Maxime-Quesnel/groma/internal/component"
	"github.com/Maxime-Quesnel/groma/internal/rule"
)

var hookNeverRuns = Rule{
	Meta: rule.Meta{
		ID:    "hook-never-runs",
		Level: rule.RedFlag,
		Title: "Hook that never runs",
		Description: "Matchers are case-sensitive, so a matcher such as bash or write never matches the Bash or Write tool. " +
			"And a hook's if condition is only evaluated on tool events; on any other event, a hook with if never runs. " +
			"A guard written this way silently guards nothing.",
		Remediation: "Spell tool names exactly as Claude Code does, such as Bash or Edit|Write. On events other than tool events, drop if and filter inside the hook's command.",
		FalsePositives: []string{
			"A matcher meant for an MCP tool whose name only differs in case from a built-in tool.",
		},
		References: []string{claudecode.HooksDocs},
	},
	Kinds: hooks,
	Check: func(c *component.Component, t *component.Tree) []string {
		var evidence []string
		for _, g := range c.Hooks.Groups {
			if !slices.Contains(claudecode.ToolEvents, g.Event) || !g.HasMatcher {
				continue
			}
			for _, alt := range strings.Split(g.Matcher, "|") {
				alt = strings.TrimSpace(alt)
				if slices.Contains(claudecode.Tools, alt) {
					continue
				}
				if i := slices.IndexFunc(claudecode.Tools, func(tool string) bool { return strings.EqualFold(tool, alt) }); i >= 0 {
					evidence = append(evidence, fmt.Sprintf("%s: %s never matches, since matchers are case-sensitive; the tool is %s", g.Where(), alt, claudecode.Tools[i]))
				}
			}
		}
		for _, h := range c.Hooks.Handlers {
			if _, ok := h.Fields["if"]; ok && !slices.Contains(claudecode.ToolEvents, h.Event) {
				evidence = append(evidence, fmt.Sprintf("%s has an if condition, which %s never evaluates, so the hook never runs", h.Where(), h.Event))
			}
		}
		return evidence
	},
}
