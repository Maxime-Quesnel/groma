package rules

import (
	"fmt"
	"path"
	"slices"
	"strings"

	"github.com/Maxime-Quesnel/groma/internal/claudecode"
	"github.com/Maxime-Quesnel/groma/internal/component"
	"github.com/Maxime-Quesnel/groma/internal/rule"
)

var componentNotLoaded = Rule{
	Meta: rule.Meta{
		ID:    "component-not-loaded",
		Level: rule.RedFlag,
		Title: "Component Claude Code doesn't load",
		Description: "The file sits where the plugin's author expects it, but not where Claude Code looks: component folders inside .claude-plugin/ aren't scanned; " +
			"when the manifest lists commands or agents, only the listed ones load and the default folder is ignored; a folder under skills/ without a SKILL.md, spelled exactly so, isn't a skill; " +
			"and a SKILL.md at the plugin's root is ignored once the plugin has a skills/ folder.",
		Remediation: "Move component folders to the plugin's root, list every command and agent in the manifest or drop the key, and name each skill's file SKILL.md.",
		FalsePositives: []string{
			"A file kept on purpose where it doesn't load, such as a draft.",
		},
		References: []string{claudecode.PluginComponentsDocs, claudecode.PluginManifestDocs, claudecode.PluginTroubleshooting},
	},
	Kinds: []component.Kind{component.Plugin},
	Check: func(c *component.Component, t *component.Tree) []string {
		root := c.Dir
		under := func(dir string) string {
			if root == "." {
				return dir
			}
			return path.Join(root, dir)
		}
		var evidence []string
		var skillDirs []string
		for _, f := range t.Files {
			rel, ok := strings.CutPrefix(f.Path, under("")+"/")
			if root == "." {
				rel, ok = f.Path, true
			}
			if !ok {
				continue
			}
			parts := strings.Split(rel, "/")
			switch {
			case len(parts) > 2 && parts[0] == ".claude-plugin" && slices.Contains(kindDirs, parts[1]):
				evidence = append(evidence, fmt.Sprintf("%s sits inside .claude-plugin/, which Claude Code doesn't scan for components", f.Path))
			case len(parts) >= 2 && parts[0] == "agents" && path.Ext(rel) == ".md" && c.Manifest.SetAgents && !slices.Contains(c.Manifest.Agents, rel):
				evidence = append(evidence, fmt.Sprintf("%s isn't listed in the manifest's agents, which replaces the agents/ folder", f.Path))
			case len(parts) >= 2 && parts[0] == "commands" && path.Ext(rel) == ".md" && c.Manifest.SetCommands && !inManifest(c.Manifest.Commands, rel):
				evidence = append(evidence, fmt.Sprintf("%s isn't listed in the manifest's commands, which replaces the commands/ folder", f.Path))
			case len(parts) >= 3 && parts[0] == "skills" && !slices.Contains(skillDirs, parts[1]):
				skillDirs = append(skillDirs, parts[1])
				if _, ok := t.File(under(path.Join("skills", parts[1], "SKILL.md"))); !ok {
					evidence = append(evidence, fmt.Sprintf("%s has no SKILL.md, so it isn't a skill", under(path.Join("skills", parts[1]))))
				}
			}
		}
		if _, rootSkill := t.File(under("SKILL.md")); rootSkill && (len(skillDirs) > 0 || c.Manifest.SetSkills) {
			evidence = append(evidence, fmt.Sprintf("%s is ignored, since the plugin also has skills/ or a skills key", under("SKILL.md")))
		}
		return evidence
	},
}

var kindDirs = []string{"skills", "commands", "agents", "hooks"}

// inManifest reports whether a command file is in the manifest's commands, as
// itself or through a directory.
func inManifest(entries []string, file string) bool {
	return slices.ContainsFunc(entries, func(e string) bool { return e == file || strings.HasPrefix(file, e+"/") })
}
