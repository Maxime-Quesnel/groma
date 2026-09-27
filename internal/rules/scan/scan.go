package scan

import (
	"slices"

	"github.com/Maxime-Quesnel/groma/internal/plugin"
	"github.com/Maxime-Quesnel/groma/internal/rule"
)

type Rule struct {
	rule.Meta
	Check func(plugin.Plugin) []Hit
}

type Hit struct {
	Subject  string
	Evidence []string
}

var Rules = []Rule{
	hiddenUnicode,
	hookRunsRemoteCode,
	preapprovesAnyCommand,
	readsCredentials,
}

func Check(p plugin.Plugin) []rule.Finding {
	var findings []rule.Finding
	for _, r := range Rules {
		for _, h := range r.Check(p) {
			findings = append(findings, rule.Finding{Rule: r.Meta, Subject: h.Subject, Evidence: h.Evidence})
		}
	}
	return findings
}

// eachFile turns a check of one file into a rule check, with the file as the
// subject of its findings.
func eachFile(check func(plugin.File) []string) func(plugin.Plugin) []Hit {
	return func(p plugin.Plugin) []Hit {
		var hits []Hit
		for _, f := range p.Files {
			if evidence := check(f); len(evidence) > 0 {
				hits = append(hits, Hit{Subject: f.Path, Evidence: evidence})
			}
		}
		return hits
	}
}

// addHit files evidence under its subject, once.
func addHit(hits []Hit, subject, evidence string) []Hit {
	i := slices.IndexFunc(hits, func(h Hit) bool { return h.Subject == subject })
	if i < 0 {
		return append(hits, Hit{Subject: subject, Evidence: []string{evidence}})
	}
	if !slices.Contains(hits[i].Evidence, evidence) {
		hits[i].Evidence = append(hits[i].Evidence, evidence)
	}
	return hits
}
