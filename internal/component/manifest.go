package component

import (
	"encoding/json"
	"path"
	"strings"
)

// Manifest is what .claude-plugin/plugin.json declares that the checks use.
type Manifest struct {
	Name string
	// UserConfig maps each option the plugin declares to whether it's
	// sensitive.
	UserConfig map[string]bool
	// Commands and Agents are the paths the manifest lists, which replace
	// the default commands/ and agents/ scans; Set* tells whether the key
	// is present at all.
	Commands, Agents       []string
	SetCommands, SetAgents bool
	SetSkills              bool
}

func parseManifest(content []byte) Manifest {
	var raw struct {
		Name       string `json:"name"`
		UserConfig map[string]struct {
			Sensitive bool `json:"sensitive"`
		} `json:"userConfig"`
		Commands json.RawMessage `json:"commands"`
		Agents   json.RawMessage `json:"agents"`
		Skills   json.RawMessage `json:"skills"`
	}
	json.Unmarshal(content, &raw)
	m := Manifest{
		Name:        raw.Name,
		UserConfig:  map[string]bool{},
		SetCommands: raw.Commands != nil,
		SetAgents:   raw.Agents != nil,
		SetSkills:   raw.Skills != nil,
		Commands:    paths(raw.Commands),
		Agents:      paths(raw.Agents),
	}
	for key, option := range raw.UserConfig {
		m.UserConfig[key] = option.Sensitive
	}
	return m
}

// paths reads a manifest key that holds a path, a list of paths, or a map of
// names to {"source": path}, cleaned relative to the plugin root.
func paths(raw json.RawMessage) []string {
	var one string
	if json.Unmarshal(raw, &one) == nil {
		return []string{clean(one)}
	}
	var list []string
	if json.Unmarshal(raw, &list) == nil {
		for i, p := range list {
			list[i] = clean(p)
		}
		return list
	}
	var named map[string]struct {
		Source string `json:"source"`
	}
	var found []string
	if json.Unmarshal(raw, &named) == nil {
		for _, entry := range named {
			if entry.Source != "" {
				found = append(found, clean(entry.Source))
			}
		}
	}
	return found
}

func clean(p string) string {
	return strings.TrimPrefix(path.Clean(strings.TrimPrefix(p, "./")), "./")
}
