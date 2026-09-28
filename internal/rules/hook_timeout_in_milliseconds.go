package rules

import (
	"encoding/json"
	"fmt"

	"github.com/Maxime-Quesnel/groma/internal/claudecode"
	"github.com/Maxime-Quesnel/groma/internal/component"
	"github.com/Maxime-Quesnel/groma/internal/fix"
	"github.com/Maxime-Quesnel/groma/internal/rule"
	"math"
	"strconv"
	"strings"
)

const maxTimeout = 3600

var hookTimeoutInMilliseconds = Rule{
	Meta: rule.Meta{
		ID:    "hook-timeout-in-milliseconds",
		Level: rule.Warning,
		Title: "Hook timeout that looks like milliseconds",
		Description: "A hook's timeout is in seconds. A value over an hour, such as 5000, is almost always meant as milliseconds, " +
			"and lets a stuck hook hold the session far longer than its author intended.",
		Remediation: "Give the timeout in seconds: 5 rather than 5000.",
		FalsePositives: []string{
			"A hook that genuinely runs for over an hour, such as a long background job.",
		},
		References: []string{claudecode.HooksDocs},
	},
	Kinds: hooks,
	Check: func(c *component.Component, t *component.Tree) []string {
		var evidence []string
		for _, h := range c.Hooks.Handlers {
			var seconds float64
			if json.Unmarshal(h.Fields["timeout"], &seconds) == nil && seconds > maxTimeout {
				evidence = append(evidence, fmt.Sprintf("%s has timeout %v, which is %.1f hours", h.Where(), seconds, seconds/3600))
			}
		}
		return evidence
	},
	Fix: fixTimeoutUnits,
}

// fixTimeoutUnits reads a timeout over an hour as milliseconds and writes it
// in seconds. A stuck hook is then cut off sooner, so the fix is unsafe.
func fixTimeoutUnits(c *component.Component, t *component.Tree, unsafe bool) []fix.Edit {
	if !unsafe {
		return nil
	}
	var edits []fix.Edit
	for _, h := range c.Hooks.Handlers {
		var ms float64
		raw := strings.TrimSpace(string(h.Fields["timeout"]))
		if json.Unmarshal(h.Fields["timeout"], &ms) != nil || ms <= maxTimeout {
			continue
		}
		seconds := strconv.Itoa(max(1, int(math.Ceil(ms/1000))))
		edits = append(edits, jsonValueEdits(c, "timeout", raw, seconds)...)
	}
	return edits
}
