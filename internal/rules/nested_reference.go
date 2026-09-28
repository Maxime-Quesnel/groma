package rules

import (
	"fmt"
	"path"
	"slices"

	"github.com/Maxime-Quesnel/groma/internal/claudecode"
	"github.com/Maxime-Quesnel/groma/internal/component"
	"github.com/Maxime-Quesnel/groma/internal/rule"
)

var nestedReference = Rule{
	Meta: rule.Meta{
		ID:    "nested-reference",
		Level: rule.Warning,
		Title: "Reference reachable only through another reference",
		Description: "When a file SKILL.md links to links in turn to another, Claude may preview the second with a partial read, such as head -100, " +
			"and miss what it holds. Anthropic's guidance keeps references one level deep: every reference file linked straight from SKILL.md.",
		Remediation: "Link the file from SKILL.md, next to the others.",
		FalsePositives: []string{
			"A file only meant for humans, such as a changelog, that Claude doesn't need to read.",
		},
		References: []string{claudecode.BestPractices},
	},
	Kinds: skills,
	Check: func(c *component.Component, t *component.Tree) []string {
		own, _ := t.File(c.Path)
		direct := linkedMarkdown(c, t, own)
		var evidence []string
		var reported []string
		for _, ref := range direct {
			f, _ := t.File(ref)
			for _, deeper := range linkedMarkdown(c, t, f) {
				if deeper == c.Path || slices.Contains(direct, deeper) || slices.Contains(reported, deeper) {
					continue
				}
				reported = append(reported, deeper)
				evidence = append(evidence, fmt.Sprintf("%s is only linked from %s", deeper, ref))
			}
		}
		return evidence
	},
}

// linkedMarkdown returns the Markdown files of the skill that f links to.
func linkedMarkdown(c *component.Component, t *component.Tree, f component.File) []string {
	var files []string
	for _, l := range links(c, t, f) {
		if _, ok := t.File(l.file); ok && path.Ext(l.file) == ".md" && !slices.Contains(files, l.file) &&
			(c.Dir == "." || len(l.file) > len(c.Dir) && l.file[:len(c.Dir)+1] == c.Dir+"/") {
			files = append(files, l.file)
		}
	}
	return files
}
