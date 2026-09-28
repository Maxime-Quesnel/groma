package rules

import (
	"slices"
	"strings"

	"github.com/Maxime-Quesnel/groma/internal/claudecode"
	"github.com/Maxime-Quesnel/groma/internal/component"
	"github.com/Maxime-Quesnel/groma/internal/fix"
	"github.com/Maxime-Quesnel/groma/internal/rule"
)

var sideEffectsModelInvocable = Rule{
	Meta: rule.Meta{
		ID:    "side-effects-model-invocable",
		Level: rule.Warning,
		Title: "Claude can start a workflow with side effects on its own",
		Description: "The name says the skill or command deploys, releases, publishes, pushes, deletes or migrates, and nothing stops Claude from invoking it when it judges fit. " +
			"Claude Code's documentation recommends disable-model-invocation: true for workflows with side effects, so they only run when the user types the slash command.",
		Remediation: "Add disable-model-invocation: true, unless Claude is meant to run it unprompted.",
		FalsePositives: []string{
			"A name that only describes the topic, such as a skill explaining a deployment process without running it.",
		},
		References: []string{claudecode.SkillsDocs},
	},
	Kinds: skillsAndCommands,
	Check: func(c *component.Component, t *component.Tree) []string {
		if !headerReadable(c) || strings.EqualFold(c.Header.Value("disable-model-invocation"), "true") {
			return nil
		}
		for _, word := range strings.FieldsFunc(c.Name(), func(r rune) bool { return r == '-' || r == ':' || r == '_' }) {
			if slices.Contains(sideEffects, strings.ToLower(word)) {
				return []string{"named " + c.Name() + ", without disable-model-invocation: true"}
			}
		}
		return nil
	},
	Fix: fixManualOnly,
}

var sideEffects = []string{"deploy", "release", "publish", "ship", "push", "delete", "destroy", "drop", "migrate", "rollback", "purge", "wipe"}

// fixManualOnly adds disable-model-invocation: true, so only the user starts
// the workflow. Claude then can't, so the fix is unsafe.
func fixManualOnly(c *component.Component, t *component.Tree, unsafe bool) []fix.Edit {
	if !unsafe || !c.Header.Found || c.Header.Has("disable-model-invocation") {
		return nil
	}
	at, _ := lineRange(c.Content, c.Header.BodyLine-1, c.Header.BodyLine-1)
	return []fix.Edit{{Path: c.Path, Start: at, End: at, New: "disable-model-invocation: true\n"}}
}
