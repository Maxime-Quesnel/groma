// Package agent describes the agents groma knows, so that rules stay neutral:
// a rule asks an Agent what it runs as and where it listens, and never
// hardcodes one agent's details.
package agent

import (
	"slices"

	"github.com/Maxime-Quesnel/groma/internal/host"
)

type Agent struct {
	Name string
	// Processes are the names the agent's own processes run under, as ss and
	// ps report them. Linux cuts process names to 15 characters.
	Processes []string
	// Runtimes are generic interpreters the agent may run under, such as node.
	// They only attribute a socket that is also on one of the agent's ports.
	Runtimes []string
	Ports    []Port
	// BindLoopback tells the user how to keep this agent on loopback.
	BindLoopback string
}

type Port struct {
	Number uint16
	Role   string
}

// portProxies hold a container's published ports on the host side.
var portProxies = []string{"docker-proxy", "rootlessport"}

// Owns reports whether the socket belongs to the agent. A socket on one of the
// agent's ports whose owner groma can't see is attributed to the agent.
func (a Agent) Owns(s host.Socket) bool {
	onAgentPort := a.PortRole(s.Port) != ""
	if len(s.Processes) == 0 {
		return onAgentPort
	}
	for _, p := range s.Processes {
		if slices.Contains(a.Processes, p.Name) {
			return true
		}
		if onAgentPort && (slices.Contains(a.Runtimes, p.Name) || slices.Contains(portProxies, p.Name)) {
			return true
		}
	}
	return false
}

func (a Agent) PortRole(number uint16) string {
	for _, p := range a.Ports {
		if p.Number == number {
			return p.Role
		}
	}
	return ""
}
