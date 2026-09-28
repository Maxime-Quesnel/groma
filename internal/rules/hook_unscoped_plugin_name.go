package rules

import (
	"encoding/json"
	"fmt"
	"path"
	"slices"
	"strings"

	"github.com/Maxime-Quesnel/groma/internal/claudecode"
	"github.com/Maxime-Quesnel/groma/internal/component"
	"github.com/Maxime-Quesnel/groma/internal/rule"
)

var hookUnscopedPluginName = Rule{
	Meta: rule.Meta{
		ID:    "hook-unscoped-plugin-name",
		Level: rule.RedFlag,
		Title: "Plugin hook names the plugin's own agent or MCP server without its prefix",
		Description: "Claude Code scopes what a plugin ships: its agents are my-plugin:reviewer, its MCP tools mcp__plugin_my-plugin_<server>__<tool>, " +
			"and an mcp_tool hook reaches its server as plugin:my-plugin:<server>. A plugin hook that uses the bare name, as the files write it, never fires or finds no server.",
		Remediation: "Use the scoped name: my-plugin:reviewer in a SubagentStart or SubagentStop matcher, mcp__plugin_my-plugin_<server>__.* in a tool matcher, " +
			"and plugin:my-plugin:<server> as an mcp_tool hook's server.",
		FalsePositives: []string{
			"A user or project agent, or an MCP server configured outside the plugin, that happens to share the name.",
		},
		References: []string{claudecode.HooksDocs},
	},
	Kinds: hooks,
	Check: func(c *component.Component, t *component.Tree) []string {
		if !c.InPlugin || c.Plugin == "" || c.PluginName == "" {
			return nil
		}
		var agentIDs []string
		for _, o := range t.Components {
			if o.Kind == component.Agent && o.InPlugin && o.Plugin == c.Plugin {
				agentIDs = append(agentIDs, o.ID())
			}
		}
		servers := mcpServers(t, c.Plugin)
		var evidence []string
		for _, g := range c.Hooks.Groups {
			if g.Event == "SubagentStart" || g.Event == "SubagentStop" {
				alternatives, _ := claudecode.MatcherAlternatives(g.Matcher)
				for _, a := range alternatives {
					if slices.Contains(agentIDs, a) {
						evidence = append(evidence, fmt.Sprintf("%s: the agent type is %s:%s", g.Where(), c.PluginName, a))
					}
				}
			}
			for _, s := range servers {
				if strings.Contains(g.Matcher, "mcp__"+s+"__") {
					evidence = append(evidence, fmt.Sprintf("%s: the plugin's %s tools are mcp__plugin_%s_%s__…", g.Where(), s, c.PluginName, s))
				}
			}
		}
		for _, h := range c.Hooks.Handlers {
			if server, ok := h.String("server"); ok && h.Type == "mcp_tool" && slices.Contains(servers, server) {
				evidence = append(evidence, fmt.Sprintf("%s: the server is plugin:%s:%s", h.Where(), c.PluginName, server))
			}
		}
		return evidence
	},
}

// mcpServers returns the names of the MCP servers a plugin's .mcp.json
// declares.
func mcpServers(t *component.Tree, root string) []string {
	f, ok := t.File(path.Join(root, ".mcp.json"))
	if !ok {
		return nil
	}
	var config map[string]json.RawMessage
	if json.Unmarshal(f.Content, &config) != nil {
		return nil
	}
	if wrapped, ok := config["mcpServers"]; ok {
		config = nil
		json.Unmarshal(wrapped, &config)
	}
	var names []string
	for name := range config {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}
