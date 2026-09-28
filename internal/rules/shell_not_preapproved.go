package rules

import (
	"regexp"
	"slices"
	"strings"

	"github.com/Maxime-Quesnel/groma/internal/claudecode"
	"github.com/Maxime-Quesnel/groma/internal/component"
	"github.com/Maxime-Quesnel/groma/internal/rule"
)

var shellNotPreapproved = Rule{
	Meta: rule.Meta{
		ID:    "shell-not-preapproved",
		Level: rule.RedFlag,
		Title: "Inline shell that isn't pre-approved",
		Description: "Claude Code runs a skill's !`command` and ```! blocks before Claude reads the skill, and never prompts for them. " +
			"Outside auto mode, a command that the user's permission rules or the skill's allowed-tools don't allow aborts the whole invocation, " +
			"so for most users the skill fails every time with \"Shell command permission check failed\".",
		Remediation: "Pre-approve each command in allowed-tools with the narrowest pattern that fits, such as Bash(git diff *).",
		FalsePositives: []string{
			"A skill meant only for users whose own settings allow the command, or who run in auto mode.",
			"A command matched by a pattern groma doesn't read the way Claude Code does; groma treats * as any text and name:* as a prefix.",
			"A read-only command Claude Code allows on its own that groma doesn't know. groma knows ls, cat, echo, pwd, head, tail, grep, find, wc, which, diff, stat, du, cd and git's read-only subcommands, which the documentation names.",
		},
		References: []string{claudecode.SkillsDocs},
	},
	Kinds: skillsAndCommands,
	Check: func(c *component.Component, t *component.Tree) []string {
		shells := c.InlineShell()
		if len(shells) == 0 || !headerReadable(c) {
			return nil
		}
		tool := "Bash"
		if strings.EqualFold(c.Header.Value("shell"), "powershell") {
			tool = "PowerShell"
		}
		f, _ := c.Header.Field("allowed-tools")
		var patterns []string
		for _, entry := range f.Items() {
			if claudecode.Tool(entry) != tool {
				continue
			}
			inner, hasParens := strings.CutPrefix(strings.TrimSpace(entry), tool+"(")
			if !hasParens {
				return nil
			}
			patterns = append(patterns, strings.TrimSuffix(inner, ")"))
		}
		var evidence []string
		for _, s := range shells {
			if !allowed(s.Command, patterns) {
				evidence = append(evidence, at(s.Line, "%s isn't pre-approved by allowed-tools", clip(s.Command)))
			}
		}
		return evidence
	},
}

var commandSeparator = regexp.MustCompile(`&&|\|\||[;|\n]`)

// allowed reports whether patterns allow the command, either whole or each
// of its parts between &&, ||, ;, | and newlines.
func allowed(command string, patterns []string) bool {
	if matchesAny(strings.TrimSpace(command), patterns) {
		return true
	}
	for _, part := range commandSeparator.Split(command, -1) {
		if part = strings.TrimSpace(part); part != "" && !strings.HasPrefix(part, "#") && !matchesAny(part, patterns) {
			return false
		}
	}
	return true
}

func matchesAny(command string, patterns []string) bool {
	command = unquote(command)
	if readOnly(command) {
		return true
	}
	for _, p := range patterns {
		p = unquote(strings.TrimSpace(p))
		switch {
		case p == "" || p == "*":
			return true
		case strings.HasSuffix(p, ":*"):
			if strings.HasPrefix(command, strings.TrimSuffix(p, ":*")) {
				return true
			}
		default:
			glob := "^" + strings.ReplaceAll(regexp.QuoteMeta(p), `\*`, ".*") + "$"
			if regexp.MustCompile(glob).MatchString(command) || command == strings.TrimSuffix(p, " *") {
				return true
			}
		}
	}
	return false
}

func unquote(s string) string {
	return strings.NewReplacer(`"`, "", `'`, "").Replace(s)
}

// Claude Code runs a built-in set of read-only commands without a
// permission rule. Its documentation names these; the set holds more.
var (
	readOnlyCommands = []string{"ls", "cat", "echo", "pwd", "head", "tail", "grep", "find", "wc", "which", "diff", "stat", "du", "cd"}
	readOnlyGit      = []string{"status", "diff", "log", "show", "rev-parse", "ls-files", "blame", "describe", "shortlog"}
	listingGit       = regexp.MustCompile(`^git (?:branch|tag|remote)(?: (?:--show-current|-v|--verbose|-l|--list|-a|--all|-r))*$`)
	redirect         = regexp.MustCompile(`(?:^|[^2&])>`)
)

func readOnly(command string) bool {
	if redirect.MatchString(command) {
		return false
	}
	fields := strings.Fields(command)
	switch {
	case len(fields) == 0:
		return true
	case slices.Contains(readOnlyCommands, fields[0]):
		return true
	case fields[0] == "git" && len(fields) > 1 && slices.Contains(readOnlyGit, fields[1]):
		return true
	}
	return listingGit.MatchString(strings.Join(fields, " "))
}
