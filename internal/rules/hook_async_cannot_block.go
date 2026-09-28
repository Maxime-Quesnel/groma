package rules

import (
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/Maxime-Quesnel/groma/internal/claudecode"
	"github.com/Maxime-Quesnel/groma/internal/component"
	"github.com/Maxime-Quesnel/groma/internal/rule"
)

var hookAsyncCannotBlock = Rule{
	Meta: rule.Meta{
		ID:    "hook-async-cannot-block",
		Level: rule.RedFlag,
		Title: "Async hook written to block or decide",
		Description: "An async hook runs in the background after the action it watches has gone ahead, so decision, permissionDecision and continue have no effect, and neither does exit 2. " +
			"A guard marked async guards nothing.",
		Remediation: "Remove async from a hook that must block or decide. For a long check that should wake Claude on failure, use asyncRewake with exit 2.",
		FalsePositives: []string{
			"A script shared with a synchronous hook, whose deciding branch the async one never takes.",
		},
		References: []string{claudecode.HooksDocs},
	},
	Kinds: hooks,
	Check: func(c *component.Component, t *component.Tree) []string {
		var evidence []string
		for _, h := range c.Hooks.Handlers {
			var async, rewake bool
			json.Unmarshal(h.Fields["async"], &async)
			json.Unmarshal(h.Fields["asyncRewake"], &rewake)
			if !async || rewake || h.Type != "command" {
				continue
			}
			if m := decides.FindString(hookCode(c, t, h)); m != "" {
				evidence = append(evidence, fmt.Sprintf("%s is async, and its code has %s", h.Where(), m))
			}
		}
		return evidence
	},
}

var decides = regexp.MustCompile(`permissionDecision|["']?decision["']?\s*(?::|=>|=)|["']?continue["']?\s*(?::|=>|=)\s*false|\bexit\s*\(?\s*2\b|sys\.exit\(\s*2\s*\)|process\.exit\(\s*2\s*\)`)
