package scan

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"testing/fstest"
)

// Every registered rule must be documented and have dangerous and safe
// fixtures in testdata/<ID without "scan.">.
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
				evidence := r.Check(mustCollect(t, path)[0])
				if kind == "safe" && len(evidence) > 0 {
					t.Errorf("safe fixture flagged: %q", evidence)
				}
				if kind == "dangerous" && len(evidence) == 0 {
					t.Error("dangerous fixture not flagged")
				}
			})
		}
	}
}

func mustCollect(t *testing.T, path string) []File {
	t.Helper()
	files, err := Collect(path)
	if err != nil || len(files) == 0 {
		t.Fatalf("Collect(%s) = %d files, %v", path, len(files), err)
	}
	return files
}

func TestCollectSkipsGitBinariesAndSymlinks(t *testing.T) {
	root := t.TempDir()
	err := os.CopyFS(root, fstest.MapFS{
		"skills/format/SKILL.md": {Data: []byte("Format the file.")},
		".git/config":            {Data: []byte("[core]")},
		"bin/tool":               {Data: []byte("\x7fELF\x00\x00")},
	})
	if err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "id_ed25519")
	if err := os.WriteFile(outside, []byte("not a real key"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "skills", "format", "key")); err != nil {
		t.Fatal(err)
	}

	var paths []string
	for _, f := range mustCollect(t, root) {
		paths = append(paths, f.Path)
	}
	if want := []string{"skills/format/SKILL.md"}; !slices.Equal(paths, want) {
		t.Errorf("collected %v, want %v", paths, want)
	}
}

func TestCollectFollowsASymlinkedRoot(t *testing.T) {
	plugin := t.TempDir()
	if err := os.WriteFile(filepath.Join(plugin, "SKILL.md"), []byte("Format the file."), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(t.TempDir(), "plugin")
	if err := os.Symlink(plugin, link); err != nil {
		t.Fatal(err)
	}

	if files := mustCollect(t, link); files[0].Path != "SKILL.md" {
		t.Errorf("collected %v", files)
	}
}
