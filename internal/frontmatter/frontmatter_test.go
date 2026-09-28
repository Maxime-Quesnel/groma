package frontmatter

import (
	"reflect"
	"strconv"
	"strings"
	"testing"
)

func TestParseReadsEveryValueForm(t *testing.T) {
	h := Parse([]byte(`---
name: pdf-tools   # a comment
description: "Extracts text from PDFs. Use when the user mentions \"PDF\"."
when_to_use: 'It''s for forms'
argument-hint: [file] [pages]
allowed-tools: Read, Bash(git add *, git commit *)
tools: [Read, Grep]
skills:
  - pdf
  - "forms"
disallowedTools:
- Write
long: |
  first line
    kept indent
folded: >-
  one
  two

  three
plain: starts here
  and goes on
hooks:
  PreToolUse:
    - matcher: Bash
metadata: {owner: docs}
---
# Body
`))

	want := map[string]Field{
		"name":            {Value: "pdf-tools"},
		"description":     {Value: `Extracts text from PDFs. Use when the user mentions "PDF".`},
		"when_to_use":     {Value: "It's for forms"},
		"argument-hint":   {Value: "[file] [pages]"},
		"allowed-tools":   {Value: "Read, Bash(git add *, git commit *)"},
		"tools":           {IsList: true, List: []string{"Read", "Grep"}},
		"skills":          {IsList: true, List: []string{"pdf", "forms"}},
		"disallowedTools": {IsList: true, List: []string{"Write"}},
		"long":            {Value: "first line\n  kept indent"},
		"folded":          {Value: "one two\nthree"},
		"plain":           {Value: "starts here and goes on"},
		"hooks":           {IsMap: true},
		"metadata":        {IsMap: true},
	}
	if len(h.Problems) > 0 {
		t.Fatalf("problems: %+v", h.Problems)
	}
	for key, w := range want {
		got, ok := h.Field(key)
		got.Key, got.Line, got.EndLine = "", 0, 0
		if !ok || !reflect.DeepEqual(got, w) {
			t.Errorf("%s: got %+v, want %+v", key, got, w)
		}
	}
	if f, _ := h.Field("allowed-tools"); !reflect.DeepEqual(f.Items(), []string{"Read", "Bash(git add *, git commit *)"}) {
		t.Errorf("items: %q", f.Items())
	}
	if f, _ := h.Field("tools"); f.Line != 7 || f.EndLine != 7 {
		t.Errorf("tools on lines %d-%d", f.Line, f.EndLine)
	}
	if f, _ := h.Field("skills"); f.Line != 8 || f.EndLine != 10 {
		t.Errorf("skills on lines %d-%d", f.Line, f.EndLine)
	}
	if h.Body != "# Body\n" || h.BodyLine != 28 {
		t.Errorf("body %q on line %d", h.Body, h.BodyLine)
	}
}

func TestParseReportsWhatYAMLRejects(t *testing.T) {
	tests := []struct {
		name, content, want string
	}{
		{"never closed", "---\nname: x\n# Body\n", "line 1: the header opened on line 1 is never closed by a --- line"},
		{"text line", "---\nname: x\nThis agent reviews code\n---\n", `line 3: "This agent reviews code" is not a key: value line`},
		{"missing space", "---\ndescription:Reviews code\n---\n", `line 2: "description:Reviews code" is not a key: value line`},
		{"set twice", "---\nname: a\nname: b\n---\n", "line 3: name is set twice, on lines 2 and 3"},
		{"tab", "---\ndescription: a\n\tb\n---\n", "line 3: a tab indents this line; YAML only allows spaces"},
		{"open quote", "---\ndescription: \"Reviews code\n---\n", "line 2: the quote opening description's value is never closed"},
		{"after quote", "---\ndescription: \"Reviews\" code\n---\n", "line 2: description's value goes on after its closing quote"},
		{"open list", "---\ntools: [Read, Grep\n---\n", "line 2: the [ opening tools's list is never closed"},
		{"reserved", "---\nargument-hint: @file\n---\n", "line 2: argument-hint's value starts with @, which YAML reserves: quote the value"},
		{"alias", "---\ntools: *\n---\n", "line 2: tools's value starts with *, which YAML reserves: quote the value"},
		{"stray indent", "---\n  name: x\n---\n", "line 2: this indented line belongs to no field"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := Parse([]byte(tt.content))
			var got []string
			for _, p := range h.Problems {
				got = append(got, "line "+strconv.Itoa(p.Line)+": "+p.Text)
			}
			if len(got) != 1 || got[0] != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseWithoutHeader(t *testing.T) {
	for content, misplaced := range map[string]bool{
		"# Title\n\nText\n":                false,
		"\n---\nname: x\n---\nText\n":      true,
		"Intro\n---\nname: x\n---\nText\n": false,
	} {
		h := Parse([]byte(content))
		if h.Found || h.Misplaced != misplaced || len(h.Fields) > 0 || h.Body != content || h.BodyLine != 1 {
			t.Errorf("%q: got %+v", content, h)
		}
	}
}

func TestParseSkipsAByteOrderMarkAndCRLF(t *testing.T) {
	h := Parse([]byte(string(rune(0xFEFF)) + "---\r\nname: x\r\n---\r\nBody\r\n"))

	if !h.Found || h.Value("name") != "x" || strings.Contains(h.Body, "\r") {
		t.Errorf("got %+v", h)
	}
}

func TestParseDocumentReadsAWholeFile(t *testing.T) {
	h := ParseDocument([]byte("# groma\ndisable:\n  - description-emphatic\nexclude: [legacy/**]\n"))

	if f, _ := h.Field("disable"); len(h.Problems) > 0 || f.Line != 2 || !reflect.DeepEqual(f.List, []string{"description-emphatic"}) {
		t.Errorf("got %+v", h)
	}
	if f, _ := h.Field("exclude"); !reflect.DeepEqual(f.List, []string{"legacy/**"}) {
		t.Errorf("exclude: %+v", f)
	}
}
