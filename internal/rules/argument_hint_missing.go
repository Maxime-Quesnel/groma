package rules

import (
	"regexp"
	"strings"

	"github.com/Maxime-Quesnel/groma/internal/claudecode"
	"github.com/Maxime-Quesnel/groma/internal/component"
	"github.com/Maxime-Quesnel/groma/internal/rule"
)

var argumentHintMissing = Rule{
	Meta: rule.Meta{
		ID:    "argument-hint-missing",
		Level: rule.Warning,
		Title: "Takes arguments without saying which",
		Description: "The body uses $ARGUMENTS, $ARGUMENTS[N] or $N, but no argument-hint tells the user what to type, " +
			"so the slash menu shows nothing and the user has to open the file to find out.",
		Remediation: "Add an argument-hint such as [issue-number] or [file] [format], or name the arguments with the arguments field.",
		FalsePositives: []string{
			"A $1 meant for the shell or awk in a code sample; Claude Code replaces it with an argument anyway, so it deserves a look too.",
		},
		References: []string{claudecode.SkillsDocs},
	},
	Kinds: skillsAndCommands,
	Check: func(c *component.Component, t *component.Tree) []string {
		if !headerReadable(c) || c.Header.Has("argument-hint") || c.Header.Has("arguments") {
			return nil
		}
		for i, line := range strings.Split(c.Header.Body, "\n") {
			if m := positional.FindString(line); m != "" {
				return []string{at(c.Header.BodyLine+i, "uses %s", m)}
			}
		}
		return nil
	},
}

var positional = regexp.MustCompile(`\$ARGUMENTS(?:\[\d+\])?|\$\d+\b`)
