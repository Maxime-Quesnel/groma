package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/Maxime-Quesnel/groma/internal/component"
	"github.com/Maxime-Quesnel/groma/internal/config"
	"github.com/Maxime-Quesnel/groma/internal/report"
	"github.com/Maxime-Quesnel/groma/internal/rule"
	"github.com/Maxime-Quesnel/groma/internal/rules"
	"github.com/Maxime-Quesnel/groma/internal/style"
)

func printUsage(w io.Writer) {
	st := style.For(w)
	command := func(name, what string) {
		fmt.Fprintf(w, "  %s  %s\n", style.Pad(name, 18, st.Cyan), what)
	}
	fmt.Fprintf(w, "%s  %s\n\n", st.Bold("groma"), "checks how Claude Code skills, agents, commands and hooks are written")
	fmt.Fprintln(w, st.Bold("Usage"))
	command("groma check <path>", "check a skill, agent, command or hooks file, or every one")
	command("", "in a directory, such as a plugin or a marketplace")
	command("groma fix <path>", "correct what can be corrected without changing what runs;")
	command("", "shows the diff and asks before writing")
	command("  --unsafe", "also fixes what changes what runs, when, or with which tools")
	fmt.Fprintf(w, "\n%s  turn rules off in %s or with a %s comment\n", st.Bold("Config"), st.Cyan(".groma.yml"), st.Cyan("# groma:disable <rule>"))
	fmt.Fprintf(w, "%s  0 no red flags %s 1 red flags %s 2 error\n", st.Bold("Exit status"), st.Dim("·"), st.Dim("·"))
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		printUsage(stderr)
		return 2
	}
	switch args[0] {
	case "check":
		return runCheck(args[1:], stdout, stderr)
	case "fix":
		return runFix(args[1:], stdin, stdout, stderr)
	case "help", "-h", "--help":
		printUsage(stdout)
		return 0
	}
	fmt.Fprintf(stderr, "groma: unknown command %q\n\n", args[0])
	printUsage(stderr)
	return 2
}

func runCheck(args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 || strings.HasPrefix(args[0], "-") {
		fmt.Fprintf(stderr, "groma: check takes one path\n\n")
		printUsage(stderr)
		return 2
	}
	tree, err := component.Load(args[0])
	if err != nil {
		fmt.Fprintf(stderr, "groma: %v\n", err)
		return 2
	}
	cfg, err := config.Find(args[0], rules.IDs())
	if err != nil {
		fmt.Fprintf(stderr, "groma: %v\n", err)
		return 2
	}
	findings, silenced := rules.Check(tree, cfg)
	report.Text(stdout, shortenHome(args[0]), tree.Targets, findings, silenced)
	for _, f := range findings {
		if f.Rule.Level == rule.RedFlag {
			return 1
		}
	}
	return 0
}

// shortenHome writes the home directory as ~ in the path the report shows.
func shortenHome(p string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return p
	}
	abs, err := filepath.Abs(p)
	if err != nil {
		return p
	}
	if rest, ok := strings.CutPrefix(abs, home+string(filepath.Separator)); ok && filepath.IsAbs(p) {
		return "~/" + filepath.ToSlash(rest)
	}
	return p
}
