package rules

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/Maxime-Quesnel/groma/internal/claudecode"
	"github.com/Maxime-Quesnel/groma/internal/component"
	"github.com/Maxime-Quesnel/groma/internal/rule"
)

var stopHookCanLoop = Rule{
	Meta: rule.Meta{
		ID:    "stop-hook-can-loop",
		Level: rule.Warning,
		Title: "Stop hook that can keep Claude going forever",
		Description: "A Stop or SubagentStop hook that blocks, with exit 2 or a \"block\" decision, sends Claude back to work. " +
			"Claude Code sets stop_hook_active in the hook's input when Claude is already continuing because of a Stop hook; " +
			"a hook that never reads it can block again and again, burning the user's usage until they interrupt.",
		Remediation: "Read stop_hook_active from the hook's JSON input and let Claude stop when it's true.",
		FalsePositives: []string{
			"A hook that bounds its retries some other way, such as a counter in a file.",
			"A script whose exit 2 or block decision sits in a branch that can't run on Stop.",
		},
		References: []string{claudecode.HooksDocs},
	},
	Kinds: hooks,
	Check: func(c *component.Component, t *component.Tree) []string {
		var evidence []string
		for _, h := range c.Hooks.Handlers {
			if h.Event != "Stop" && h.Event != "SubagentStop" || h.Type != "command" {
				continue
			}
			code := []string{h.Run()}
			for _, s := range t.Scripts(h.Run(), c.Roots()) {
				if f, ok := t.File(s.Path); ok {
					code = append(code, string(f.Content))
				}
			}
			text := strings.Join(code, "\n")
			if blocks.MatchString(text) && !strings.Contains(text, "stop_hook_active") {
				evidence = append(evidence, fmt.Sprintf("%s can block, and never reads stop_hook_active", h.Where()))
			}
		}
		return evidence
	},
}

var blocks = regexp.MustCompile(`\bexit\s*\(?\s*2\s*\)?(?:\s|;|$)|exit!\s*\(?2|process\.exit\(\s*2\s*\)|sys\.exit\(\s*2\s*\)|os\.Exit\(\s*2\s*\)|["']?decision["']?\s*(?::|=>|=)\s*["']block["']`)
