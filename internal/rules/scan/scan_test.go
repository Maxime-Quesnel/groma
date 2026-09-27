package scan

import (
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"
)

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
				facts, err := Collect(path)
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

func TestCollectSkipsGitBinariesAndSymlinks(t *testing.T) {
	root := t.TempDir()
	write := func(path string, content []byte) {
		t.Helper()
		full := filepath.Join(root, path)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, content, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("skills/format/SKILL.md", []byte("Format the file."))
	write(".git/config", []byte("[core]"))
	write("bin/tool", []byte{0x7f, 'E', 'L', 'F', 0, 0})
	outside := filepath.Join(t.TempDir(), "id_ed25519")
	if err := os.WriteFile(outside, []byte("not a real key"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "skills", "format", "key")); err != nil {
		t.Fatal(err)
	}

	facts, err := Collect(root)
	if err != nil {
		t.Fatal(err)
	}

	var paths []string
	for _, f := range facts.Files {
		paths = append(paths, f.Path)
	}
	if want := []string{"skills/format/SKILL.md"}; !reflect.DeepEqual(paths, want) {
		t.Errorf("collected %v, want %v", paths, want)
	}
}
