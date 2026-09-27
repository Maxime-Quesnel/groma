package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/Maxime-Quesnel/groma/internal/agent/claudecode"
	"github.com/Maxime-Quesnel/groma/internal/plugin"
	"github.com/Maxime-Quesnel/groma/internal/report"
	"github.com/Maxime-Quesnel/groma/internal/rules/scan"
)

const usage = `groma draws the perimeter around your AI agents.

Usage:
  groma scan <path>   check a plugin, a marketplace or a skill directory

Exit status: 0 when nothing is found, 1 when there are findings, 2 on error.
`

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stderr, usage)
		return 2
	}
	switch args[0] {
	case "scan":
		return runScan(args[1:], stdout, stderr)
	case "help", "-h", "--help":
		fmt.Fprint(stdout, usage)
		return 0
	}
	fmt.Fprintf(stderr, "groma: unknown command %q\n\n%s", args[0], usage)
	return 2
}

func runScan(args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 || strings.HasPrefix(args[0], "-") {
		fmt.Fprintf(stderr, "groma: scan takes one path\n\n%s", usage)
		return 2
	}
	p, err := plugin.Read(args[0])
	if err != nil {
		fmt.Fprintf(stderr, "groma: %v\n", err)
		return 2
	}
	p.Hooks = claudecode.Hooks(p)
	p.Grants = claudecode.Grants(p)
	findings := scan.Check(p)
	report.Text(stdout, findings)
	if len(findings) > 0 {
		return 1
	}
	return 0
}
