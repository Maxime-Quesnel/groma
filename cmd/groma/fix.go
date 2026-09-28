package main

import (
	"bufio"
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Maxime-Quesnel/groma/internal/component"
	"github.com/Maxime-Quesnel/groma/internal/config"
	"github.com/Maxime-Quesnel/groma/internal/fix"
	"github.com/Maxime-Quesnel/groma/internal/report"
	"github.com/Maxime-Quesnel/groma/internal/rules"
	"github.com/Maxime-Quesnel/groma/internal/style"
)

func runFix(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	unsafe := false
	var paths []string
	for _, a := range args {
		switch {
		case a == "--unsafe":
			unsafe = true
		case strings.HasPrefix(a, "-"):
			fmt.Fprintf(stderr, "groma: fix has no option %s\n\n", a)
			printUsage(stderr)
			return 2
		default:
			paths = append(paths, a)
		}
	}
	if len(paths) != 1 {
		fmt.Fprintf(stderr, "groma: fix takes one path\n\n")
		printUsage(stderr)
		return 2
	}
	tree, err := component.Load(paths[0])
	if err != nil {
		fmt.Fprintf(stderr, "groma: %v\n", err)
		return 2
	}
	cfg, err := config.Find(paths[0], rules.IDs())
	if err != nil {
		fmt.Fprintf(stderr, "groma: %v\n", err)
		return 2
	}
	read := func(p string) ([]byte, bool) {
		f, ok := tree.File(p)
		return f.Content, ok
	}
	files, dropped := fix.Plan(rules.Fixes(tree, cfg, unsafe), read)
	st := style.For(stdout)
	more := 0
	if !unsafe {
		all, _ := fix.Plan(rules.Fixes(tree, cfg, true), read)
		more = fixCount(all) - fixCount(files)
	}
	if len(files) == 0 {
		fmt.Fprintln(stdout, st.Green("✔ Nothing groma can fix without changing what runs."))
		if more > 0 {
			fmt.Fprintf(stdout, "%s\n", st.Dim(fmt.Sprintf("%s what runs, when, or with which tools: review with groma fix --unsafe %s", changes(more), paths[0])))
		}
		return 0
	}

	for _, f := range files {
		report.Diff(stdout, fix.Diff(f.Path, f.Old, f.New))
		if f.MakeExecutable {
			fmt.Fprintf(stdout, "%s %s\n", st.Bold("chmod +x"), f.Path)
		}
		fmt.Fprintln(stdout)
	}
	fmt.Fprintf(stdout, "%s in %s: %s\n", st.Bold(count(fixCount(files), "fix")), count(len(files), "file"), byRule(files))
	if dropped > 0 {
		fmt.Fprintf(stdout, "%s\n", st.Dim(fmt.Sprintf("%s overlap these and wait for the next run", count(dropped, "more fix"))))
	}
	if more > 0 {
		fmt.Fprintf(stdout, "%s\n", st.Dim(fmt.Sprintf("%s what runs, when, or with which tools: review with groma fix --unsafe", changes(more))))
	}

	if f, ok := stdin.(*os.File); ok && !terminal(f) {
		fmt.Fprintln(stdout, "Nothing written: groma fix asks before changing a file, so run it in a terminal.")
		return 0
	}
	fmt.Fprint(stdout, "Apply these changes? [y/N] ")
	answer, _ := bufio.NewReader(stdin).ReadString('\n')
	if a := strings.ToLower(strings.TrimSpace(answer)); a != "y" && a != "yes" {
		fmt.Fprintln(stdout, "Nothing written.")
		return 0
	}
	for _, f := range files {
		if err := write(filepath.Join(tree.Root, filepath.FromSlash(f.Path)), f); err != nil {
			fmt.Fprintf(stderr, "groma: %v\n", err)
			return 2
		}
	}
	fmt.Fprintf(stdout, "%s\n", st.Green(fmt.Sprintf("✔ Fixed %s in %s. Run groma check %s to see what's left.", count(fixCount(files), "problem"), count(len(files), "file"), paths[0])))
	return 0
}

func write(path string, f fix.File) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	mode := info.Mode().Perm()
	if f.MakeExecutable {
		// Execute permission follows read permission: whoever may read the
		// script may run it.
		mode |= (mode & 0o444) >> 2
	}
	if string(f.New) != string(f.Old) {
		if err := os.WriteFile(path, f.New, mode); err != nil {
			return err
		}
	}
	return os.Chmod(path, mode)
}

func fixCount(files []fix.File) int {
	n := 0
	for _, f := range files {
		for _, k := range f.Rules {
			n += k
		}
	}
	return n
}

// byRule lists how many fixes each rule makes: unquoted-path 2, field-typo 1.
func byRule(files []fix.File) string {
	counts := map[string]int{}
	for _, f := range files {
		for rule, n := range f.Rules {
			counts[rule] += n
		}
	}
	var parts []string
	for _, rule := range slices.Sorted(maps.Keys(counts)) {
		parts = append(parts, fmt.Sprintf("%s %d", rule, counts[rule]))
	}
	return strings.Join(parts, ", ")
}

// changes says how many more fixes change something: "1 more fix changes".
func changes(n int) string {
	if n == 1 {
		return "1 more fix changes"
	}
	return count(n, "more fix") + " change"
}

func terminal(f *os.File) bool {
	info, err := f.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

func count(n int, noun string) string {
	if n == 1 {
		return "1 " + noun
	}
	if strings.HasSuffix(noun, "x") {
		return fmt.Sprintf("%d %ses", n, noun)
	}
	return fmt.Sprintf("%d %ss", n, noun)
}
