package rules

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/Maxime-Quesnel/groma/internal/claudecode"
	"github.com/Maxime-Quesnel/groma/internal/component"
	"github.com/Maxime-Quesnel/groma/internal/rule"
)

var hookExit1DoesNotBlock = Rule{
	Meta: rule.Meta{
		ID:    "hook-exit-1-does-not-block",
		Level: rule.RedFlag,
		Title: "Guard that exits 1, which doesn't block",
		Description: "Exit code 2 is the only exit code that blocks. Without JSON on stdout, Claude Code treats exit 1, the usual Unix failure, as a non-blocking error " +
			"and goes ahead with the action. A hook that prints \"blocked\" and exits 1 lets everything through.",
		Remediation: "Exit 2 to block, with the reason on stderr, or print a JSON decision on stdout and exit 0.",
		FalsePositives: []string{
			"A script whose blocking logic lives in a module groma doesn't read, or that exits 1 for an error it means not to block on.",
		},
		References: []string{claudecode.HooksDocs},
	},
	Kinds: hooks,
	Check: func(c *component.Component, t *component.Tree) []string {
		var evidence []string
		for _, h := range c.Hooks.Handlers {
			if h.Type != "command" || !slices.Contains(claudecode.BlockingEvents, h.Event) {
				continue
			}
			code := hookCode(c, t, h)
			if exitTwo.MatchString(code) || strings.Contains(code, "permissionDecision") || decisionKey.MatchString(code) {
				continue
			}
			lines := strings.Split(code, "\n")
			for i, line := range lines {
				if !exitOne.MatchString(line) {
					continue
				}
				for _, before := range lines[max(0, i-3) : i+1] {
					if denial.MatchString(before) {
						evidence = append(evidence, fmt.Sprintf("%s denies with %s", h.Where(), clip(strings.TrimSpace(line))))
						break
					}
				}
			}
		}
		return evidence
	},
}

var (
	exitOne     = regexp.MustCompile(`\bexit\s*\(?\s*1\b|sys\.exit\(\s*1\s*\)|process\.exit\(\s*1\s*\)|os\.Exit\(\s*1\s*\)`)
	exitTwo     = regexp.MustCompile(`\bexit\s*\(?\s*2\b|sys\.exit\(\s*2\s*\)|process\.exit\(\s*2\s*\)|os\.Exit\(\s*2\s*\)`)
	decisionKey = regexp.MustCompile(`["']?decision["']?\s*(?::|=>|=)`)
	denial      = regexp.MustCompile(`(?i)\b(?:block(?:ed|ing)?|den(?:y|ied)|not allowed|forbidden|refus(?:e|ed)|reject(?:ed)?)\b`)
)
