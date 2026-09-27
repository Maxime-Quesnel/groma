// Package openclaw describes the OpenClaw agent.
package openclaw

import "github.com/Maxime-Quesnel/groma/internal/agent"

var Agent = agent.Agent{
	Name:      "OpenClaw",
	Processes: []string{"openclaw", "openclaw-gateway", "openclaw-gatewa"},
	Runtimes:  []string{"node"},
	Ports: []agent.Port{
		{Number: 18789, Role: "gateway: Control UI, WebSocket and HTTP APIs"},
		{Number: 18790, Role: "MCP Apps sandbox"},
		{Number: 18791, Role: "browser control"},
	},
	BindLoopback: `On a host install, set gateway.bind to "loopback" and remove any --bind lan from the systemd unit. ` +
		`In Docker, keep the container's bind and publish the port as 127.0.0.1:18789:18789. ` +
		`Reach the gateway through an SSH tunnel or Tailscale Serve.`,
}
