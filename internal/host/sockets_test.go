package host

import (
	"context"
	"os"
	"reflect"
	"testing"
)

type fakeRunner struct {
	out   []byte
	calls [][]string
}

func (f *fakeRunner) Output(_ context.Context, name string, args ...string) ([]byte, error) {
	f.calls = append(f.calls, append([]string{name}, args...))
	return f.out, nil
}

func TestListeningSockets(t *testing.T) {
	out, err := os.ReadFile("testdata/ss-tlnp.txt")
	if err != nil {
		t.Fatal(err)
	}
	runner := &fakeRunner{out: out}

	sockets, err := ListeningSockets(context.Background(), runner)
	if err != nil {
		t.Fatal(err)
	}

	type want struct {
		addr          string
		loopback, all bool
		processes     []Process
	}
	wants := []want{
		{"127.0.0.53:53", true, false, []Process{{"systemd-resolve", 512}}},
		{"0.0.0.0:22", false, true, []Process{{"sshd", 801}, {"systemd", 1}}},
		{"127.0.0.1:3000", true, false, []Process{{"node", 2204}}},
		{"[::]:22", false, true, []Process{{"sshd", 801}, {"systemd", 1}}},
		{"[::1]:3000", true, false, []Process{{"node", 2204}}},
		{"[::]:9100", false, true, nil},
		{"127.0.0.1:8080", true, false, []Process{{"java", 3000}}},
		{"[fe80::1]:123", false, false, nil},
		{"[::]:5432", false, true, nil},
	}
	if len(sockets) != len(wants) {
		t.Fatalf("got %d sockets, want %d: %v", len(sockets), len(wants), sockets)
	}
	for i, w := range wants {
		s := sockets[i]
		if s.String() != w.addr || s.Loopback() != w.loopback || s.AllInterfaces() != w.all {
			t.Errorf("socket %d = %s loopback=%v all=%v, want %s loopback=%v all=%v",
				i, s, s.Loopback(), s.AllInterfaces(), w.addr, w.loopback, w.all)
		}
		if !reflect.DeepEqual(s.Processes, w.processes) {
			t.Errorf("socket %s processes = %v, want %v", s, s.Processes, w.processes)
		}
	}

	wantCall := []string{"ss", "-H", "-t", "-l", "-n", "-p"}
	if len(runner.calls) != 1 || !reflect.DeepEqual(runner.calls[0], wantCall) {
		t.Errorf("calls = %v, want [%v]", runner.calls, wantCall)
	}
}

func TestParseSSRejectsMalformedLines(t *testing.T) {
	for _, line := range []string{
		"LISTEN 0 128",
		"LISTEN 0 128 0.0.0.0 0.0.0.0:*",
		"LISTEN 0 128 0.0.0.0:http 0.0.0.0:*",
		"LISTEN 0 128 not-an-ip:22 0.0.0.0:*",
	} {
		if _, err := parseSS(line); err == nil {
			t.Errorf("parseSS(%q) returned no error", line)
		}
	}
}
