package rules

import (
	"encoding/json"
	"fmt"
	"maps"
	"regexp"
	"slices"

	"github.com/Maxime-Quesnel/groma/internal/claudecode"
	"github.com/Maxime-Quesnel/groma/internal/component"
	"github.com/Maxime-Quesnel/groma/internal/rule"
)

var hookFieldIgnored = Rule{
	Meta: rule.Meta{
		ID:    "hook-field-ignored",
		Level: rule.RedFlag,
		Title: "Hook setting Claude Code ignores",
		Description: "Claude Code ignores these fields where they stand: a matcher written inside a hook rather than on its group, " +
			"once outside a skill's frontmatter, shell on an exec-form hook, or a header variable missing from allowedEnvVars, which Claude Code sends empty. " +
			"The hook then runs more often, or differently, than its author intended.",
		Remediation: "Move the field where Claude Code reads it, or remove it. An http hook must list in allowedEnvVars every variable its headers use.",
		FalsePositives: []string{
			"None known. Fields the documentation doesn't list at all are unknown-field warnings.",
		},
		References: []string{claudecode.HooksDocs},
	},
	Kinds: hooks,
	Check: func(c *component.Component, t *component.Tree) []string {
		var evidence []string
		ignored := func(where, format string, a ...any) {
			evidence = append(evidence, where+": "+fmt.Sprintf(format, a...))
		}
		for _, h := range c.Hooks.Handlers {
			for _, key := range h.Keys {
				switch {
				case key == "matcher":
					ignored(h.Where(), "matcher belongs on the group, next to hooks; inside a hook Claude Code ignores it")
				case key == "once":
					ignored(h.Where(), "once only works in a skill's frontmatter")
				case key == "shell" && h.Exec:
					ignored(h.Where(), "shell is ignored when args is set")
				}
			}
			if h.Type == "http" {
				var allowed []string
				json.Unmarshal(h.Fields["allowedEnvVars"], &allowed)
				var headers map[string]string
				json.Unmarshal(h.Fields["headers"], &headers)
				for _, name := range slices.Sorted(maps.Keys(headers)) {
					for _, m := range envVar.FindAllStringSubmatch(headers[name], -1) {
						if v := m[1] + m[2]; !slices.Contains(allowed, v) {
							ignored(h.Where(), "header %s uses $%s, which allowedEnvVars doesn't list, so Claude Code sends it empty", name, v)
						}
					}
				}
			}
		}
		return evidence
	},
}

var envVar = regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)\}|\$([A-Za-z_][A-Za-z0-9_]*)`)
