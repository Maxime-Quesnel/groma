package scan

import (
	"fmt"

	"github.com/Maxime-Quesnel/groma/internal/plugin"
	"github.com/Maxime-Quesnel/groma/internal/rule"
)

var hookRunsRemoteCode = Rule{
	Meta: rule.Meta{
		ID:       "scan.hook-runs-remote-code",
		Severity: rule.High,
		Title:    "Hook downloads and runs remote code",
		Description: "Hooks run on their own, at session start or around every tool call, without asking the user. " +
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
	Check: func(p plugin.Plugin) []Hit {
		var hits []Hit
		for _, h := range p.Hooks {
			for _, c := range findRemoteCode([]byte(h.Command)) {
				hits = addHit(hits, h.Source, fmt.Sprintf("%s hook runs %s", h.Event, clip(c.text)))
			}
			for _, script := range h.Scripts {
				f, _ := p.File(script)
				for _, c := range findRemoteCode(f.Content) {
					hits = addHit(hits, h.Source, fmt.Sprintf("%s hook runs %s, which at line %d runs %s", h.Event, script, c.line, clip(c.text)))
				}
			}
		}
		return hits
	},
}

const maxCommand = 100

func clip(s string) string {
	if runes := []rune(s); len(runes) > maxCommand {
		return string(runes[:maxCommand]) + "…"
	}
	return s
}
