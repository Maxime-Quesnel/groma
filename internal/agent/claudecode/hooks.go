// Package claudecode reads what Claude Code plugins and projects declare.
package claudecode

import (
	"encoding/json"
	"maps"
	"path"
	"regexp"
	"slices"
	"strings"

	"github.com/Maxime-Quesnel/groma/internal/plugin"
)

type handler struct {
	Type    string   `json:"type"`
	Command string   `json:"command"`
	Args    []string `json:"args"`
}

type events map[string][]struct {
	Hooks []handler `json:"hooks"`
}

// Hooks lists the command hooks declared in hooks/hooks.json, in the hooks
// field of .claude-plugin/plugin.json and in .claude/settings*.json.
func Hooks(p plugin.Plugin) []plugin.Hook {
	var hooks []plugin.Hook
	parsed := map[string]bool{}
	add := func(source, root string, e events) {
		for _, event := range slices.Sorted(maps.Keys(e)) {
			for _, g := range e[event] {
				for _, h := range g.Hooks {
					if h.Type != "" && h.Type != "command" {
						continue
					}
					command := strings.Join(append([]string{h.Command}, h.Args...), " ")
					hooks = append(hooks, plugin.Hook{
						Event:   event,
						Command: command,
						Source:  source,
						Scripts: scripts(p, root, command),
					})
				}
			}
		}
	}
	addFile := func(file, root string) {
		if parsed[file] {
			return
		}
		parsed[file] = true
		if f, ok := p.File(file); ok {
			add(file, root, hooksFile(f.Content))
		}
	}

	for _, f := range p.Files {
		switch dir, base := path.Split(f.Path); {
		case base == "hooks.json" && strings.HasSuffix(dir, "hooks/"):
			addFile(f.Path, path.Dir(strings.TrimSuffix(dir, "/")))
		case base == "plugin.json" && strings.HasSuffix(dir, ".claude-plugin/"):
			root := path.Dir(strings.TrimSuffix(dir, "/"))
			files, inline := manifestHooks(f.Content)
			for _, file := range files {
				addFile(path.Join(root, file), root)
			}
			for _, e := range inline {
				add(f.Path, root, e)
			}
		case (base == "settings.json" || base == "settings.local.json") && strings.HasSuffix(dir, ".claude/"):
			addFile(f.Path, path.Dir(strings.TrimSuffix(dir, "/")))
		}
	}
	return hooks
}

// A hooks file holds {"hooks": {...}}; a manifest's inline form is the event
// map itself. Accept both wherever either may appear.
func hooksFile(content []byte) events {
	var wrapped struct {
		Hooks events `json:"hooks"`
	}
	if json.Unmarshal(content, &wrapped) == nil && wrapped.Hooks != nil {
		return wrapped.Hooks
	}
	var bare events
	json.Unmarshal(content, &bare)
	return bare
}

// The manifest's hooks field is a path, an inline object, or an array of both.
func manifestHooks(content []byte) (files []string, inline []events) {
	var manifest struct {
		Hooks json.RawMessage `json:"hooks"`
	}
	if json.Unmarshal(content, &manifest) != nil || manifest.Hooks == nil {
		return nil, nil
	}
	entries := []json.RawMessage{manifest.Hooks}
	var list []json.RawMessage
	if json.Unmarshal(manifest.Hooks, &list) == nil {
		entries = list
	}
	for _, entry := range entries {
		var file string
		if json.Unmarshal(entry, &file) == nil {
			files = append(files, file)
		} else {
			inline = append(inline, hooksFile(entry))
		}
	}
	return files, inline
}

// Plugin hooks reach their scripts through ${CLAUDE_PLUGIN_ROOT}, project
// hooks through ${CLAUDE_PROJECT_DIR}; both are often quoted apart from the
// rest of the path, as in "${CLAUDE_PLUGIN_ROOT}"/scripts/x.sh.
var scriptPath = regexp.MustCompile(`\$\{?CLAUDE_(?:PLUGIN_ROOT|PROJECT_DIR)\}?(/[^\s;&|)]+)`)

func scripts(p plugin.Plugin, root, command string) []string {
	var found []string
	unquoted := strings.NewReplacer(`"`, "", `'`, "").Replace(command)
	for _, m := range scriptPath.FindAllStringSubmatch(unquoted, -1) {
		file := path.Join(root, m[1])
		if _, ok := p.File(file); ok {
			found = append(found, file)
		}
	}
	return found
}
