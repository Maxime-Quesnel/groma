// Package style colors what groma prints, and only in a terminal: output
// piped to a file or a CI log, or with NO_COLOR set, stays plain text.
package style

import (
	"fmt"
	"io"
	"os"
	"strings"
	"unicode/utf8"
)

type Style struct{ on bool }

// For returns the style to write to w with.
func For(w io.Writer) Style {
	f, ok := w.(*os.File)
	if !ok || os.Getenv("NO_COLOR") != "" {
		return Style{}
	}
	info, err := f.Stat()
	return Style{on: err == nil && info.Mode()&os.ModeCharDevice != 0}
}

// On returns a style that always colors, for tests and for prompts drawn on
// the terminal itself.
func On() Style { return Style{on: true} }

func (s Style) paint(code, text string) string {
	if !s.on || text == "" {
		return text
	}
	return "\x1b[" + code + "m" + text + "\x1b[0m"
}

func (s Style) Bold(text string) string    { return s.paint("1", text) }
func (s Style) Dim(text string) string     { return s.paint("2", text) }
func (s Style) Red(text string) string     { return s.paint("31", text) }
func (s Style) Green(text string) string   { return s.paint("32", text) }
func (s Style) Yellow(text string) string  { return s.paint("33", text) }
func (s Style) Blue(text string) string    { return s.paint("34", text) }
func (s Style) Cyan(text string) string    { return s.paint("36", text) }
func (s Style) BoldRed(text string) string { return s.paint("1;31", text) }

// Badge prints text as white on red, for what must stand out most.
func (s Style) Badge(text string) string { return s.paint("1;97;41", text) }

// Pad pads text to width before styling it, so colors never break column
// alignment.
func Pad(text string, width int, paint func(string) string) string {
	return paint(fmt.Sprintf("%-*s", width, text))
}

// Cell is one table value: Text is what it reads as, Styled what it prints
// as. Column widths come from Text, so colors never shift the borders.
type Cell struct{ Text, Styled string }

func Plain(text string) Cell { return Cell{text, text} }

// Table draws rows under headers, in a box.
func (s Style) Table(w io.Writer, headers []string, rows [][]Cell) {
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = width(h)
	}
	for _, r := range rows {
		for i, c := range r {
			widths[i] = max(widths[i], width(c.Text))
		}
	}
	rule := func(left, mid, right string) string {
		parts := make([]string, len(widths))
		for i, n := range widths {
			parts[i] = strings.Repeat("─", n+2)
		}
		return s.Dim(left + strings.Join(parts, mid) + right)
	}
	row := func(cells []Cell) string {
		var b strings.Builder
		b.WriteString(s.Dim("│"))
		for i, c := range cells {
			b.WriteString(" " + c.Styled + strings.Repeat(" ", widths[i]-width(c.Text)) + " " + s.Dim("│"))
		}
		return b.String()
	}
	head := make([]Cell, len(headers))
	for i, h := range headers {
		head[i] = Cell{h, s.Bold(h)}
	}
	fmt.Fprintln(w, rule("┌", "┬", "┐"))
	fmt.Fprintln(w, row(head))
	fmt.Fprintln(w, rule("├", "┼", "┤"))
	for _, r := range rows {
		fmt.Fprintln(w, row(r))
	}
	fmt.Fprintln(w, rule("└", "┴", "┘"))
}

func width(text string) int { return utf8.RuneCountInString(text) }
