package claudecode

import (
	"encoding/json"
	"maps"
	"os"
	"path/filepath"
	"slices"
)

type Location struct {
	Path string
	// For names the project a plugin is installed for, when it isn't
	// installed for the user as a whole.
	For string
}

// Installed lists what Claude Code loads for a user: their settings,
// instructions, skills, agents, commands and hook scripts, every installed
// plugin version, including those installed for another project, and the
// .claude directory, CLAUDE.md and .mcp.json of project. It leaves out what
// doesn't exist, and cached versions or marketplace clones that aren't
// installed.
func Installed(configDir, project string) []Location {
	var locations []Location
	add := func(l Location) {
		seen := slices.ContainsFunc(locations, func(m Location) bool { return m.Path == l.Path })
		if _, err := os.Stat(l.Path); err == nil && !seen {
			locations = append(locations, l)
		}
	}
	for _, name := range []string{"settings.json", "CLAUDE.md", "skills", "agents", "commands", "hooks"} {
		add(Location{Path: filepath.Join(configDir, name)})
	}
	for _, l := range installedPlugins(filepath.Join(configDir, "plugins", "installed_plugins.json")) {
		add(l)
	}
	for _, name := range []string{".claude", "CLAUDE.md", ".mcp.json"} {
		add(Location{Path: filepath.Join(project, name)})
	}
	return locations
}

func installedPlugins(registry string) []Location {
	content, err := os.ReadFile(registry)
	if err != nil {
		return nil
	}
	var installed struct {
		Plugins map[string][]struct {
			InstallPath string `json:"installPath"`
			ProjectPath string `json:"projectPath"`
		} `json:"plugins"`
	}
	if json.Unmarshal(content, &installed) != nil {
		return nil
	}
	var locations []Location
	for _, name := range slices.Sorted(maps.Keys(installed.Plugins)) {
		for _, install := range installed.Plugins[name] {
			if install.InstallPath != "" {
				locations = append(locations, Location{Path: install.InstallPath, For: install.ProjectPath})
			}
		}
	}
	return locations
}
