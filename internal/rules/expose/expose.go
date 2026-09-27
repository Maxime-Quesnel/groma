// Package expose holds the checks behind `groma expose`.
package expose

import (
	"context"

	"github.com/Maxime-Quesnel/groma/internal/agent"
	"github.com/Maxime-Quesnel/groma/internal/host"
	"github.com/Maxime-Quesnel/groma/internal/rule"
)

// Facts is everything the rules may look at. Collecting it is the only step
// that touches the host, so rules stay pure functions and are tested against
// captured command output.
type Facts struct {
	Agents  []agent.Agent
	Sockets []host.Socket
}

type Rule interface {
	Meta() rule.Meta
	Check(Facts) []rule.Finding
}

func Rules() []Rule {
	return []Rule{
		agentPortPublic{},
	}
}

func Collect(ctx context.Context, r host.Runner, agents []agent.Agent) (Facts, error) {
	sockets, err := host.ListeningSockets(ctx, r)
	if err != nil {
		return Facts{}, err
	}
	return Facts{Agents: agents, Sockets: sockets}, nil
}

func Check(facts Facts, rules []Rule) []rule.Finding {
	var findings []rule.Finding
	for _, r := range rules {
		findings = append(findings, r.Check(facts)...)
	}
	return findings
}
