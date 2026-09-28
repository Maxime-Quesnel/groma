package rules

import (
	"fmt"
	"path"
	"regexp"
	"strings"

	"github.com/Maxime-Quesnel/groma/internal/claudecode"
	"github.com/Maxime-Quesnel/groma/internal/component"
	"github.com/Maxime-Quesnel/groma/internal/fix"
	"github.com/Maxime-Quesnel/groma/internal/rule"
)

const tocAfter = 100

var referenceNoToc = Rule{
	Meta: rule.Meta{
		ID:    "reference-no-toc",
		Level: rule.Warning,
		Title: "Long reference file without a table of contents",
		Description: "Claude often previews a reference file with a partial read, such as its first hundred lines. Anthropic's guidance is to open any reference over 100 lines " +
			"with a table of contents, so Claude sees everything the file covers and can jump to the right section.",
		Remediation: "Add a short \"Contents\" list at the top naming the file's sections.",
		FalsePositives: []string{
			"A long file of data, such as a table, that Claude reads whole or searches with grep.",
		},
		References: []string{claudecode.BestPractices},
	},
	Kinds: skills,
	Check: func(c *component.Component, t *component.Tree) []string {
		var evidence []string
		for _, f := range markdownFiles(c, t) {
			if f.Path == c.Path || path.Ext(f.Path) != ".md" {
				continue
			}
			lines := strings.Split(string(f.Content), "\n")
			if len(lines) > tocAfter && !hasToc(lines) {
				evidence = append(evidence, fmt.Sprintf("%s runs %d lines", f.Path, len(lines)))
			}
		}
		return evidence
	},
	Fix: fixReferenceToc,
}

var (
	tocHeading = regexp.MustCompile(`(?i)^#{1,6}\s*(?:contents|table of contents|toc|index|sections)\b`)
	anchorItem = regexp.MustCompile(`^\s*(?:[-*]|\d+\.)\s+\[[^\]]+\]\(#`)
	listItem   = regexp.MustCompile(`^\s*(?:[-*]|\d+\.)\s+\S`)
)

// hasToc reports whether a file opens with a table of contents: a Contents
// heading, or a list of links to its own sections, within its first lines.
func hasToc(lines []string) bool {
	anchors := 0
	for _, line := range lines[:min(len(lines), 40)] {
		if tocHeading.MatchString(line) {
			return true
		}
		if anchorItem.MatchString(line) {
			anchors++
		}
	}
	return anchors >= 3
}

// fixReferenceToc opens each long reference file with a Contents list of its
// sections, after its title.
func fixReferenceToc(c *component.Component, t *component.Tree, unsafe bool) []fix.Edit {
	var edits []fix.Edit
	for _, f := range markdownFiles(c, t) {
		lines := strings.Split(string(f.Content), "\n")
		if f.Path == c.Path || path.Ext(f.Path) != ".md" || len(lines) <= tocAfter || hasToc(lines) {
			continue
		}
		var sections []string
		fenced := false
		for _, line := range lines {
			if strings.HasPrefix(strings.TrimSpace(line), "```") {
				fenced = !fenced
			}
			if title, ok := strings.CutPrefix(line, "## "); ok && !fenced {
				sections = append(sections, "- "+strings.TrimSpace(title))
			}
		}
		if len(sections) < 2 {
			continue
		}
		at := 0
		if strings.HasPrefix(lines[0], "# ") {
			at = len(lines[0]) + 1
			if len(lines) > 1 && strings.TrimSpace(lines[1]) == "" {
				at += len(lines[1]) + 1
			}
		}
		edits = append(edits, fix.Edit{Path: f.Path, Start: at, End: at, New: "## Contents\n\n" + strings.Join(sections, "\n") + "\n\n"})
	}
	return edits
}
