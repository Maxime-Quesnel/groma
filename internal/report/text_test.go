package report

import (
	"strings"
	"testing"

	"github.com/Maxime-Quesnel/groma/internal/rule"
)

func TestTextSortsBySeverityAndSummarises(t *testing.T) {
	high := rule.Meta{ID: "expose.b", Severity: rule.High, Title: "B is open", Remediation: "close B"}
	critical := rule.Meta{ID: "expose.a", Severity: rule.Critical, Title: "A is open", Remediation: "close A"}
	var out strings.Builder

	Text(&out, []rule.Finding{
		{Rule: high, Evidence: []string{"0.0.0.0:2"}},
		{Rule: critical, Evidence: []string{"0.0.0.0:1"}},
		{Rule: high, Subject: "OpenClaw", Evidence: []string{"0.0.0.0:3"}, Remediation: "close B in OpenClaw"},
	})

	want := `CRITICAL  expose.a
          A is open
          - 0.0.0.0:1
          fix: close A

HIGH      expose.b
          B is open
          - 0.0.0.0:2
          fix: close B

HIGH      expose.b  OpenClaw
          B is open
          - 0.0.0.0:3
          fix: close B in OpenClaw

3 findings: 1 critical, 2 high
`
	if out.String() != want {
		t.Errorf("got:\n%s\nwant:\n%s", out.String(), want)
	}
}

func TestTextWithoutFindings(t *testing.T) {
	var out strings.Builder

	Text(&out, nil)

	if out.String() != "No findings.\n" {
		t.Errorf("got %q", out.String())
	}
}
