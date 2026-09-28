package rules

import (
	"fmt"
	"slices"
	"strings"

	"github.com/Maxime-Quesnel/groma/internal/claudecode"
	"github.com/Maxime-Quesnel/groma/internal/component"
	"github.com/Maxime-Quesnel/groma/internal/fix"
	"github.com/Maxime-Quesnel/groma/internal/rule"
)

var hookNeverRuns = Rule{
	Meta: rule.Meta{
		ID:    "hook-never-runs",
		Level: rule.RedFlag,
		Title: "Hook that never runs",
		Description: "A matcher made of letters, digits, _, -, | and commas is compared as exact values, case-sensitively: bash never matches the Bash tool, " +
			"MultiEdit matches a tool Claude Code no longer has, mcp__memory matches no MCP tool, and SessionStart only matches startup, resume, clear, compact or fork. " +
			"A hook's if condition holds one permission rule and is only evaluated on tool events. A guard written this way silently guards nothing.",
		Remediation: "Spell tool names exactly as Claude Code does, such as Bash or Edit|Write, write mcp__memory__.* for every tool of a server, " +
			"and use one if rule per hook, on a tool event, naming a tool the matcher covers.",
		FalsePositives: []string{
			"A matcher value added by a Claude Code release newer than groma's lists.",
		},
		References: []string{claudecode.HooksDocs},
	},
	Kinds: hooks,
	Check: func(c *component.Component, t *component.Tree) []string {
		var evidence []string
		for _, g := range c.Hooks.Groups {
			alternatives, exact := claudecode.MatcherAlternatives(g.Matcher)
			if !exact {
				continue
			}
			var reasons []string
			for _, a := range alternatives {
				if why := deadMatcher(g.Event, a); why != "" {
					reasons = append(reasons, why)
				}
			}
			if len(reasons) == len(alternatives) {
				evidence = append(evidence, fmt.Sprintf("%s: %s", g.Where(), strings.Join(reasons, "; ")))
			}
		}
		for _, h := range c.Hooks.Handlers {
			condition, ok := h.String("if")
			if !ok {
				continue
			}
			if why := deadCondition(h, condition); why != "" {
				evidence = append(evidence, fmt.Sprintf("%s has if %q, %s", h.Where(), condition, why))
			}
		}
		return evidence
	},
	Fix: fixNeverRuns,
}

// deadMatcher says why one exact matcher value can never match on event, or
// returns "" when it can or groma can't tell.
func deadMatcher(event, value string) string {
	if values, ok := claudecode.MatcherValues[event]; ok && !slices.Contains(values, value) {
		return fmt.Sprintf("%s isn't a value %s matches", value, event)
	}
	if !slices.Contains(claudecode.ToolEvents, event) {
		return ""
	}
	switch {
	case slices.Contains(claudecode.Tools, value):
		return ""
	case strings.HasPrefix(value, "mcp__") && !strings.Contains(strings.TrimPrefix(value, "mcp__"), "__"):
		return fmt.Sprintf("%s matches no tool; %s__.* matches every tool of the server", value, value)
	case claudecode.FormerTools[value] != "":
		return fmt.Sprintf("Claude Code no longer has %s", value)
	}
	if i := slices.IndexFunc(claudecode.Tools, func(tool string) bool { return strings.EqualFold(tool, value) }); i >= 0 {
		return fmt.Sprintf("%s never matches, since matchers are case-sensitive; the tool is %s", value, claudecode.Tools[i])
	}
	return ""
}

// deadCondition says why a hook's if condition never lets it run.
func deadCondition(h component.Handler, condition string) string {
	if !slices.Contains(claudecode.ToolEvents, h.Event) {
		return fmt.Sprintf("which %s never evaluates", h.Event)
	}
	if strings.Contains(condition, "&&") || strings.Contains(condition, "||") || len(splitRules(condition)) > 1 {
		return "but if holds exactly one permission rule, with no && , || or list"
	}
	tool := claudecode.Tool(condition)
	if why := deadMatcher(h.Event, tool); why != "" {
		return "and " + why
	}
	if alternatives, exact := claudecode.MatcherAlternatives(h.Matcher); exact && h.HasMatcher && !slices.Contains(alternatives, tool) {
		return fmt.Sprintf("but the group's matcher never lets %s through", tool)
	}
	return ""
}

// splitRules splits a condition at commas outside parentheses.
func splitRules(s string) []string {
	var rules []string
	depth, start := 0, 0
	for i, r := range s {
		switch {
		case r == '(':
			depth++
		case r == ')':
			depth--
		case r == ',' && depth == 0:
			rules = append(rules, s[start:i])
			start = i + 1
		}
	}
	return append(rules, s[start:])
}

// fixNeverRuns makes a dead matcher or if condition name what it evidently
// meant: the tool in its right case, the tool that replaced a removed one,
// every tool of an MCP server. The hook then runs, so the fix is unsafe.
func fixNeverRuns(c *component.Component, t *component.Tree, unsafe bool) []fix.Edit {
	if !unsafe {
		return nil
	}
	var edits []fix.Edit
	for _, g := range c.Hooks.Groups {
		alternatives, exact := claudecode.MatcherAlternatives(g.Matcher)
		if !exact || !slices.Contains(claudecode.ToolEvents, g.Event) {
			continue
		}
		var meant []string
		for _, a := range alternatives {
			m := meantTool(a)
			if m == "" || deadMatcher(g.Event, a) == "" {
				meant = nil
				break
			}
			if !slices.Contains(meant, m) {
				meant = append(meant, m)
			}
		}
		// An MCP server's wildcard makes the whole matcher a regular
		// expression, so it only replaces a matcher on its own.
		if len(meant) == 0 || len(meant) > 1 && slices.ContainsFunc(meant, func(m string) bool { return strings.Contains(m, "*") }) {
			continue
		}
		edits = append(edits, jsonValueEdits(c, "matcher", jsonString(g.Matcher), jsonString(joinMatcher(g.Matcher, meant)))...)
	}
	for _, h := range c.Hooks.Handlers {
		condition, ok := h.String("if")
		if !ok || !slices.Contains(claudecode.ToolEvents, h.Event) {
			continue
		}
		tool := claudecode.Tool(condition)
		if m := meantTool(tool); m != "" && m != tool && !strings.Contains(m, "*") {
			edits = append(edits, jsonValueEdits(c, "if", jsonString(condition), jsonString(m+strings.TrimPrefix(condition, tool)))...)
		}
	}
	return edits
}

// meantTool returns the tool a dead matcher value evidently means, or "".
func meantTool(value string) string {
	switch {
	case claudecode.FormerTools[value] != "":
		return claudecode.FormerTools[value]
	case strings.HasPrefix(value, "mcp__") && !strings.Contains(strings.TrimPrefix(value, "mcp__"), "__"):
		return value + "__.*"
	}
	return toolCase(value)
}
