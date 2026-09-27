package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/Maxime-Quesnel/groma/internal/report"
	"github.com/Maxime-Quesnel/groma/internal/rules/scan"
)

const usage = `groma draws the perimeter around your AI agents.

Usage:
  groma scan <path>   check a plugin, a marketplace or a skill directory

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
	case "scan":
		return runScan(ctx, args[1:], stdout, stderr)
	case "help", "-h", "--help":
		fmt.Fprint(stdout, usage)
		return 0
	}
	fmt.Fprintf(stderr, "groma: unknown command %q\n\n%s", args[0], usage)
	return 2
}

func runScan(_ context.Context, args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("scan", flag.ContinueOnError)
	flags.SetOutput(stderr)
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if flags.NArg() != 1 {
		fmt.Fprintf(stderr, "groma: scan takes one path\n\n%s", usage)
		return 2
	}

	facts, err := scan.Collect(flags.Arg(0))
	if err != nil {
		fmt.Fprintf(stderr, "groma: %v\n", err)
		return 2
	}
	findings := scan.Check(facts, scan.Rules())
	report.Text(stdout, findings)
	if len(findings) > 0 {
		return 1
	}
	return 0
}
