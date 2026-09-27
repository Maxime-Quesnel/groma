package claudecode

import (
	"path"
	"strings"

	"github.com/Maxime-Quesnel/groma/internal/frontmatter"
	"github.com/Maxime-Quesnel/groma/internal/plugin"
)

// Grants lists the tools that skills and commands pre-approve with
// allowed-tools. Agents' tools field restricts what they may use instead of
// pre-approving it, so it grants nothing.
func Grants(p plugin.Plugin) []plugin.Grant {
	var grants []plugin.Grant
	for _, f := range p.Files {
		if !skillOrCommand(f.Path) {
			continue
		}
		for _, tool := range frontmatter.List(f.Content, "allowed-tools") {
			grants = append(grants, plugin.Grant{Tool: tool, Source: f.Path})
		}
	}
	return grants
}

func skillOrCommand(file string) bool {
	dir, base := path.Split(file)
	return base == "SKILL.md" || strings.HasSuffix(base, ".md") && (strings.HasPrefix(dir, "commands/") || strings.Contains(dir, "/commands/"))
}
