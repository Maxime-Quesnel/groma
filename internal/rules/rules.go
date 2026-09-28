// Package rules holds groma's checks, one per file. Each checks some kinds
// of component and returns its evidence, one line per problem found.
package rules

import (
	"fmt"
	"slices"
	"strings"

	"github.com/Maxime-Quesnel/groma/internal/component"
	"github.com/Maxime-Quesnel/groma/internal/rule"
)

type Rule struct {
	rule.Meta
	Kinds []component.Kind
	Check func(c *component.Component, t *component.Tree) []string
}

var All = []Rule{
	// Red flags: security risks.
	hiddenUnicode,
	runsRemoteCode,
	readsCredentials,
	preapprovesAnyCommand,
	argumentsInShell,
	promptInjection,
	bypassPermissions,
	// Red flags: what Claude Code won't load, won't run or ignores.
	frontmatterUnreadable,
	agentNotLoaded,
	fieldTypo,
	invalidValue,
	fieldIgnored,
	unknownTool,
	toolUnavailableToAgents,
	duplicateName,
	shellNotPreapproved,
	missingScript,
	hookFileInvalid,
	hookUnknownEvent,
	hookHandlerInvalid,
	hookFieldIgnored,
	hookNeverRuns,
	hookMatcherIgnored,
	hookUserConfigInShell,
	// Warnings.
	unknownField,
	nameFormat,
	descriptionMissing,
	descriptionTooLong,
	descriptionNoTrigger,
	descriptionVoice,
	bodyEmpty,
	bodyTooLong,
	brokenLink,
	nestedReference,
	nonPortablePath,
	agentToolsUnrestricted,
	argumentHintMissing,
	sideEffectsModelInvocable,
	unquotedPath,
	stopHookCanLoop,
}

func Check(t *component.Tree) []rule.Finding {
	var findings []rule.Finding
	for _, c := range t.Components {
		for _, r := range All {
			if !slices.Contains(r.Kinds, c.Kind) {
				continue
			}
			if evidence := r.Check(c, t); len(evidence) > 0 {
				findings = append(findings, rule.Finding{Rule: r.Meta, Path: c.Path, Evidence: evidence})
			}
		}
	}
	return findings
}

var (
	markdown          = []component.Kind{component.Skill, component.Agent, component.Command}
	skillsAndCommands = []component.Kind{component.Skill, component.Command}
	skillsAndAgents   = []component.Kind{component.Skill, component.Agent}
	agents            = []component.Kind{component.Agent}
	skills            = []component.Kind{component.Skill}
	hooks             = []component.Kind{component.Hooks}
	everything        = []component.Kind{component.Skill, component.Agent, component.Command, component.Hooks}
)

func at(line int, format string, a ...any) string {
	return fmt.Sprintf("line %d: ", line) + fmt.Sprintf(format, a...)
}

// fieldAt returns the evidence for a frontmatter field: its line and text.
func fieldAt(c *component.Component, key, format string, a ...any) string {
	f, _ := c.Header.Field(key)
	return at(f.Line, format, a...)
}

// headerReadable reports whether the frontmatter parsed, so that checks of
// its fields don't repeat what frontmatter-unreadable says.
func headerReadable(c *component.Component) bool {
	return len(c.Header.Problems) == 0 && !c.Header.Misplaced
}

// A command is a line that runs: a hook's command line or a skill's inline
// shell, and where the component declares it.
type command struct {
	where, run string
	exec       bool
}

func commands(c *component.Component) []command {
	var cmds []command
	for _, h := range c.Hooks.Handlers {
		if h.Type == "command" || h.Type == "" {
			cmds = append(cmds, command{where: h.Where(), run: h.Run(), exec: h.Exec})
		}
	}
	for _, s := range c.InlineShell() {
		cmds = append(cmds, command{where: fmt.Sprintf("line %d, inline shell", s.Line), run: s.Command})
	}
	return cmds
}

// attached returns the files a component is made of and the scripts its
// commands run.
func attached(c *component.Component, t *component.Tree) []component.File {
	files := t.Owned(c)
	for _, cmd := range commands(c) {
		for _, s := range t.Scripts(cmd.run, c.Roots()) {
			if f, ok := t.File(s.Path); ok && !slices.ContainsFunc(files, func(o component.File) bool { return o.Path == f.Path }) {
				files = append(files, f)
			}
		}
	}
	return files
}

// in prefixes evidence from a file other than the component's own.
func in(c *component.Component, f component.File, evidence string) string {
	if f.Path == c.Path {
		return evidence
	}
	return f.Path + ", " + evidence
}

const maxQuoted = 100

func clip(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	if runes := []rune(s); len(runes) > maxQuoted {
		return string(runes[:maxQuoted]) + "…"
	}
	return s
}

// suggest returns the candidate closest to s, when one is close enough to
// be what s meant.
func suggest(s string, candidates []string) string {
	norm := func(x string) string { return strings.ToLower(strings.NewReplacer("-", "", "_", "").Replace(x)) }
	for _, c := range candidates {
		if norm(c) == norm(s) {
			return c
		}
	}
	best, bestDistance := "", 3
	for _, c := range candidates {
		if d := distance(strings.ToLower(s), strings.ToLower(c)); d < bestDistance && len(s) >= 5 {
			best, bestDistance = c, d
		}
	}
	return best
}

// distance is the Levenshtein distance between a and b.
func distance(a, b string) int {
	prev := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		cur := make([]int, len(b)+1)
		cur[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			cur[j] = min(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
		}
		prev = cur
	}
	return prev[len(b)]
}
