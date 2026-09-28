package rules

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Maxime-Quesnel/groma/internal/claudecode"
	"github.com/Maxime-Quesnel/groma/internal/component"
	"github.com/Maxime-Quesnel/groma/internal/rule"
)

var hookHandlerInvalid = Rule{
	Meta: rule.Meta{
		ID:    "hook-handler-invalid",
		Level: rule.RedFlag,
		Title: "Hook Claude Code can't run",
		Description: "The hook has no type or one Claude Code doesn't have, lacks the field its type needs (command, url, server and tool, or prompt), " +
			"or sets a timeout that isn't a number of seconds above 0. Claude Code skips it or fails to run it.",
		Remediation: "Give the hook a type of command, http, mcp_tool, prompt or agent, and the fields that type requires.",
		FalsePositives: []string{
			"A hook type added by a Claude Code release newer than groma's list.",
		},
		References: []string{claudecode.HooksDocs},
	},
	Kinds: hooks,
	Check: func(c *component.Component, t *component.Tree) []string {
		var evidence []string
		for _, h := range c.Hooks.Handlers {
			spec, known := claudecode.HandlerFields[h.Type]
			switch {
			case h.Type == "":
				evidence = append(evidence, fmt.Sprintf("%s has no type", h.Where()))
			case !known:
				evidence = append(evidence, fmt.Sprintf("%s has type %q, which isn't command, http, mcp_tool, prompt or agent", h.Where(), h.Type))
			default:
				for _, field := range spec.Required {
					if s, ok := h.String(field); !ok || strings.TrimSpace(s) == "" {
						evidence = append(evidence, fmt.Sprintf("%s is a %s hook without %s", h.Where(), h.Type, field))
					}
				}
			}
			if raw, ok := h.Fields["timeout"]; ok {
				var seconds float64
				if json.Unmarshal(raw, &seconds) != nil || seconds <= 0 {
					evidence = append(evidence, fmt.Sprintf("%s has timeout %s, which isn't a number of seconds above 0", h.Where(), raw))
				}
			}
			if h.Exec && h.Args == nil {
				evidence = append(evidence, fmt.Sprintf("%s has args that aren't a list of strings", h.Where()))
			}
		}
		return evidence
	},
}
