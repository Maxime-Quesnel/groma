package rules

import (
	"fmt"
	"regexp"

	"github.com/Maxime-Quesnel/groma/internal/claudecode"
	"github.com/Maxime-Quesnel/groma/internal/component"
	"github.com/Maxime-Quesnel/groma/internal/fix"
	"github.com/Maxime-Quesnel/groma/internal/rule"
	"strings"
)

var unquotedPath = Rule{
	Meta: rule.Meta{
		ID:    "unquoted-path",
		Level: rule.Warning,
		Title: "Directory variable left unquoted in a shell command",
		Description: "The shell splits an unquoted ${CLAUDE_PLUGIN_ROOT}, ${CLAUDE_PROJECT_DIR} or ${CLAUDE_SKILL_DIR} at spaces, " +
			"so the command breaks for every user whose home, project or plugin path holds one. Claude Code's documentation quotes them.",
		Remediation: `Wrap the variable in double quotes, as in "${CLAUDE_PLUGIN_ROOT}"/scripts/check.sh, or use the exec form with args, which needs no quoting.`,
		FalsePositives: []string{
			"A command only ever run where the path has no spaces.",
		},
		References: []string{claudecode.HooksDocs},
	},
	Kinds: []component.Kind{component.Hooks, component.Skill, component.Command},
	Check: func(c *component.Component, t *component.Tree) []string {
		var evidence []string
		for _, cmd := range commands(c) {
			if cmd.exec {
				continue
			}
			if v := unquotedVariable(cmd.run); v != "" {
				evidence = append(evidence, fmt.Sprintf("%s: %s is unquoted in %s", cmd.where, v, clip(cmd.run)))
			}
		}
		return evidence
	},
	Fix: fixUnquotedPaths,
}

var directoryVariable = regexp.MustCompile(`\$\{?CLAUDE_(?:PLUGIN_ROOT|PROJECT_DIR|SKILL_DIR|PLUGIN_DATA)\}?`)

// unquotedVariable returns the first directory variable of command that sits
// outside double quotes. Single quotes keep the shell from expanding it at
// all, which is a different mistake.
func unquotedVariable(command string) string {
	quoted := make([]bool, len(command))
	inDouble, inSingle := false, false
	for i := 0; i < len(command); i++ {
		switch c := command[i]; {
		case c == '\\' && !inSingle:
			i++
			continue
		case c == '"' && !inSingle:
			inDouble = !inDouble
		case c == '\'' && !inDouble:
			inSingle = !inSingle
		}
		quoted[i] = inDouble || inSingle
	}
	for _, m := range directoryVariable.FindAllStringIndex(command, -1) {
		if !quoted[m[0]] {
			return command[m[0]:m[1]]
		}
	}
	return ""
}

// fixUnquotedPaths wraps each unquoted path built on a directory variable
// in double quotes. Where the path has no spaces, the command runs the same.
func fixUnquotedPaths(c *component.Component, t *component.Tree, unsafe bool) []fix.Edit {
	var edits []fix.Edit
	for _, h := range c.Hooks.Handlers {
		if h.Exec || h.Type != "command" && h.Type != "" {
			continue
		}
		if quoted, ok := quoteVariables(h.Command); ok {
			edits = append(edits, jsonValueEdits(c, "command", jsonString(h.Command), jsonString(quoted))...)
		}
	}
	for _, s := range c.InlineShell() {
		for i, line := range strings.Split(s.Command, "\n") {
			if quoted, ok := quoteVariables(line); ok {
				if e, found := replaceInLine(c, s.Line+i, line, quoted); found {
					edits = append(edits, e)
				}
			}
		}
	}
	return edits
}

// quoteVariables wraps each word of command that holds an unquoted
// directory variable in double quotes. It leaves the command alone when such
// a word has globs, quotes or substitutions, which quoting would change.
func quoteVariables(command string) (string, bool) {
	if unquotedVariable(command) == "" {
		return command, false
	}
	quoted := make([]bool, len(command))
	inDouble, inSingle := false, false
	for i := 0; i < len(command); i++ {
		switch c := command[i]; {
		case c == '\\' && !inSingle:
			i++
			continue
		case c == '"' && !inSingle:
			inDouble = !inDouble
		case c == '\'' && !inDouble:
			inSingle = !inSingle
		}
		quoted[i] = inDouble || inSingle
	}
	matches := directoryVariable.FindAllStringIndex(command, -1)
	out := command
	for i := len(matches) - 1; i >= 0; i-- {
		m := matches[i]
		if quoted[m[0]] {
			continue
		}
		start := strings.LastIndexAny(command[:m[0]], " \t;&|(=") + 1
		end := m[1] + strings.IndexAny(command[m[1]:]+" ", " \t;&|)")
		word := command[start:end]
		rest := strings.Replace(word, command[m[0]:m[1]], "", 1)
		if strings.ContainsAny(rest, "*?[]\"'`$") {
			return command, false
		}
		out = out[:start] + `"` + word + `"` + out[end:]
	}
	return out, out != command
}
