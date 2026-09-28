package claudecode

import (
	"slices"
	"testing"

	"github.com/Maxime-Quesnel/groma/internal/plugin"
)

func TestGrantsComeFromSkillsAndCommandsOnly(t *testing.T) {
	header := []byte("---\nallowed-tools: Bash\ntools: Bash\n---\n")
	var p plugin.Plugin
	for _, path := range []string{"skills/a/SKILL.md", "commands/b.md", "plugins/x/commands/c.md", "agents/d.md", "README.md"} {
		p.Files = append(p.Files, plugin.File{Path: path, Content: header})
	}

	var sources []string
	for _, g := range Grants(p) {
		sources = append(sources, g.Source)
	}

	if want := []string{"skills/a/SKILL.md", "commands/b.md", "plugins/x/commands/c.md"}; !slices.Equal(sources, want) {
		t.Errorf("got %v, want %v", sources, want)
	}
}
