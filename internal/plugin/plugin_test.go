package plugin

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
	"testing/fstest"
)

func mustRead(t *testing.T, root string) Plugin {
	t.Helper()
	p, err := Read(root)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestReadSkipsGitBinariesAndSymlinks(t *testing.T) {
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
	for _, f := range mustRead(t, root).Files {
		paths = append(paths, f.Path)
	}
	if want := []string{"skills/format/SKILL.md"}; !slices.Equal(paths, want) {
		t.Errorf("read %v, want %v", paths, want)
	}
}

func TestReadFollowsASymlinkedRoot(t *testing.T) {
	plugin := t.TempDir()
	if err := os.WriteFile(filepath.Join(plugin, "SKILL.md"), []byte("Format the file."), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(t.TempDir(), "plugin")
	if err := os.Symlink(plugin, link); err != nil {
		t.Fatal(err)
	}

	if _, ok := mustRead(t, link).File("SKILL.md"); !ok {
		t.Error("SKILL.md not read through the symlinked root")
	}
}
