// Package fix applies the edits rules propose to the files they touch, and
// shows the result as a unified diff. Edits change text in place, so a file
// keeps its layout, comments and quoting everywhere else.
package fix

import (
	"cmp"
	"fmt"
	"slices"
	"strings"
)

type Edit struct {
	// Path is the file, relative to the tree's root.
	Path string
	// Start and End delimit the bytes New replaces; an insertion has
	// Start == End.
	Start, End int
	New        string
	// Executable asks to make the file executable instead of changing its
	// text.
	Executable bool
	// Rule is the ID of the rule that proposes the edit.
	Rule string
}

// A File is one file's content before and after its edits.
type File struct {
	Path           string
	Old, New       []byte
	MakeExecutable bool
	// Rules counts the edits applied to the file by rule ID.
	Rules map[string]int
}

// Plan applies the edits to the files they touch, reading each file with
// read. It keeps one copy of identical edits, and drops an edit that
// overlaps one applied before it: the next run fixes what it would have.
func Plan(edits []Edit, read func(path string) ([]byte, bool)) (files []File, dropped int) {
	byPath := map[string][]Edit{}
	var paths []string
	for _, e := range edits {
		if _, seen := byPath[e.Path]; !seen {
			paths = append(paths, e.Path)
		}
		byPath[e.Path] = append(byPath[e.Path], e)
	}
	slices.Sort(paths)
	for _, p := range paths {
		old, ok := read(p)
		if !ok {
			dropped += len(byPath[p])
			continue
		}
		f := File{Path: p, Old: old, Rules: map[string]int{}}
		var text []Edit
		for _, e := range byPath[p] {
			if e.Executable {
				if !f.MakeExecutable {
					f.MakeExecutable = true
					f.Rules[e.Rule]++
				}
				continue
			}
			text = append(text, e)
		}
		slices.SortStableFunc(text, func(a, b Edit) int { return cmp.Or(cmp.Compare(a.Start, b.Start), cmp.Compare(a.End, b.End)) })
		var b strings.Builder
		at := 0
		var last *Edit
		for i := range text {
			e := text[i]
			switch {
			case e.Start < 0 || e.End > len(old) || e.Start > e.End:
				dropped++
				continue
			case last != nil && e.Start == last.Start && e.End == last.End && e.New == last.New:
				continue
			case e.Start < at || last != nil && e.Start == last.Start && e.Start == e.End:
				dropped++
				continue
			}
			b.Write(old[at:e.Start])
			b.WriteString(e.New)
			at = e.End
			last = &text[i]
			f.Rules[e.Rule]++
		}
		b.Write(old[at:])
		f.New = []byte(b.String())
		if string(f.New) != string(f.Old) || f.MakeExecutable {
			files = append(files, f)
		}
	}
	return files, dropped
}

// Diff returns the unified diff between old and new, with three lines of
// context around each change, one line per string.
func Diff(path string, old, new []byte) []string {
	a, b := lines(old), lines(new)
	prefix := 0
	for prefix < len(a) && prefix < len(b) && a[prefix] == b[prefix] {
		prefix++
	}
	suffix := 0
	for suffix < len(a)-prefix && suffix < len(b)-prefix && a[len(a)-1-suffix] == b[len(b)-1-suffix] {
		suffix++
	}
	var ops []op
	for _, l := range a[:prefix] {
		ops = append(ops, op{' ', l})
	}
	ops = append(ops, myers(a[prefix:len(a)-suffix], b[prefix:len(b)-suffix])...)
	for _, l := range a[len(a)-suffix:] {
		ops = append(ops, op{' ', l})
	}
	if !slices.ContainsFunc(ops, func(o op) bool { return o.kind != ' ' }) {
		return nil
	}
	return append([]string{"--- a/" + path, "+++ b/" + path}, hunks(ops, 3)...)
}

type op struct {
	kind byte
	line string
}

func lines(content []byte) []string {
	s := strings.TrimSuffix(string(content), "\n")
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}

// myers returns the shortest edit script from a to b.
func myers(a, b []string) []op {
	n, m := len(a), len(b)
	limit := n + m
	off := limit + 1
	v := make([]int, 2*limit+3)
	var trace [][]int
search:
	for d := 0; d <= limit; d++ {
		trace = append(trace, slices.Clone(v))
		for k := -d; k <= d; k += 2 {
			x := v[off+k-1] + 1
			if k == -d || k != d && v[off+k-1] < v[off+k+1] {
				x = v[off+k+1]
			}
			y := x - k
			for x < n && y < m && a[x] == b[y] {
				x, y = x+1, y+1
			}
			v[off+k] = x
			if x >= n && y >= m {
				break search
			}
		}
	}
	var ops []op
	x, y := n, m
	for d := len(trace) - 1; d >= 0; d-- {
		v := trace[d]
		k := x - y
		prevK := k - 1
		if k == -d || k != d && v[off+k-1] < v[off+k+1] {
			prevK = k + 1
		}
		prevX := v[off+prevK]
		prevY := prevX - prevK
		for x > prevX && y > prevY {
			ops = append(ops, op{' ', a[x-1]})
			x, y = x-1, y-1
		}
		if d > 0 {
			if x == prevX {
				ops = append(ops, op{'+', b[y-1]})
			} else {
				ops = append(ops, op{'-', a[x-1]})
			}
		}
		x, y = prevX, prevY
	}
	slices.Reverse(ops)
	return ops
}

// hunks groups an edit script into unified-diff hunks.
func hunks(ops []op, context int) []string {
	var out []string
	oldLine, newLine := 1, 1
	for i := 0; i < len(ops); {
		if ops[i].kind == ' ' {
			oldLine, newLine, i = oldLine+1, newLine+1, i+1
			continue
		}
		start := max(0, i-context)
		oldStart, newStart := oldLine-(i-start), newLine-(i-start)
		end, quiet := i, 0
		for j := i; j < len(ops) && quiet <= 2*context; j++ {
			if ops[j].kind == ' ' {
				quiet++
			} else {
				quiet, end = 0, j
			}
		}
		stop := min(len(ops), end+1+context)
		var body []string
		oldCount, newCount := 0, 0
		for _, o := range ops[start:stop] {
			body = append(body, string(o.kind)+o.line)
			if o.kind != '+' {
				oldCount++
			}
			if o.kind != '-' {
				newCount++
			}
		}
		out = append(out, fmt.Sprintf("@@ -%d,%d +%d,%d @@", oldStart, oldCount, newStart, newCount))
		out = append(out, body...)
		for _, o := range ops[i:stop] {
			if o.kind != '+' {
				oldLine++
			}
			if o.kind != '-' {
				newLine++
			}
		}
		i = stop
	}
	return out
}
