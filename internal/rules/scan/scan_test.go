package scan

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/Maxime-Quesnel/groma/internal/agent/claudecode"
	"github.com/Maxime-Quesnel/groma/internal/plugin"
)

// Every registered rule must be documented and have dangerous and safe
// fixtures in testdata/<ID without "scan.">. A fixture is a file or a whole
// plugin directory.
func TestRules(t *testing.T) {
	seen := map[string]bool{}
	for _, r := range Rules {
		t.Run(r.ID, func(t *testing.T) {
			if r.ID == "" || r.Severity == 0 || r.Title == "" || r.Description == "" || r.Remediation == "" || len(r.FalsePositives) == 0 {
				t.Error("missing metadata")
			}
			if seen[r.ID] {
				t.Error("ID used twice")
			}
			seen[r.ID] = true
			checkFixtures(t, r, strings.TrimPrefix(r.ID, "scan."))
		})
	}
}

func checkFixtures(t *testing.T, r Rule, dir string) {
	for _, kind := range []string{"dangerous", "safe"} {
		paths, _ := filepath.Glob(filepath.Join("testdata", dir, kind, "*"))
		if len(paths) == 0 {
			t.Fatalf("no %s fixtures in testdata/%s", kind, dir)
		}
		for _, path := range paths {
			t.Run(kind+"/"+filepath.Base(path), func(t *testing.T) {
				hits := r.Check(load(t, path))
				if kind == "safe" && len(hits) > 0 {
					t.Errorf("safe fixture flagged: %+v", hits)
				}
				if kind == "dangerous" && len(hits) == 0 {
					t.Error("dangerous fixture not flagged")
				}
			})
		}
	}
}

func load(t *testing.T, path string) plugin.Plugin {
	t.Helper()
	p, err := plugin.Read(path)
	if err != nil || len(p.Files) == 0 {
		t.Fatalf("plugin.Read(%s) = %d files, %v", path, len(p.Files), err)
	}
	p.Hooks = claudecode.Hooks(p)
	return p
}
