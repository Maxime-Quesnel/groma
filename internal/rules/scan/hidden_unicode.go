package scan

import (
	"fmt"
	"strconv"
	"unicode"
	"unicode/utf8"

	"github.com/Maxime-Quesnel/groma/internal/rule"
)

var hiddenUnicode = Rule{
	Meta: rule.Meta{
		ID:       "scan.hidden-unicode",
		Severity: rule.High,
		Title:    "Text a reviewer can't see",
		Description: "Unicode tag characters and runs of variation selectors can encode a whole hidden message that a language model reads and a human reviewer doesn't. " +
			"Bidirectional controls make code or instructions display differently from how they are parsed (Trojan Source, CVE-2021-42574). " +
			"Zero-width characters split words so that filters and reviewers miss them.",
		Remediation: "Remove the characters and review the file in an editor that shows invisible characters. Don't install someone else's plugin until they explain them.",
		FalsePositives: []string{
			"Subdivision flags such as England's use tag characters; groma skips them.",
			"Arabic or Hebrew text can legitimately contain bidirectional isolates.",
			"Zero-width joiners in emoji and non-joiners in Persian or Indic text; groma only flags them next to ASCII.",
			"Vendored dependencies, such as TypeScript in node_modules, ship zero-width characters in their data files.",
		},
		References: []string{
			"https://www.unicode.org/charts/PDF/UE0000.pdf",
			"https://trojansource.codes/",
		},
	},
	Check: func(f File) []string {
		return findHidden(string(f.Content))
	},
}

const (
	tagBase   = 0xE0000
	cancelTag = 0xE007F
	blackFlag = 0x1F3F4
)

func findHidden(text string) []string {
	runes := []rune(text)
	var hits []string
	line, lineStart := 1, 0
	for i := 0; i < len(runes); {
		r, end, what := runes[i], i+1, ""
		switch {
		case r == '\n':
			line, lineStart = line+1, i+1
		case isTag(r):
			end = runEnd(runes, i, isTag)
			if !subdivisionFlag(runes, i, end) {
				what = describeTags(runes[i:end])
			}
		case isVariationSelector(r):
			end = runEnd(runes, i, isVariationSelector)
			if end-i >= 2 {
				what = describeVariationSelectors(runes[i:end])
			}
		case isBidiControl(r):
			end = runEnd(runes, i, isBidiControl)
			what = fmt.Sprintf("%d bidirectional control character(s) starting with %U", end-i, r)
		case isZeroWidth(r):
			if r != '\uFEFF' || i > 0 {
				what = fmt.Sprintf("zero-width character %U", r)
			}
		case unicode.Is(unicode.Join_Control, r):
			if nextToASCII(runes, i) {
				what = fmt.Sprintf("zero-width joiner %U next to ASCII text", r)
			}
		}
		if what != "" {
			hits = append(hits, fmt.Sprintf("line %d, column %d: %s", line, i-lineStart+1, what))
		}
		i = end
	}
	return hits
}

func runEnd(runes []rune, start int, in func(rune) bool) int {
	end := start
	for end < len(runes) && in(runes[end]) {
		end++
	}
	return end
}

func isTag(r rune) bool {
	return r >= tagBase && r <= cancelTag
}

func isVariationSelector(r rune) bool {
	return (r >= 0xFE00 && r <= 0xFE0F) || (r >= 0xE0100 && r <= 0xE01EF)
}

// Not unicode.Bidi_Control: it adds the Arabic letter mark and the
// left-to-right and right-to-left marks, which ordinary right-to-left text uses.
func isBidiControl(r rune) bool {
	return (r >= 0x202A && r <= 0x202E) || (r >= 0x2066 && r <= 0x2069)
}

func isZeroWidth(r rune) bool {
	switch r {
	case '\u200B', '\u2060', '\u180E', '\u2061', '\u2062', '\u2063', '\u2064', '\uFEFF':
		return true
	}
	return false
}

func subdivisionFlag(runes []rune, start, end int) bool {
	if start == 0 || runes[start-1] != blackFlag || runes[end-1] != cancelTag {
		return false
	}
	code := runes[start : end-1]
	if len(code) < 3 || len(code) > 7 {
		return false
	}
	for _, r := range code {
		c := r - tagBase
		if !(c >= 'a' && c <= 'z') && !(c >= '0' && c <= '9') {
			return false
		}
	}
	return true
}

// Tag characters U+E0020 to U+E007E mirror printable ASCII.
func describeTags(tags []rune) string {
	var hidden []rune
	for _, r := range tags {
		if c := r - tagBase; c >= 0x20 && c <= 0x7E {
			hidden = append(hidden, c)
		}
	}
	if len(hidden) == 0 {
		return fmt.Sprintf("%d Unicode tag character(s)", len(tags))
	}
	return fmt.Sprintf("%d Unicode tag characters hiding the text %s", len(tags), quote(string(hidden)))
}

// Runs of variation selectors can carry one byte each: U+FE00 to U+FE0F for
// bytes 0 to 15, U+E0100 to U+E01EF for bytes 16 to 255.
func describeVariationSelectors(selectors []rune) string {
	hidden := make([]byte, len(selectors))
	for i, r := range selectors {
		if r <= 0xFE0F {
			hidden[i] = byte(r - 0xFE00)
		} else {
			hidden[i] = byte(r - 0xE0100 + 16)
		}
	}
	return fmt.Sprintf("%d variation selectors hiding the text %s", len(selectors), quote(string(hidden)))
}

func nextToASCII(runes []rune, i int) bool {
	ascii := func(j int) bool {
		return j >= 0 && j < len(runes) && runes[j] < utf8.RuneSelf && runes[j] > ' '
	}
	return ascii(i-1) || ascii(i+1)
}

const maxQuoted = 80

func quote(s string) string {
	runes := []rune(s)
	if len(runes) > maxQuoted {
		return strconv.Quote(string(runes[:maxQuoted])) + "…"
	}
	return strconv.Quote(s)
}
