package rules

import (
	"slices"
	"strconv"
	"strings"

	"github.com/Maxime-Quesnel/groma/internal/claudecode"
	"github.com/Maxime-Quesnel/groma/internal/component"
	"github.com/Maxime-Quesnel/groma/internal/rule"
)

var invalidValue = Rule{
	Meta: rule.Meta{
		ID:    "invalid-value",
		Level: rule.RedFlag,
		Title: "Frontmatter value Claude Code doesn't accept",
		Description: "The field takes one of a fixed set of values, and this isn't one of them: a model Claude Code doesn't know, an effort level, a colour, " +
			"a permission mode or a yes/no field spelled another way. Claude Code ignores the value, so the component runs with the default the author meant to change.",
		Remediation: "Use one of the values the evidence lists.",
		FalsePositives: []string{
			"A value added by a Claude Code release newer than groma's lists.",
		},
		References: []string{claudecode.SkillsDocs, claudecode.SubagentsDocs},
	},
	Kinds: markdown,
	Check: func(c *component.Component, t *component.Tree) []string {
		if !headerReadable(c) {
			return nil
		}
		checks := skillValues
		if c.Kind == component.Agent {
			checks = agentValues
		}
		var evidence []string
		for _, v := range checks {
			f, ok := c.Header.Field(v.key)
			if !ok {
				continue
			}
			if f.IsList || f.IsMap || !v.valid(strings.TrimSpace(f.Value)) {
				evidence = append(evidence, at(f.Line, "%s: %s isn't %s", v.key, shown(f.Value, f.IsList || f.IsMap), v.allowed))
			}
		}
		return evidence
	},
}

type valueCheck struct {
	key     string
	valid   func(string) bool
	allowed string
}

func oneOf(values []string) func(string) bool {
	return func(s string) bool { return slices.Contains(values, s) }
}

func listed(values []string) string {
	return strings.Join(values[:len(values)-1], ", ") + " or " + values[len(values)-1]
}

var (
	modelValue  = valueCheck{"model", claudecode.Model, "a model alias such as sonnet, opus, haiku or inherit, or a claude-… model ID"}
	effortValue = valueCheck{"effort", oneOf(claudecode.Effort), listed(claudecode.Effort)}
	skillValues = []valueCheck{
		modelValue, effortValue,
		{"context", oneOf(claudecode.Context), "fork"},
		{"shell", oneOf(claudecode.Shells), listed(claudecode.Shells)},
		yesOrNo("disable-model-invocation"), yesOrNo("user-invocable"), yesOrNo("background"),
	}
	agentValues = []valueCheck{
		modelValue, effortValue,
		{"color", oneOf(claudecode.Colors), listed(claudecode.Colors)},
		{"permissionMode", oneOf(claudecode.PermissionModes), listed(claudecode.PermissionModes)},
		{"memory", oneOf(claudecode.Memory), listed(claudecode.Memory)},
		{"isolation", oneOf(claudecode.Isolation), "worktree"},
		{"maxTurns", positive, "a whole number above 0"},
		yesOrNo("background"), yesOrNo("omitClaudeMd"),
	}
)

func yesOrNo(key string) valueCheck { return valueCheck{key, claudecode.Bool, "true or false"} }

func positive(s string) bool {
	n, err := strconv.Atoi(s)
	return err == nil && n > 0
}

func shown(value string, structured bool) string {
	if structured {
		return "a list or map"
	}
	if value == "" {
		return "an empty value"
	}
	return strconv.Quote(value)
}
