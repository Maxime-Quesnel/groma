// Package prompt asks the user to choose in the terminal with the arrow keys,
// the space bar and enter. It draws with ANSI escapes and puts the terminal in
// raw mode through stty, so it needs no dependency.
package prompt

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"slices"
	"strings"

	"github.com/Maxime-Quesnel/groma/internal/style"
)

type Option struct {
	Label string
	// Hint is shown dimmed after the label.
	Hint    string
	Checked bool
}

var ErrCanceled = errors.New("canceled")

type Terminal struct {
	in    *bufio.Reader
	out   io.Writer
	close func() error
}

const hideCursor, showCursor = "\x1b[?25l", "\x1b[?25h"

// Prompts always draw on a terminal, so they always color.
var st = style.On()

// Open takes over the controlling terminal until Close restores it.
func Open() (*Terminal, error) {
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return nil, err
	}
	saved, err := stty(tty, "-g")
	if err == nil {
		_, err = stty(tty, "raw", "-echo")
	}
	if err != nil {
		tty.Close()
		return nil, err
	}
	fmt.Fprint(tty, hideCursor)
	return &Terminal{in: bufio.NewReader(tty), out: tty, close: func() error {
		fmt.Fprint(tty, showCursor)
		_, err := stty(tty, strings.TrimSpace(saved))
		return errors.Join(err, tty.Close())
	}}, nil
}

// New reads keys from in and draws on out, for callers that bring their own
// terminal, such as tests.
func New(in io.Reader, out io.Writer) *Terminal {
	return &Terminal{in: bufio.NewReader(in), out: out}
}

func (t *Terminal) Close() error {
	if t.close == nil {
		return nil
	}
	return t.close()
}

func stty(tty *os.File, args ...string) (string, error) {
	cmd := exec.Command("stty", args...)
	cmd.Stdin = tty
	out, err := cmd.Output()
	return string(out), err
}

// Say writes a line that stays above the prompts.
func (t *Terminal) Say(format string, a ...any) {
	fmt.Fprintf(t.out, format+"\r\n", a...)
}

// Done records an answer the way a prompt does once answered, for answers
// that need no question, such as the only plugin there is.
func (t *Terminal) Done(title, answer string) {
	t.Say("%s %s  %s", st.Green("✔"), st.Bold(title), st.Cyan(answer))
}

// Select lets the user pick one option, starting on cursor, and returns its
// index.
func (t *Terminal) Select(title string, options []Option, cursor int) (int, error) {
	i, _, err := t.choose(title, options, cursor, false)
	return i, err
}

// Check lets the user check and uncheck options, and returns them in their
// final state.
func (t *Terminal) Check(title string, options []Option) ([]Option, error) {
	_, opts, err := t.choose(title, options, 0, true)
	return opts, err
}

func (t *Terminal) choose(title string, options []Option, cursor int, multi bool) (int, []Option, error) {
	opts := slices.Clone(options)
	drawn := 0
	for {
		drawn = t.draw(title, opts, cursor, multi, drawn)
		k, err := readKey(t.in)
		if err != nil {
			return 0, nil, err
		}
		switch k {
		case keyUp:
			cursor = (cursor + len(opts) - 1) % len(opts)
		case keyDown:
			cursor = (cursor + 1) % len(opts)
		case keySpace:
			if multi {
				opts[cursor].Checked = !opts[cursor].Checked
			}
		case keyAll:
			if multi {
				all := !slices.ContainsFunc(opts, func(o Option) bool { return !o.Checked })
				for i := range opts {
					opts[i].Checked = !all
				}
			}
		case keyEnter:
			t.erase(drawn)
			t.Done(title, answer(opts, cursor, multi))
			return cursor, opts, nil
		case keyCancel:
			t.erase(drawn)
			return 0, nil, ErrCanceled
		}
	}
}

func (t *Terminal) draw(title string, opts []Option, cursor int, multi bool, drawn int) int {
	t.erase(drawn)
	help := "↑↓ move · enter choose"
	if multi {
		help = "↑↓ move · space check · a all · enter confirm"
	}
	t.Say("%s %s  %s", st.Cyan("?"), st.Bold(title), st.Dim(help))
	// Labels share one width, so that hints line up in a column.
	width := 0
	for _, o := range opts {
		width = max(width, len([]rune(o.Label)))
	}
	for i, o := range opts {
		pointer, label := "  ", style.Pad(o.Label, width, func(s string) string { return s })
		if i == cursor {
			pointer, label = st.Cyan("❯ "), style.Pad(o.Label, width, st.Cyan)
		}
		box := ""
		if multi {
			box = st.Dim("○ ")
			if o.Checked {
				box = st.Green("● ")
			}
		}
		hint := ""
		if o.Hint != "" {
			hint = "  " + st.Dim(o.Hint)
		}
		t.Say("%s%s%s", pointer, box, strings.TrimRight(label+hint, " "))
	}
	return len(opts) + 1
}

// erase moves back over the lines drawn last and clears them.
func (t *Terminal) erase(lines int) {
	if lines > 0 {
		fmt.Fprintf(t.out, "\x1b[%dA\r\x1b[J", lines)
	}
}

func answer(opts []Option, cursor int, multi bool) string {
	if !multi {
		return opts[cursor].Label
	}
	var checked []string
	for _, o := range opts {
		if o.Checked {
			checked = append(checked, o.Label)
		}
	}
	if len(checked) == 0 {
		return "none"
	}
	return strings.Join(checked, ", ")
}

type key int

const (
	keyNone key = iota
	keyUp
	keyDown
	keySpace
	keyAll
	keyEnter
	keyCancel
)

// readKey reads one keystroke. Arrows arrive as escape sequences, ESC [ A
// and ESC [ B, or ESC O A in application mode; a lone ESC cancels.
func readKey(r *bufio.Reader) (key, error) {
	b, err := r.ReadByte()
	if err != nil {
		return keyNone, err
	}
	switch b {
	case '\r', '\n':
		return keyEnter, nil
	case ' ':
		return keySpace, nil
	case 'a':
		return keyAll, nil
	case 'k':
		return keyUp, nil
	case 'j':
		return keyDown, nil
	case 3: // ctrl-c, which raw mode delivers as a byte instead of a signal
		return keyCancel, nil
	case 27:
		if r.Buffered() == 0 {
			return keyCancel, nil
		}
		if next, _ := r.ReadByte(); next != '[' && next != 'O' {
			return keyNone, nil
		}
		switch arrow, _ := r.ReadByte(); arrow {
		case 'A':
			return keyUp, nil
		case 'B':
			return keyDown, nil
		}
	}
	return keyNone, nil
}
