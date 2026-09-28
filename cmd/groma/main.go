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
	"github.com/Maxime-Quesnel/groma/internal/style"
)

func printUsage(w io.Writer) {
	st := style.For(w)
	command := func(name, what string) {
		fmt.Fprintf(w, "  %s  %s\n", style.Pad(name, 18, st.Cyan), what)
	}
	fmt.Fprintf(w, "%s  %s\n\n", st.Bold("groma"), "draws the perimeter around your AI agents")
	fmt.Fprintln(w, st.Bold("Usage"))
	command("groma scan", "check everything Claude Code loads: your settings, skills, agents")
	command("", "and hooks, every installed plugin, and this project's .claude")
	command("groma scan <path>", "check one plugin, marketplace or skill directory")
	command("groma bench", "measure how precisely Claude routes work to each agent of a plugin;")
	command("", "asks what to measure (flags: groma bench -h)")
	fmt.Fprintf(w, "\n%s  0 nothing found %s 1 findings %s 2 error\n", st.Bold("Exit status"), st.Dim("·"), st.Dim("·"))
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		printUsage(stderr)
		return 2
	}
	switch args[0] {
	case "scan":
		return runScan(args[1:], stdout, stderr)
	case "bench":
		return runBench(args[1:], stdout, stderr)
	case "help", "-h", "--help":
		printUsage(stdout)
		return 0
	}
	fmt.Fprintf(stderr, "groma: unknown command %q\n\n", args[0])
	printUsage(stderr)
	return 2
}

func runScan(args []string, stdout, stderr io.Writer) int {
	var p plugin.Plugin
	var err error
	switch {
	case len(args) == 0:
		p, err = readInstalled(stdout)
	case len(args) == 1 && !strings.HasPrefix(args[0], "-"):
		if p, err = plugin.Read(args[0]); err == nil {
			fmt.Fprintf(stdout, "%s\n\n", style.For(stdout).Dim(fmt.Sprintf("Scanning %s · %s", args[0], count(len(p.Files), "file"))))
		}
	default:
		fmt.Fprintf(stderr, "groma: scan takes at most one path\n\n")
		printUsage(stderr)
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
	var user, here []string
	plugins := 0
	for _, l := range claudecode.Installed(configDir, project) {
		p, err := plugin.ReadAs(l.Path, filepath.ToSlash(l.Path))
		if err != nil {
			return all, err
		}
		all.Files = append(all.Files, p.Files...)
		switch {
		case strings.HasPrefix(l.Path, filepath.Join(configDir, "plugins")+string(filepath.Separator)):
			plugins++
		case strings.HasPrefix(l.Path, configDir+string(filepath.Separator)):
			user = append(user, filepath.Base(l.Path))
		default:
			here = append(here, filepath.Base(l.Path))
		}
	}
	var parts []string
	if len(user) > 0 {
		parts = append(parts, fmt.Sprintf("%s (%s)", tilde(configDir, home), strings.Join(user, ", ")))
	}
	if plugins > 0 {
		parts = append(parts, count(plugins, "installed plugin"))
	}
	if len(here) > 0 {
		parts = append(parts, fmt.Sprintf("this project (%s)", strings.Join(here, ", ")))
	}
	fmt.Fprintf(stdout, "%s\n\n", style.For(stdout).Dim(fmt.Sprintf("Scanning what Claude Code loads: %s · %s", strings.Join(parts, ", "), count(len(all.Files), "file"))))
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
