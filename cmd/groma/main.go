package main

import (
	"cmp"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/Maxime-Quesnel/groma/internal/agent/claudecode"
	"github.com/Maxime-Quesnel/groma/internal/plugin"
	"github.com/Maxime-Quesnel/groma/internal/report"
	"github.com/Maxime-Quesnel/groma/internal/rule"
	"github.com/Maxime-Quesnel/groma/internal/rules/scan"
)

const usage = `groma draws the perimeter around your AI agents.

Usage:
  groma scan          check what Claude Code loads: your settings, skills, agents
                      and hooks, every installed plugin, and this project's .claude
  groma scan <path>   check a plugin, a marketplace or a skill directory
  groma bench <plugin>  measure how precisely Claude routes work to each agent
                      of a plugin, from its eval suite (runs Claude on your
                      Claude Code plan; see groma bench -h)

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
	case "bench":
		return runBench(args[1:], stdout, stderr)
	case "help", "-h", "--help":
		fmt.Fprint(stdout, usage)
		return 0
	}
	fmt.Fprintf(stderr, "groma: unknown command %q\n\n%s", args[0], usage)
	return 2
}

func runScan(args []string, stdout, stderr io.Writer) int {
	var p plugin.Plugin
	var err error
	switch {
	case len(args) == 0:
		p, err = readInstalled(stdout)
	case len(args) == 1 && !strings.HasPrefix(args[0], "-"):
		p, err = plugin.Read(args[0])
	default:
		fmt.Fprintf(stderr, "groma: scan takes at most one path\n\n%s", usage)
		return 2
	}
	if err != nil {
		fmt.Fprintf(stderr, "groma: %v\n", err)
		return 2
	}
	p.Hooks = claudecode.Hooks(p)
	p.Grants = claudecode.Grants(p)
	findings := shortenHome(scan.Check(p))
	report.Text(stdout, findings)
	if len(findings) > 0 {
		return 1
	}
	return 0
}

func readInstalled(stdout io.Writer) (plugin.Plugin, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return plugin.Plugin{}, err
	}
	project, err := os.Getwd()
	if err != nil {
		return plugin.Plugin{}, err
	}
	configDir := cmp.Or(os.Getenv("CLAUDE_CONFIG_DIR"), filepath.Join(home, ".claude"))

	var all plugin.Plugin
	fmt.Fprintln(stdout, "Scanned:")
	for _, l := range claudecode.Installed(configDir, project) {
		p, err := plugin.ReadAs(l.Path, filepath.ToSlash(l.Path))
		if err != nil {
			return all, err
		}
		all.Files = append(all.Files, p.Files...)
		if l.For != "" {
			fmt.Fprintf(stdout, "  %s (for %s)\n", tilde(l.Path, home), tilde(l.For, home))
		} else {
			fmt.Fprintf(stdout, "  %s\n", tilde(l.Path, home))
		}
	}
	fmt.Fprintln(stdout)
	return all, nil
}

// shortenHome writes the home directory as ~ in what the report shows.
func shortenHome(findings []rule.Finding) []rule.Finding {
	home, err := os.UserHomeDir()
	if err != nil {
		return findings
	}
	for i, f := range findings {
		findings[i].Subject = tilde(f.Subject, home)
		evidence := make([]string, len(f.Evidence))
		for j, e := range f.Evidence {
			evidence[j] = strings.ReplaceAll(e, home+"/", "~/")
		}
		findings[i].Evidence = evidence
	}
	return findings
}

func tilde(path, home string) string {
	if rest, ok := strings.CutPrefix(path, home+string(filepath.Separator)); ok {
		return "~/" + filepath.ToSlash(rest)
	}
	return path
}
