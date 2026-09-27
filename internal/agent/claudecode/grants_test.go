package claudecode

import (
	"slices"
	"testing"

	"github.com/Maxime-Quesnel/groma/internal/plugin"
)

func TestFrontmatterListReadsEveryForm(t *testing.T) {
	for name, tc := range map[string]struct {
		content string
		want    []string
	}{
		"inline":         {"---\nallowed-tools: Read, Bash(git add *) Bash(git commit *)\n---\n", []string{"Read", "Bash(git add *)", "Bash(git commit *)"}},
		"flow":           {"---\nallowed-tools: [Read, \"Bash(*)\"]\n---\n", []string{"Read", "Bash(*)"}},
		"flow on lines":  {"---\nallowed-tools: [\n  Read,\n  'Bash(ls *)'\n]\nname: x\n---\n", []string{"Read", "Bash(ls *)"}},
		"block list":     {"---\nname: x\nallowed-tools:\n  - Read\n  - \"Bash(node:*)\"\nmodel: sonnet\n---\n", []string{"Read", "Bash(node:*)"}},
		"windows lines":  {"---\r\nallowed-tools: Bash\r\n---\r\n", []string{"Bash"}},
		"only in body":   {"---\nname: x\n---\nallowed-tools: Bash\n", nil},
		"no frontmatter": {"allowed-tools: Bash\n", nil},
	} {
		if got := frontmatterList([]byte(tc.content), "allowed-tools"); !slices.Equal(got, tc.want) {
			t.Errorf("%s: got %q, want %q", name, got, tc.want)
		}
	}
}

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
