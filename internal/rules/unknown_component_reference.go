package rules

import (
	"fmt"
	"path"
	"regexp"
	"slices"
	"strings"

	"github.com/Maxime-Quesnel/groma/internal/claudecode"
	"github.com/Maxime-Quesnel/groma/internal/component"
	"github.com/Maxime-Quesnel/groma/internal/rule"
)

var unknownComponentReference = Rule{
	Meta: rule.Meta{
		ID:    "unknown-component-reference",
		Level: rule.RedFlag,
		Title: "Names a skill, command or agent that doesn't exist",
		Description: "The text sends Claude to plugin:name, /plugin:name or @agent-plugin:name, and that plugin, which groma is checking, has no skill, command, agent or workflow " +
			"of that name. A skill's name field, not its folder, sets its command, and an agent in a subfolder of agents/ carries the subfolder in its name, " +
			"so a rename leaves such references pointing at nothing, and Claude either fails to invoke them or improvises.",
		Remediation: "Use the component's name as Claude Code builds it: plugin:<skill name>, plugin:<command path with : between folders>, or plugin:<agent subfolders>:<agent name>.",
		FalsePositives: []string{
			"A reference to something a plugin gets from elsewhere, such as an output style or an MCP prompt.",
			"References to plugins groma isn't checking are never flagged.",
		},
		References: []string{claudecode.SkillsDocs, claudecode.PluginComponentsDocs},
	},
	Kinds: markdown,
	Check: func(c *component.Component, t *component.Tree) []string {
		known := componentIndex(t)
		if len(known) == 0 {
			return nil
		}
		plugins := make([]string, 0, len(known))
		for name := range known {
			plugins = append(plugins, regexp.QuoteMeta(name))
		}
		slices.Sort(plugins)
		reference := regexp.MustCompile(`(?:^|[^A-Za-z0-9_.@/-])(?:/|@agent-)?(` + strings.Join(plugins, "|") + `):([A-Za-z0-9][A-Za-z0-9_-]*(?::[A-Za-z0-9][A-Za-z0-9_-]*)*)`)
		var evidence []string
		for i, line := range strings.Split(string(c.Content), "\n") {
			for _, m := range reference.FindAllStringSubmatch(line, -1) {
				plugin, id := m[1], m[2]
				// groma:disable is groma's own directive, not a reference
				// to a plugin named groma.
				if slices.Contains(known[plugin], id) || plugin == "groma" && id == "disable" {
					continue
				}
				e := at(i+1, "%s:%s names nothing in %s", plugin, id, plugin)
				if s := suggest(id, known[plugin]); s != "" {
					e += fmt.Sprintf("; did you mean %s:%s?", plugin, s)
				}
				evidence = append(evidence, e)
			}
		}
		return evidence
	},
}

// componentIndex maps each plugin of the tree to the names of its skills,
// commands, agents and workflows, as other components reference them.
func componentIndex(t *component.Tree) map[string][]string {
	known := map[string][]string{}
	roots := map[string]string{}
	for _, c := range t.Components {
		if !c.InPlugin || c.PluginName == "" || c.Kind == component.Hooks {
			continue
		}
		roots[c.PluginName] = c.Plugin
		if c.Kind != component.Plugin {
			known[c.PluginName] = append(known[c.PluginName], c.ID())
		}
	}
	for plugin, root := range roots {
		prefix := path.Join(root, "workflows") + "/"
		for _, f := range t.Files {
			if rest, ok := strings.CutPrefix(f.Path, prefix); ok {
				known[plugin] = append(known[plugin], strings.ReplaceAll(strings.TrimSuffix(rest, path.Ext(rest)), "/", ":"))
			}
		}
	}
	return known
}
