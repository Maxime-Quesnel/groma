package frontmatter

import (
	"slices"
	"testing"
)

func TestListReadsEveryForm(t *testing.T) {
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
		if got := List([]byte(tc.content), "allowed-tools"); !slices.Equal(got, tc.want) {
			t.Errorf("%s: got %q, want %q", name, got, tc.want)
		}
	}
}

func TestScalarUnquotesYAMLStrings(t *testing.T) {
	content := []byte("---\ntype: tool_used\ninput_match: '\"subagent_type\"\\s*:\\s*\"p:a\"'\nnote: 'it''s'\nmin: 1\n---\nbody: no\n")
	for key, want := range map[string]string{"type": "tool_used", "input_match": `"subagent_type"\s*:\s*"p:a"`, "note": "it's", "min": "1"} {
		if got, ok := Scalar(content, key); !ok || got != want {
			t.Errorf("%s: got %q, %v; want %q", key, got, ok, want)
		}
	}
	if _, ok := Scalar(content, "body"); ok {
		t.Error("read a key from the body")
	}
}
