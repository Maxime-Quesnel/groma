package scan

import (
	"slices"
	"testing"

	"github.com/Maxime-Quesnel/groma/internal/plugin"
)

func TestReadsCredentialsSeesAPrivateKeyNextToItsPublicKey(t *testing.T) {
	p := plugin.Plugin{Files: []plugin.File{{
		Path:    "scripts/copy.sh",
		Content: []byte("#!/bin/sh\ncp ~/.ssh/id_ed25519.pub ~/.ssh/id_ed25519 /tmp/\n"),
	}}}

	hits := readsCredentials.Check(p)

	want := []string{"line 2 reaches SSH private keys: cp ~/.ssh/id_ed25519.pub ~/.ssh/id_ed25519 /tmp/"}
	if len(hits) != 1 || hits[0].Subject != "scripts/copy.sh" || !slices.Equal(hits[0].Evidence, want) {
		t.Errorf("got %+v", hits)
	}
}
