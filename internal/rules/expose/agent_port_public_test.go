package expose

import (
	"context"
	"os"
	"reflect"
	"testing"

	"github.com/Maxime-Quesnel/groma/internal/agent"
	"github.com/Maxime-Quesnel/groma/internal/agent/openclaw"
)

func TestAgentPortPublic(t *testing.T) {
	checkFixtures(t, agentPortPublic{}, "agent-port-public")
}

func TestAgentPortPublicGroupsSocketsPerAgent(t *testing.T) {
	out, err := os.ReadFile("testdata/agent-port-public/dangerous/docker-published-all-interfaces.txt")
	if err != nil {
		t.Fatal(err)
	}
	facts, err := Collect(context.Background(), capturedOutput(out), []agent.Agent{openclaw.Agent})
	if err != nil {
		t.Fatal(err)
	}

	findings := agentPortPublic{}.Check(facts)

	if len(findings) != 1 {
		t.Fatalf("got %d findings, want 1", len(findings))
	}
	f := findings[0]
	wantEvidence := []string{
		"0.0.0.0:18789, gateway: Control UI, WebSocket and HTTP APIs (docker-proxy pid 3101)",
		"[::]:18789, gateway: Control UI, WebSocket and HTTP APIs (docker-proxy pid 3108)",
		"0.0.0.0:18790, MCP Apps sandbox (docker-proxy pid 3115)",
	}
	if f.Subject != "OpenClaw" || !reflect.DeepEqual(f.Evidence, wantEvidence) {
		t.Errorf("got subject %q evidence\n%q\nwant OpenClaw\n%q", f.Subject, f.Evidence, wantEvidence)
	}
	if f.Fix() != openclaw.Agent.BindLoopback {
		t.Errorf("fix = %q, want the OpenClaw steps", f.Fix())
	}
}
