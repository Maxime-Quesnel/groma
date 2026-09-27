package expose

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/Maxime-Quesnel/groma/internal/agent"
	"github.com/Maxime-Quesnel/groma/internal/agent/openclaw"
)

type capturedOutput []byte

func (c capturedOutput) Output(context.Context, string, ...string) ([]byte, error) {
	return c, nil
}

// checkFixtures runs a rule against every file in testdata/<dir>/dangerous,
// which must each yield a finding, and testdata/<dir>/safe, which must not.
func checkFixtures(t *testing.T, r Rule, dir string) {
	t.Helper()
	meta := r.Meta()
	for _, kind := range []string{"dangerous", "safe"} {
		paths, err := filepath.Glob(filepath.Join("testdata", dir, kind, "*"))
		if err != nil || len(paths) == 0 {
			t.Fatalf("no %s fixtures in testdata/%s", kind, dir)
		}
		for _, path := range paths {
			t.Run(kind+"/"+filepath.Base(path), func(t *testing.T) {
				out, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				facts, err := Collect(context.Background(), capturedOutput(out), []agent.Agent{openclaw.Agent})
				if err != nil {
					t.Fatal(err)
				}

				findings := r.Check(facts)

				if kind == "safe" {
					if len(findings) > 0 {
						t.Errorf("safe fixture flagged: %+v", findings)
					}
					return
				}
				if len(findings) == 0 {
					t.Fatal("dangerous fixture not flagged")
				}
				for _, f := range findings {
					if f.Rule.ID != meta.ID || f.Rule.Severity != meta.Severity {
						t.Errorf("finding %s/%s, want %s/%s", f.Rule.ID, f.Rule.Severity, meta.ID, meta.Severity)
					}
				}
			})
		}
	}
}

func TestEveryRuleIsDocumented(t *testing.T) {
	var ids []string
	for _, r := range Rules() {
		m := r.Meta()
		if m.ID == "" || m.Severity == 0 || m.Title == "" || m.Description == "" || m.Remediation == "" {
			t.Errorf("rule %q is missing metadata", m.ID)
		}
		if len(m.FalsePositives) == 0 {
			t.Errorf("rule %q documents no false positives", m.ID)
		}
		if slices.Contains(ids, m.ID) {
			t.Errorf("rule ID %q is used twice", m.ID)
		}
		ids = append(ids, m.ID)
	}
}
