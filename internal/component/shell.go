package component

import (
	"path"
	"regexp"
	"slices"
	"strings"
)

// A Shell is a command a skill or command runs before Claude reads it.
type Shell struct {
	Line    int
	Command string
}

// Skills and commands run !`command` placeholders and ```! blocks without a
// prompt. The inline form only counts at the start of a line or after
// whitespace, and outside code: skills that document the syntax show it in
// fenced blocks and code spans, which Claude Code leaves as text.
var (
	inlineCommand = regexp.MustCompile("(?:^|\\s)!`([^`\\n]+)`")
	codeSpan      = regexp.MustCompile("``[^`\\n]*``")
)

func (c *Component) InlineShell() []Shell {
	if c.Kind != Skill && c.Kind != Command {
		return nil
	}
	var shells []Shell
	var block []string
	blockLine, inShellBlock, inFence := 0, false, false
	for i, line := range strings.Split(c.Header.Body, "\n") {
		n := c.Header.BodyLine + i
		trimmed := strings.TrimSpace(line)
		switch {
		case inShellBlock && strings.HasPrefix(trimmed, "```"):
			shells = append(shells, Shell{blockLine, strings.TrimSpace(strings.Join(block, "\n"))})
			inShellBlock, block = false, nil
		case inShellBlock:
			block = append(block, line)
		case inFence:
			inFence = !strings.HasPrefix(trimmed, "```")
		case strings.HasPrefix(line, "```!"):
			inShellBlock, blockLine = true, n+1
		case strings.HasPrefix(trimmed, "```"):
			inFence = true
		default:
			for _, m := range inlineCommand.FindAllStringSubmatch(codeSpan.ReplaceAllString(line, ""), -1) {
				shells = append(shells, Shell{n, strings.TrimSpace(m[1])})
			}
		}
	}
	return shells
}

// A Script is a file of the plugin or project that a command names.
type Script struct {
	// Ref is the path as the command writes it.
	Ref string
	// Path is the file in the tree, or "" when Ref's root lies outside it.
	Path  string
	Found bool
	// Direct reports that the command runs the file itself, rather than
	// passing it to an interpreter, so the file must be executable.
	Direct bool
}

// Commands reach their scripts through ${CLAUDE_PLUGIN_ROOT},
// ${CLAUDE_PROJECT_DIR} or ${CLAUDE_SKILL_DIR}, often quoted apart from the
// rest of the path, as in "${CLAUDE_PLUGIN_ROOT}"/scripts/x.sh.
var scriptRef = regexp.MustCompile(`\$\{?CLAUDE_(PLUGIN_ROOT|PROJECT_DIR|SKILL_DIR)\}?(/[^\s;&|)<>]+)`)

// Scripts returns the files command names through the directory variables
// of roots.
func (t *Tree) Scripts(command string, roots map[string]string) []Script {
	unquoted := strings.TrimSpace(strings.NewReplacer(`"`, "", `'`, "").Replace(command))
	var scripts []Script
	for _, m := range scriptRef.FindAllStringSubmatchIndex(unquoted, -1) {
		s := Script{Ref: unquoted[m[0]:m[1]], Direct: m[0] == 0}
		if root, ok := roots[unquoted[m[2]:m[3]]]; ok {
			s.Path = path.Join(root, unquoted[m[4]:m[5]])
			s.Found = t.Exists(s.Path)
		}
		if !slices.ContainsFunc(scripts, func(o Script) bool { return o.Ref == s.Ref }) {
			scripts = append(scripts, s)
		}
	}
	return scripts
}

// Exists reports whether p is a file of the tree or a directory holding one.
func (t *Tree) Exists(p string) bool {
	if _, ok := t.File(p); ok {
		return true
	}
	return slices.ContainsFunc(t.Files, func(f File) bool { return strings.HasPrefix(f.Path, p+"/") })
}
