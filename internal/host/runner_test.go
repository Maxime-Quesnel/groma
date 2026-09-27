package host

import (
	"reflect"
	"testing"
)

func TestNewSSHRejectsOptionLikeDestinations(t *testing.T) {
	for _, dest := range []string{"", "-oProxyCommand=touch /tmp/pwned", "-p2222"} {
		if _, err := NewSSH(dest); err == nil {
			t.Errorf("NewSSH(%q) returned no error", dest)
		}
	}
}

func TestSSHQuotesTheRemoteCommand(t *testing.T) {
	s, err := NewSSH("deploy@192.0.2.10")
	if err != nil {
		t.Fatal(err)
	}

	got := s.args("stat", "-c", "%a %U", "/srv/it's here; rm -rf ~")

	want := []string{
		"-o", "BatchMode=yes",
		"-o", "ConnectTimeout=10",
		"--", "deploy@192.0.2.10",
		`'stat' '-c' '%a %U' '/srv/it'\''s here; rm -rf ~'`,
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("args =\n%q\nwant\n%q", got, want)
	}
}
