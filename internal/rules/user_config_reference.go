package rules

import (
	"fmt"
	"regexp"

	"github.com/Maxime-Quesnel/groma/internal/claudecode"
	"github.com/Maxime-Quesnel/groma/internal/component"
	"github.com/Maxime-Quesnel/groma/internal/rule"
)

var userConfigReference = Rule{
	Meta: rule.Meta{
		ID:    "user-config-reference",
		Level: rule.RedFlag,
		Title: "Plugin option that isn't there",
		Description: "${user_config.KEY} is replaced with the value of an option the manifest's userConfig declares. A key the manifest doesn't declare has no value, " +
			"and in skill and agent content a sensitive option becomes a placeholder, so Claude never sees the value the text relies on.",
		Remediation: "Declare the option under userConfig in plugin.json. Pass sensitive values to scripts through the environment, never through skill or agent text.",
		FalsePositives: []string{
			"An option shown as an example outside a code block. Components outside a plugin, and code blocks, aren't checked.",
		},
		References: []string{claudecode.PluginManifestDocs},
	},
	Kinds: everything,
	Check: func(c *component.Component, t *component.Tree) []string {
		plugin := t.PluginOf(c)
		if plugin == nil {
			return nil
		}
		var evidence []string
		report := func(where, key string, inText bool) {
			sensitive, declared := plugin.Manifest.UserConfig[key]
			switch {
			case !declared:
				evidence = append(evidence, fmt.Sprintf("%s uses user_config.%s, which the manifest doesn't declare", where, key))
			case sensitive && inText:
				evidence = append(evidence, fmt.Sprintf("%s uses user_config.%s, a sensitive option Claude only sees as a placeholder", where, key))
			}
		}
		if c.Kind == component.Hooks {
			for _, cmd := range commands(c) {
				for _, m := range optionRef.FindAllStringSubmatch(cmd.run, -1) {
					report(cmd.where, m[1], false)
				}
			}
			return evidence
		}
		// Code blocks are skipped: MCP bundle manifests, which skills about
		// them show, use the same ${user_config.…} syntax for their own options.
		bodyLines(c, func(n int, line string) {
			for _, m := range optionRef.FindAllStringSubmatch(line, -1) {
				report(fmt.Sprintf("line %d", n), m[1], c.Kind != component.Command)
			}
		})
		return evidence
	},
}

var optionRef = regexp.MustCompile(`\$\{user_config\.([A-Za-z_][A-Za-z0-9_]*)\}`)
