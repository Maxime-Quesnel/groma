package rules

import (
	"bytes"
	"cmp"
	"regexp"
	"slices"
	"strings"
)

const (
	fetch       = `(?:curl|wget)`
	interpreter = `(?:(?:ba|z|da|k)?sh|fish|python[0-9.]*|perl|ruby|node|php|deno|bun)`
)

// Each pattern runs downloaded bytes as code. An interpreter only runs its
// standard input when nothing but flags follows it: "| python3 -m json.tool"
// reads data, "| bash -s -- --yes" runs it.
var runsDownload = []*regexp.Regexp{
	regexp.MustCompile(`\b` + fetch + `\b[^|\n]*\|\s*(?:sudo\s+(?:-\S+\s+)*)?` + interpreter + `(?:\s+-\S*)*(?:\s*(?:$|[;&|)\n])|["'` + "`" + `])`),
	regexp.MustCompile(`(?:^|[\s;&|(])(?:source|\.|` + interpreter + `)\s+<\(\s*` + fetch + `\b`),
	regexp.MustCompile(`(?:\beval|\b` + interpreter + `\s+-[ce])\s+["']?(?:\$\(|` + "`" + `)\s*` + fetch + `\b`),
}

var (
	savedDownload = regexp.MustCompile(`\b` + fetch + `\b[^\n;&|]*?\s(?:-[a-zA-Z]*[oO]|--output(?:-document)?)(?:\s+|=)["']?([^\s"';&|]+)`)
	verification  = regexp.MustCompile(`\b(?:sha(?:1|256|512)?sum|shasum|gpg\s+--verify|cosign\s+verify|minisign)\b`)
)

type codeLine struct {
	line int
	text string
}

// findRemoteCode returns the lines of text that download code and run it,
// either in one pipeline or by saving the download and running the file
// without verifying a checksum or signature first.
func findRemoteCode(text []byte) []codeLine {
	var found []codeLine
	report := func(offset int) {
		start := bytes.LastIndexByte(text[:offset], '\n') + 1
		end := bytes.IndexByte(text[start:], '\n')
		if end < 0 {
			end = len(text) - start
		}
		line := strings.TrimSpace(string(text[start : start+end]))
		n := bytes.Count(text[:start], []byte("\n")) + 1
		if strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//") ||
			slices.ContainsFunc(found, func(c codeLine) bool { return c.line == n }) {
			return
		}
		found = append(found, codeLine{line: n, text: line})
	}

	for _, re := range runsDownload {
		for _, m := range re.FindAllIndex(text, -1) {
			report(m[0])
		}
	}
	for _, m := range savedDownload.FindAllSubmatchIndex(text, -1) {
		file, rest := text[m[2]:m[3]], text[m[1]:]
		runs := regexp.MustCompile(`(?m)(?:(?:^|&&|\|\||;|\bthen|\bdo)\s*|(?:\b` + interpreter + `|\bsource|(?:^|\s)\.|\bchmod\s+\S*x\S*)\s+)["']?(?:\./)?` +
			regexp.QuoteMeta(string(file)) + `(?:["']|\s|$|;|&)`)
		if loc := runs.FindIndex(rest); loc != nil && !verification.Match(rest[:loc[0]]) {
			report(m[0])
		}
	}
	slices.SortFunc(found, func(a, b codeLine) int { return cmp.Compare(a.line, b.line) })
	return found
}
