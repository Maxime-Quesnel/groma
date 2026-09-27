package expose

import (
	"fmt"
	"net/netip"
	"strings"

	"github.com/Maxime-Quesnel/groma/internal/agent"
	"github.com/Maxime-Quesnel/groma/internal/host"
	"github.com/Maxime-Quesnel/groma/internal/rule"
)

type agentPortPublic struct{}

func (agentPortPublic) Meta() rule.Meta {
	return rule.Meta{
		ID:       "expose.agent-port-public",
		Severity: rule.High,
		Title:    "Agent port reachable from outside the host",
		Description: "The agent listens on every interface or on a public address, so anyone who can reach the host can reach it. " +
			"For OpenClaw this is the gateway: the Control UI, the WebSocket that drives the agent, and HTTP APIs that have had repeated authentication bypasses. " +
			"With strong authentication and a current version an attacker still needs a second flaw, which is why this is high and not critical.",
		Remediation: "Bind the agent to loopback and reach it through an SSH tunnel or a private network such as Tailscale.",
		FalsePositives: []string{
			"A host or cloud firewall drops the port. groma doesn't read firewall rules yet; ports published by Docker bypass UFW, so for those the finding usually stands.",
			"Another program listens on one of the agent's default ports and groma can't see its owner. Running groma as root shows the owner.",
		},
		References: []string{"https://docs.openclaw.ai/gateway/security/network-exposure"},
	}
}

func (r agentPortPublic) Check(facts Facts) []rule.Finding {
	var findings []rule.Finding
	for _, a := range facts.Agents {
		var evidence []string
		for _, s := range facts.Sockets {
			if a.Owns(s) && reachableFromOutside(s.Addr) {
				evidence = append(evidence, describeSocket(a, s))
			}
		}
		if len(evidence) > 0 {
			findings = append(findings, rule.Finding{
				Rule:        r.Meta(),
				Subject:     a.Name,
				Evidence:    evidence,
				Remediation: a.BindLoopback,
			})
		}
	}
	return findings
}

// Tailscale hands out addresses from the shared address space of RFC 6598,
// which is not routable on the internet.
var sharedAddressSpace = netip.MustParsePrefix("100.64.0.0/10")

func reachableFromOutside(addr netip.Addr) bool {
	if addr.IsUnspecified() {
		return true
	}
	return addr.IsGlobalUnicast() && !addr.IsPrivate() && !sharedAddressSpace.Contains(addr)
}

func describeSocket(a agent.Agent, s host.Socket) string {
	var b strings.Builder
	b.WriteString(s.String())
	if role := a.PortRole(s.Port); role != "" {
		fmt.Fprintf(&b, ", %s", role)
	}
	if len(s.Processes) == 0 {
		b.WriteString(" (owner not visible, run as root to see it)")
		return b.String()
	}
	owners := make([]string, len(s.Processes))
	for i, p := range s.Processes {
		owners[i] = fmt.Sprintf("%s pid %d", p.Name, p.PID)
	}
	fmt.Fprintf(&b, " (%s)", strings.Join(owners, ", "))
	return b.String()
}
