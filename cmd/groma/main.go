package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"runtime"

	"github.com/Maxime-Quesnel/groma/internal/agent"
	"github.com/Maxime-Quesnel/groma/internal/agent/openclaw"
	"github.com/Maxime-Quesnel/groma/internal/host"
	"github.com/Maxime-Quesnel/groma/internal/report"
	"github.com/Maxime-Quesnel/groma/internal/rules/expose"
)

var agents = []agent.Agent{openclaw.Agent}

const usage = `groma draws the perimeter around your AI agents.

Usage:
  groma expose [--host user@host]   check what this machine or a remote host exposes

Exit status: 0 when nothing is found, 1 when there are findings, 2 on error.
`

func main() {
	os.Exit(run(context.Background(), os.Args[1:], os.Stdout, os.Stderr))
}

func run(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stderr, usage)
		return 2
	}
	switch args[0] {
	case "expose":
		return runExpose(ctx, args[1:], stdout, stderr)
	case "help", "-h", "--help":
		fmt.Fprint(stdout, usage)
		return 0
	}
	fmt.Fprintf(stderr, "groma: unknown command %q\n\n%s", args[0], usage)
	return 2
}

func runExpose(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("expose", flag.ContinueOnError)
	flags.SetOutput(stderr)
	destination := flags.String("host", "", "audit a remote host over SSH, as user@host")
	if err := flags.Parse(args); err != nil {
		return 2
	}

	var runner host.Runner = host.Local{}
	if *destination != "" {
		ssh, err := host.NewSSH(*destination)
		if err != nil {
			fmt.Fprintf(stderr, "groma: %v\n", err)
			return 2
		}
		runner = ssh
	} else if runtime.GOOS != "linux" {
		fmt.Fprintln(stderr, "groma: expose audits Linux hosts. Run it on the server, or from here with --host user@host.")
		return 2
	}

	facts, err := expose.Collect(ctx, runner, agents)
	if err != nil {
		fmt.Fprintf(stderr, "groma: %v\n", err)
		return 2
	}
	findings := expose.Check(facts, expose.Rules())
	report.Text(stdout, findings)
	if len(findings) > 0 {
		return 1
	}
	return 0
}
