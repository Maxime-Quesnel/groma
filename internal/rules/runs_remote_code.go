package rules

import (
	"fmt"

	"github.com/Maxime-Quesnel/groma/internal/component"
	"github.com/Maxime-Quesnel/groma/internal/rule"
)

var runsRemoteCode = Rule{
	Meta: rule.Meta{
		ID:    "runs-remote-code",
		Level: rule.RedFlag,
		Title: "Downloads code and runs it",
		Description: "Hooks run on their own, at session start or around every tool call, and so does the inline shell of skills and commands, before Claude reads them; none of it asks the user. " +
			"This one downloads code and executes it, so whoever controls the URL runs commands on the user's machine with their permissions, " +
			"and can change what runs at any time after the plugin was reviewed.",
		Remediation: "Ship the code inside the plugin, or pin the download to a checksum and verify it before running it. Never pipe a download into a shell.",
		FalsePositives: []string{
			"A download verified by a tool groma doesn't know, rather than sha256sum, shasum, gpg, cosign or minisign.",
			"A URL that points at a local service rather than the internet.",
			"A matching line in a heredoc or a string the script never runs. Comment lines starting with # or // are skipped.",
		},
		References: []string{"https://code.claude.com/docs/en/hooks"},
	},
	Kinds: []component.Kind{component.Hooks, component.Skill, component.Command},
	Check: func(c *component.Component, t *component.Tree) []string {
		var evidence []string
		for _, cmd := range commands(c) {
			for _, rc := range findRemoteCode([]byte(cmd.run)) {
				evidence = append(evidence, fmt.Sprintf("%s runs %s", cmd.where, clip(rc.text)))
			}
			for _, s := range t.Scripts(cmd.run, c.Roots()) {
				f, ok := t.File(s.Path)
				if !ok {
					continue
				}
				for _, rc := range findRemoteCode(f.Content) {
					evidence = append(evidence, fmt.Sprintf("%s runs %s, which at line %d runs %s", cmd.where, s.Path, rc.line, clip(rc.text)))
				}
			}
		}
		return evidence
	},
}
