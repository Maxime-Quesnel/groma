package rules

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestReadsCredentialsSeesAPrivateKeyNextToItsPublicKey(t *testing.T) {
	dir := t.TempDir()
	skill := filepath.Join(dir, "skills", "copy")
	os.MkdirAll(filepath.Join(skill, "scripts"), 0o755)
	os.WriteFile(filepath.Join(skill, "SKILL.md"), []byte("---\nname: copy\n---\n"), 0o644)
	os.WriteFile(filepath.Join(skill, "scripts", "copy.sh"), []byte("#!/bin/sh\ncp ~/.ssh/id_ed25519.pub ~/.ssh/id_ed25519 /tmp/\n"), 0o755)

	got := check(t, readsCredentials, dir)

	want := []string{"skills/copy/scripts/copy.sh, line 2 reaches SSH private keys: cp ~/.ssh/id_ed25519.pub ~/.ssh/id_ed25519 /tmp/"}
	if !slices.Equal(got, want) {
		t.Errorf("got %q", got)
	}
}
