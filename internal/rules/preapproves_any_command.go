package rules

import (
	"regexp"

	"github.com/Maxime-Quesnel/groma/internal/component"
	"github.com/Maxime-Quesnel/groma/internal/rule"
)

var preapprovesAnyCommand = Rule{
	Meta: rule.Meta{
		ID:    "preapproves-any-command",
		Level: rule.RedFlag,
		Title: "Skill runs any shell command without asking",
		Description: "The skill or command pre-approves the shell tool with no limit on the command, or with a program that runs whatever it is given, such as a shell, an interpreter or npx. " +
			"While it runs, Claude executes any command without a permission prompt, so text that steers Claude during that turn, such as a prompt injection in a file it reads, runs code on the user's machine unchecked.",
		Remediation: "List the exact commands the skill needs, such as Bash(git status *), instead of Bash, Bash(*) or an interpreter. " +
			"Add disable-model-invocation: true so only the user can start it.",
		FalsePositives: []string{
			"A skill that genuinely needs any command, such as a shell helper the user invokes by hand; it still deserves disable-model-invocation.",
			"An interpreter pre-approved to run the plugin's own scripts, when the pattern could have named the script instead.",
		},
		References: []string{"https://code.claude.com/docs/en/skills"},
	},
	Kinds: skillsAndCommands,
	Check: func(c *component.Component, t *component.Tree) []string {
		var evidence []string
		f, _ := c.Header.Field("allowed-tools")
		for _, tool := range f.Items() {
			if anyCommand.MatchString(tool) {
				evidence = append(evidence, at(f.Line, "allowed-tools pre-approves %s", tool))
			}
		}
		return evidence
	},
}

// The shell tool alone, a pattern made only of wildcards or starting with
// one, or a program that runs arbitrary code followed by a wildcard, possibly
// after flags: Bash, Bash(*), Bash(* --help), Bash(node:*), Bash(bash -c *).
var anyCommand = regexp.MustCompile(`^(?:Bash|PowerShell)(?:\(\s*(?:[*:\s]*|\*.*|` +
	`(?:sudo|env|xargs|eval|exec|(?:ba|z|da|k)?sh|fish|pwsh|python[0-9.]*|node|deno|bun|bunx|npx|pnpx|uvx|pipx|ruby|perl|php)` +
	`(?:\s+-\S+)*(?::\*|\s*\*).*)\))?$`)
