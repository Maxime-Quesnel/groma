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
// field of .claude-plugin/plugin.json and in .claude/settings*.json, and the
// inline shell of skills and commands.
func Hooks(p plugin.Plugin) []plugin.Hook {
	var hooks []plugin.Hook
	parsed := map[string]bool{}
	add := func(source string, roots map[string]string, e events) {
		for _, event := range slices.Sorted(maps.Keys(e)) {
			for _, g := range e[event] {
				for _, h := range g.Hooks {
					if h.Type != "" && h.Type != "command" {
						continue
					}
					command := strings.Join(append([]string{h.Command}, h.Args...), " ")
					hooks = append(hooks, plugin.Hook{
						Trigger: event + " hook",
						Command: command,
						Source:  source,
						Scripts: scripts(p, command, roots),
					})
				}
			}
		}
	}
	addFile := func(file string, roots map[string]string) {
		if parsed[file] {
			return
		}
		parsed[file] = true
		if f, ok := p.File(file); ok {
			add(file, roots, hooksFile(f.Content))
		}
	}

	for _, f := range p.Files {
		switch dir, base := path.Split(f.Path); {
		case base == "hooks.json" && strings.HasSuffix(dir, "hooks/"):
			addFile(f.Path, map[string]string{"PLUGIN_ROOT": parent(dir)})
		case base == "plugin.json" && strings.HasSuffix(dir, ".claude-plugin/"):
			root := parent(dir)
			files, inline := manifestHooks(f.Content)
			for _, file := range files {
				addFile(path.Join(root, file), map[string]string{"PLUGIN_ROOT": root})
			}
			for _, e := range inline {
				add(f.Path, map[string]string{"PLUGIN_ROOT": root}, e)
			}
		case (base == "settings.json" || base == "settings.local.json") && strings.HasSuffix(dir, ".claude/"):
			addFile(f.Path, map[string]string{"PROJECT_DIR": parent(dir)})
		case skillOrCommand(f.Path):
			roots := map[string]string{"SKILL_DIR": path.Dir(f.Path)}
			if root, ok := above(f.Path, "skills", "commands"); ok {
				roots["PLUGIN_ROOT"] = root
			}
			if root, ok := above(f.Path, ".claude"); ok {
				roots["PROJECT_DIR"] = root
			}
			for _, command := range inlineShell(f.Content) {
				hooks = append(hooks, plugin.Hook{Trigger: "inline shell", Command: command, Source: f.Path, Scripts: scripts(p, command, roots)})
			}
		}
	}
	return hooks
}

// Skills and commands run !`command` placeholders and ```! blocks before
// Claude reads them, without a prompt. The inline form only counts at the
// start of a line or after whitespace.
var (
	inlineCommand = regexp.MustCompile("(?m)(?:^|\\s)!`([^`\\n]+)`")
	commandBlock  = regexp.MustCompile("(?ms)^```![ \\t]*\\n(.*?)^```")
)

func inlineShell(content []byte) []string {
	var commands []string
	for _, re := range []*regexp.Regexp{inlineCommand, commandBlock} {
		for _, m := range re.FindAllSubmatch(content, -1) {
			commands = append(commands, strings.TrimSpace(string(m[1])))
		}
	}
	return commands
}

func parent(dir string) string {
	return path.Dir(strings.TrimSuffix(dir, "/"))
}

// above returns the directory holding the nearest enclosing directory named
// one of names, such as the plugin root above skills/.
func above(file string, names ...string) (string, bool) {
	parts := strings.Split(file, "/")
	for i := len(parts) - 2; i >= 0; i-- {
		if slices.Contains(names, parts[i]) {
			return path.Join(append([]string{"."}, parts[:i]...)...), true
		}
	}
	return "", false
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

// Commands reach their scripts through ${CLAUDE_PLUGIN_ROOT},
// ${CLAUDE_PROJECT_DIR} or ${CLAUDE_SKILL_DIR}, often quoted apart from the
// rest of the path, as in "${CLAUDE_PLUGIN_ROOT}"/scripts/x.sh. roots maps
// each variable, without its CLAUDE_ prefix, to a directory of the scan.
var scriptPath = regexp.MustCompile(`\$\{?CLAUDE_(PLUGIN_ROOT|PROJECT_DIR|SKILL_DIR)\}?(/[^\s;&|)]+)`)

func scripts(p plugin.Plugin, command string, roots map[string]string) []string {
	var found []string
	unquoted := strings.NewReplacer(`"`, "", `'`, "").Replace(command)
	for _, m := range scriptPath.FindAllStringSubmatch(unquoted, -1) {
		root, known := roots[m[1]]
		file := path.Join(root, m[2])
		if _, ok := p.File(file); known && ok {
			found = append(found, file)
		}
	}
	return found
}
