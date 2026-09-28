package rules

import (
	"path/filepath"
	"slices"
	"testing"

	"github.com/Maxime-Quesnel/groma/internal/component"
)

// Every rule must be documented, cite its source, and have fixtures in
// testdata/<ID>: under bad/, cases it must flag, and under good/,
// near-misses it must pass. A case is a component file or a directory, such
// as a plugin.
func TestRules(t *testing.T) {
	seen := map[string]bool{}
	for _, r := range All {
		t.Run(r.ID, func(t *testing.T) {
			if r.ID == "" || r.Level == 0 || r.Title == "" || r.Description == "" || r.Remediation == "" || len(r.FalsePositives) == 0 || len(r.References) == 0 || len(r.Kinds) == 0 {
				t.Error("missing metadata")
			}
			if seen[r.ID] {
				t.Error("ID used twice")
			}
			seen[r.ID] = true
			for _, kind := range []string{"bad", "good"} {
				cases, _ := filepath.Glob(filepath.Join("testdata", r.ID, kind, "*"))
				if len(cases) == 0 {
					t.Fatalf("no %s fixtures in testdata/%s", kind, r.ID)
				}
				for _, c := range cases {
					t.Run(kind+"/"+filepath.Base(c), func(t *testing.T) {
						evidence := check(t, r, c)
						if kind == "good" && len(evidence) > 0 {
							t.Errorf("good fixture flagged: %q", evidence)
						}
						if kind == "bad" && len(evidence) == 0 {
							t.Error("bad fixture not flagged")
						}
					})
				}
			}
		})
	}
}

// check runs r on the components of a fixture it applies to.
func check(t *testing.T, r Rule, fixture string) []string {
	t.Helper()
	tree, err := component.Load(fixture)
	if err != nil {
		t.Fatal(err)
	}
	var evidence []string
	checked := false
	for _, c := range tree.Targets {
		if slices.Contains(r.Kinds, c.Kind) {
			checked = true
			evidence = append(evidence, r.Check(c, tree)...)
		}
	}
	if !checked {
		t.Fatalf("the fixture has no component %s checks", r.ID)
	}
	return evidence
}
