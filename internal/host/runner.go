// Package host collects facts from the machine being audited, locally or over
// SSH. Every command groma runs on a host is defined in this package, and every
// one of them must be read-only: add new probes here and nowhere else.
package host

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

type Runner interface {
	Output(ctx context.Context, name string, args ...string) ([]byte, error)
}

type Local struct{}

func (Local) Output(ctx context.Context, name string, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, name, args...).Output()
}

// SSH runs commands through the system ssh client, so the user's ssh config,
// agent and jump hosts apply, and groma never handles private keys.
type SSH struct {
	destination string
}

func NewSSH(destination string) (SSH, error) {
	// A destination starting with "-" would be parsed by ssh as an option,
	// such as -oProxyCommand, which runs a local command.
	if destination == "" || strings.HasPrefix(destination, "-") {
		return SSH{}, fmt.Errorf("invalid SSH destination %q", destination)
	}
	return SSH{destination: destination}, nil
}

func (s SSH) Output(ctx context.Context, name string, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, "ssh", s.args(name, args...)...).Output()
}

func (s SSH) args(name string, args ...string) []string {
	remote := []string{shellQuote(name)}
	for _, a := range args {
		remote = append(remote, shellQuote(a))
	}
	return []string{
		"-o", "BatchMode=yes",
		"-o", "ConnectTimeout=10",
		"--", s.destination,
		strings.Join(remote, " "),
	}
}

// ssh hands the remote command to the remote user's shell as one string.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

func run(ctx context.Context, r Runner, name string, args ...string) ([]byte, error) {
	out, err := r.Output(ctx, name, args...)
	if err == nil {
		return out, nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) && len(exitErr.Stderr) > 0 {
		return nil, fmt.Errorf("%s: %w: %s", name, err, bytes.TrimSpace(exitErr.Stderr))
	}
	return nil, fmt.Errorf("%s: %w", name, err)
}
