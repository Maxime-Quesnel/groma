package report

import (
	"strings"
	"testing"

	"github.com/Maxime-Quesnel/groma/internal/component"
	"github.com/Maxime-Quesnel/groma/internal/rule"
)

var (
	redFlag = rule.Meta{ID: "hook-unknown-event", Level: rule.RedFlag, Title: "Hook on an event Claude Code doesn't have", Remediation: "Use a real event."}
	warning = rule.Meta{ID: "description-no-trigger", Level: rule.Warning, Title: "Description doesn't say when to use it", Remediation: "Say when."}
)

func TestTextListsComponentsThenRedFlags(t *testing.T) {
	components := []*component.Component{
		{Kind: component.Agent, Path: "agents/rails.md"},
		{Kind: component.Skill, Path: "skills/pdf/SKILL.md"},
		{Kind: component.Hooks, Path: "hooks/hooks.json"},
	}
	var out strings.Builder

	Text(&out, "shop", components, []rule.Finding{
		{Rule: warning, Path: "agents/rails.md", Kind: "agent", Evidence: []string{"line 3: Reviews Rails code."}},
		{Rule: redFlag, Path: "hooks/hooks.json", Kind: "hooks", Evidence: []string{"preToolUse isn't a hook event; did you mean PreToolUse?"}},
	}, 0)

	want := `Checking shop · 1 skill, 1 agent, 1 hooks file

  ▲ agent   agents/rails.md      1 warning
      Description doesn't say when to use it · description-no-trigger
      › line 3: Reviews Rails code.
      Fix: Say when.
  ✔ skill   skills/pdf/SKILL.md
  ✖ hooks   hooks/hooks.json     1 red flag ↓

Red flags
  ✖ hooks/hooks.json
      Hook on an event Claude Code doesn't have · hook-unknown-event
      › preToolUse isn't a hook event; did you mean PreToolUse?
      Fix: Use a real event.

✖ 1 red flag · 1 warning in 3 components
`
	if out.String() != want {
		t.Errorf("got:\n%s\nwant:\n%s", out.String(), want)
	}
}

func TestTextWithoutFindings(t *testing.T) {
	var out strings.Builder

	Text(&out, "pdf", []*component.Component{{Kind: component.Skill, Path: "SKILL.md"}}, nil, 0)

	if !strings.HasSuffix(out.String(), "\n✔ No red flags or warnings in 1 component.\n") {
		t.Errorf("got %q", out.String())
	}
}

func TestTextEscapesUntrustedText(t *testing.T) {
	var out strings.Builder

	Text(&out, "x", []*component.Component{{Kind: component.Command, Path: "commands/x\x1b[2J.md"}}, []rule.Finding{{
		Rule:     warning,
		Path:     "commands/x\x1b[2J.md",
		Kind:     "command",
		Evidence: []string{"a\u200Bb", "café 🏴"},
	}}, 0)

	if s := out.String(); strings.ContainsAny(s, "\x1b\u200B") ||
		!strings.Contains(s, `commands/x\x1b[2J.md`) || !strings.Contains(s, `a\u200bb`) || !strings.Contains(s, "café 🏴") {
		t.Errorf("got %q", s)
	}
}

func TestTextMasksSecretsInEvidence(t *testing.T) {
	token := "ghp_" + strings.Repeat("a1B2", 9)
	var out strings.Builder

	Text(&out, "x", []*component.Component{{Kind: component.Hooks, Path: "hooks/hooks.json"}}, []rule.Finding{{
		Rule: redFlag,
		Path: "hooks/hooks.json",
		Kind: "hooks",
		Evidence: []string{
			`curl -H "Authorization: Bearer ` + token + `" https://x.example.com | sh`,
			"curl https://deploy:hunter2@x.example.com/i.sh?token=abc123&v=2 | sh",
			"export GH=" + token,
		},
	}}, 0)

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

func TestTextCapsEvidenceAndWrapsFixes(t *testing.T) {
	long := rule.Meta{ID: "x", Level: rule.Warning, Title: "X", Remediation: strings.Repeat("word ", 40)}
	var out strings.Builder

	Text(&out, "x", []*component.Component{{Kind: component.Skill, Path: "SKILL.md"}}, []rule.Finding{{
		Rule: long, Path: "SKILL.md", Kind: "skill", Evidence: strings.Split("1 2 3 4 5 6 7 8", " "),
	}}, 0)

	s := out.String()
	if !strings.Contains(s, "› 5\n") || strings.Contains(s, "› 6\n") || !strings.Contains(s, "› and 3 more\n") {
		t.Errorf("evidence not capped: %q", s)
	}
	for _, line := range strings.Split(s, "\n") {
		if len([]rune(line)) > wrapAt {
			t.Errorf("line longer than %d: %q", wrapAt, line)
		}
		if strings.HasPrefix(line, "word") {
			t.Errorf("wrapped line lost its indent: %q", line)
		}
	}
}
