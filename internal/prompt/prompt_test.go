package prompt

import (
	"bufio"
	"errors"
	"regexp"
	"strings"
	"testing"
)

func scripted(keys string) (*Terminal, *strings.Builder) {
	var out strings.Builder
	return &Terminal{in: bufio.NewReader(strings.NewReader(keys)), out: &out}, &out
}

var ansi = regexp.MustCompile(`\x1b\[[0-9;?]*[A-Za-z]`)

func TestSelectMovesWithArrowsAndVimKeys(t *testing.T) {
	term, out := scripted("\x1b[B\x1b[Bk\r")
	options := []Option{{Label: "1 run"}, {Label: "3 runs"}, {Label: "5 runs"}}

	i, err := term.Select("Runs per case?", options, 0)

	if err != nil || i != 1 {
		t.Fatalf("got %d, %v", i, err)
	}
	if last := lastLine(out.String()); last != "✔ Runs per case?  3 runs" {
		t.Errorf("summary %q", last)
	}
}

func TestSelectWrapsAround(t *testing.T) {
	term, _ := scripted("\x1b[A\r")

	i, err := term.Select("Model?", []Option{{Label: "default"}, {Label: "sonnet"}, {Label: "opus"}}, 0)

	if err != nil || i != 2 {
		t.Errorf("got %d, %v", i, err)
	}
}

func TestCheckTogglesWithSpaceAndAll(t *testing.T) {
	term, out := scripted(" \x1b[B\x1b[B \r")
	options := []Option{{Label: "rails-expert"}, {Label: "ruby-expert", Checked: true}, {Label: "vite-expert"}}

	got, err := term.Check("Agents?", options)

	if err != nil {
		t.Fatal(err)
	}
	if !got[0].Checked || !got[1].Checked || !got[2].Checked {
		t.Errorf("checked %+v", got)
	}
	if options[0].Checked {
		t.Error("changed the caller's options")
	}
	if last := lastLine(out.String()); last != "✔ Agents?  rails-expert, ruby-expert, vite-expert" {
		t.Errorf("summary %q", last)
	}

	term, _ = scripted("aa \r")
	got, _ = term.Check("Agents?", options)
	if !got[0].Checked || got[1].Checked || got[2].Checked {
		t.Errorf("after all, none, all, toggle first: %+v", got)
	}
}

func TestEscapeAndControlCCancel(t *testing.T) {
	for _, keys := range []string{"\x1b", "\x03"} {
		term, _ := scripted(keys)
		if _, err := term.Select("Plugin?", []Option{{Label: "a"}}, 0); !errors.Is(err, ErrCanceled) {
			t.Errorf("%q: got %v, want ErrCanceled", keys, err)
		}
	}
}

func TestDrawShowsHintsBoxesAndPointer(t *testing.T) {
	term, out := scripted("\r")

	term.Check("Options?", []Option{{Label: "Run scaffolds", Hint: "as you", Checked: true}, {Label: "Judge with Sonnet"}})

	plain := ansi.ReplaceAllString(out.String(), "")
	for _, want := range []string{"? Options?", "❯ ● Run scaffolds      as you", "  ○ Judge with Sonnet\r\n"} {
		if !strings.Contains(plain, want) {
			t.Errorf("drawing lacks %q:\n%s", want, plain)
		}
	}
}

func lastLine(s string) string {
	lines := strings.Split(strings.TrimRight(ansi.ReplaceAllString(s, ""), "\r\n"), "\r\n")
	return strings.TrimLeft(lines[len(lines)-1], "\r")
}
