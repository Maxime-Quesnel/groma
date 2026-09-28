package style

import (
	"regexp"
	"strings"
	"testing"
)

func TestColorsOnlyInATerminal(t *testing.T) {
	if got := For(&strings.Builder{}).Red("x"); got != "x" {
		t.Errorf("colored a non-terminal writer: %q", got)
	}
	if got := On().Red("x"); got != "\x1b[31mx\x1b[0m" {
		t.Errorf("On didn't color: %q", got)
	}
	if got := On().Red(""); got != "" {
		t.Errorf("colored an empty string: %q", got)
	}
}

func TestPadStylesAfterPadding(t *testing.T) {
	if got := Pad("ab", 4, On().Bold); got != "\x1b[1mab  \x1b[0m" {
		t.Errorf("got %q", got)
	}
}

func TestTableSizesColumnsOnPlainText(t *testing.T) {
	var out strings.Builder

	On().Table(&out, []string{"Agent", "Recall"}, [][]Cell{
		{Plain("rails-expert"), {"96 %", On().Green("96 %")}},
		{Plain("ruby"), Plain("100 %")},
	})

	plain := regexp.MustCompile(`\x1b\[[0-9;]*m`).ReplaceAllString(out.String(), "")
	want := `┌──────────────┬────────┐
│ Agent        │ Recall │
├──────────────┼────────┤
│ rails-expert │ 96 %   │
│ ruby         │ 100 %  │
└──────────────┴────────┘
`
	if plain != want {
		t.Errorf("got:\n%s\nwant:\n%s", plain, want)
	}
}
