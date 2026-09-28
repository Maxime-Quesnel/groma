package rules

import (
	"path"
	"regexp"
	"strings"

	"github.com/Maxime-Quesnel/groma/internal/claudecode"
	"github.com/Maxime-Quesnel/groma/internal/component"
	"github.com/Maxime-Quesnel/groma/internal/rule"
)

var brokenLink = Rule{
	Meta: rule.Meta{
		ID:    "broken-link",
		Level: rule.Warning,
		Title: "Link to a file that isn't there",
		Description: "A Markdown link, or a path through ${CLAUDE_SKILL_DIR} or ${CLAUDE_PLUGIN_ROOT}, points at a file the component doesn't ship. " +
			"Claude follows it when the task calls for it, finds nothing, and carries on without the instructions or reference it was meant to read.",
		Remediation: "Fix the path, or ship the file it names.",
		FalsePositives: []string{
			"A file generated at install or first run.",
			"A link written as an example inside a code block. Markdown links inside fenced code blocks are skipped; ${…} paths are not, since Claude Code substitutes them there too.",
		},
		References: []string{claudecode.BestPractices},
	},
	Kinds: markdown,
	Check: func(c *component.Component, t *component.Tree) []string {
		var evidence []string
		for _, f := range markdownFiles(c, t) {
			for _, l := range links(c, t, f) {
				if !l.found && !l.outside {
					evidence = append(evidence, in(c, f, at(l.line, "%s isn't there", l.target)))
				}
			}
		}
		return evidence
	},
}

type link struct {
	line   int
	target string
	// file is the linked path, relative to the tree's root; outside reports
	// that it lies above the root, where groma can't see.
	file           string
	found, outside bool
}

var (
	markdownLink = regexp.MustCompile(`\[[^\]\n]*\]\(\s*<?([^)\s>]+)>?(?:\s+"[^"]*")?\s*\)`)
	variablePath = regexp.MustCompile(`\$\{CLAUDE_(SKILL_DIR|PLUGIN_ROOT)\}(/[^\s"'` + "`" + `)\]<>]+)`)
)

// markdownFiles returns the component's own file and, for a skill, the other
// Markdown files it ships.
func markdownFiles(c *component.Component, t *component.Tree) []component.File {
	var files []component.File
	for _, f := range t.Owned(c) {
		// Files under assets/ are templates the skill copies elsewhere, so
		// their links point into the destination.
		if f.Path == c.Path || path.Ext(f.Path) == ".md" && !strings.Contains("/"+f.Path, "/assets/") {
			files = append(files, f)
		}
	}
	return files
}

// links returns the local files f links to.
func links(c *component.Component, t *component.Tree, f component.File) []link {
	var found []link
	roots := c.Roots()
	fenced := false
	for i, line := range strings.Split(string(f.Content), "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			fenced = !fenced
		}
		if !fenced {
			for _, m := range markdownLink.FindAllStringSubmatch(line, -1) {
				target := m[1]
				if strings.Contains(target, "://") || strings.HasPrefix(target, "mailto:") || strings.HasPrefix(target, "#") ||
					strings.HasPrefix(target, "/") || strings.Contains(target, "$") {
					continue
				}
				target, _, _ = strings.Cut(target, "#")
				target, _, _ = strings.Cut(target, "?")
				file := path.Join(path.Dir(f.Path), target)
				found = append(found, newLink(t, i+1, target, file))
			}
		}
		for _, m := range variablePath.FindAllStringSubmatch(line, -1) {
			root, ok := roots[m[1]]
			if !ok {
				continue
			}
			rest := strings.TrimRight(m[2], ".,:;")
			found = append(found, newLink(t, i+1, "${CLAUDE_"+m[1]+"}"+rest, path.Join(root, rest)))
		}
	}
	return found
}

func newLink(t *component.Tree, line int, target, file string) link {
	outside := file == ".." || strings.HasPrefix(file, "../")
	return link{line: line, target: target, file: file, outside: outside, found: !outside && t.Exists(file)}
}

// within reports whether p lies inside dir, both relative to the tree's root.
func within(p, dir string) bool {
	if dir == "." {
		return p != ".." && !strings.HasPrefix(p, "../")
	}
	return p == dir || strings.HasPrefix(p, dir+"/")
}
