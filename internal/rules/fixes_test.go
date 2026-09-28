package rules

import (
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/Maxime-Quesnel/groma/internal/component"
	"github.com/Maxime-Quesnel/groma/internal/fix"
)

// Every fix must leave its rule with no more evidence on each bad fixture
// than before, fully correct at least one of them, and never break a file
// it touches. Good fixtures get no edit.
func TestFixes(t *testing.T) {
	for _, r := range All {
		if r.Fix == nil {
			continue
		}
		t.Run(r.ID, func(t *testing.T) {
			fixed := 0
			cases, _ := filepath.Glob(filepath.Join("testdata", r.ID, "bad", "*"))
			for _, c := range cases {
				before := len(check(t, r, c))
				dir := copyFixture(t, c)
				applyFixes(t, r, dir)
				after := check(t, r, dir)
				if len(after) > before {
					t.Errorf("%s: %d findings after the fix, %d before: %q", filepath.Base(c), len(after), before, after)
				}
				if len(after) == 0 {
					fixed++
				}
				for _, structural := range []Rule{frontmatterUnreadable, hookFileInvalid} {
					if broken := checkAny(t, structural, dir); len(broken) > 0 {
						t.Errorf("%s: the fix broke a file: %q", filepath.Base(c), broken)
					}
				}
			}
			if fixed == 0 {
				t.Error("no bad fixture is fully fixed")
			}
			goods, _ := filepath.Glob(filepath.Join("testdata", r.ID, "good", "*"))
			for _, g := range goods {
				if edits := fixEdits(t, r, g); len(edits) > 0 {
					t.Errorf("%s: good fixture edited: %+v", filepath.Base(g), edits)
				}
			}
		})
	}
}

func fixEdits(t *testing.T, r Rule, fixture string) []fix.Edit {
	t.Helper()
	tree, err := component.Load(fixture)
	if err != nil {
		t.Fatal(err)
	}
	var edits []fix.Edit
	for _, c := range tree.Targets {
		if slices.Contains(r.Kinds, c.Kind) && len(r.Check(c, tree)) > 0 {
			edits = append(edits, r.Fix(c, tree, true)...)
		}
	}
	return edits
}

func applyFixes(t *testing.T, r Rule, dir string) {
	t.Helper()
	tree, err := component.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	read := func(p string) ([]byte, bool) {
		f, ok := tree.File(p)
		return f.Content, ok
	}
	files, _ := fix.Plan(fixEdits(t, r, dir), read)
	for _, f := range files {
		p := filepath.Join(tree.Root, filepath.FromSlash(f.Path))
		info, err := os.Stat(p)
		if err != nil {
			t.Fatal(err)
		}
		mode := info.Mode()
		if f.MakeExecutable {
			mode |= 0o111
		}
		if err := os.WriteFile(p, f.New, mode); err != nil {
			t.Fatal(err)
		}
		os.Chmod(p, mode)
	}
}

// checkAny runs r on every component of a fixture, whatever its kind.
func checkAny(t *testing.T, r Rule, dir string) []string {
	t.Helper()
	tree, err := component.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	var evidence []string
	for _, c := range tree.Targets {
		if slices.Contains(r.Kinds, c.Kind) {
			evidence = append(evidence, r.Check(c, tree)...)
		}
	}
	return evidence
}

// copyFixture copies a fixture, file or directory, into a temporary
// directory, keeping file modes, and returns the copy's path.
func copyFixture(t *testing.T, src string) string {
	t.Helper()
	dst := filepath.Join(t.TempDir(), filepath.Base(src))
	err := filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, p)
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		content, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(target, content, info.Mode())
	})
	if err != nil {
		t.Fatal(err)
	}
	return dst
}
