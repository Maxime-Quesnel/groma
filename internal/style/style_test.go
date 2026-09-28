package style

import (
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
