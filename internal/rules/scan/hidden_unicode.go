package scan

import (
	"fmt"
	"strconv"
	"unicode"
	"unicode/utf8"

	"github.com/Maxime-Quesnel/groma/internal/rule"
)

type hiddenUnicode struct{}

func (hiddenUnicode) Meta() rule.Meta {
	return rule.Meta{
		ID:       "scan.hidden-unicode",
		Severity: rule.High,
		Title:    "Text a reviewer can't see",
		Description: "The file contains characters that render as nothing, or that reorder what is displayed. " +
			"Unicode tag characters and runs of variation selectors can encode a whole hidden message that a language model reads and a human reviewer doesn't. " +
			"Bidirectional controls make code or instructions display differently from how they are parsed (Trojan Source, CVE-2021-42574). " +
			"Zero-width characters split words so that filters and reviewers miss them.",
		Remediation: "Remove the characters, then review the file in an editor that shows invisible characters. If the plugin comes from someone else, don't install it until they explain them.",
		FalsePositives: []string{
			"Subdivision flag emoji such as England's use tag characters; groma skips them.",
			"Right-to-left text such as Arabic or Hebrew can legitimately contain bidirectional isolates.",
			"Zero-width joiners inside emoji sequences, and zero-width non-joiners in Persian or Indic text; groma only flags them next to ASCII.",
			"A byte order mark at the start of a file is allowed; one elsewhere is flagged.",
		},
		References: []string{
			"https://www.unicode.org/charts/PDF/UE0000.pdf",
			"https://trojansource.codes/",
		},
	}
}

const maxEvidencePerFile = 5

func (r hiddenUnicode) Check(facts Facts) []rule.Finding {
	var findings []rule.Finding
	for _, f := range facts.Files {
		hits := findHidden(string(f.Content))
		if len(hits) == 0 {
			continue
		}
		evidence := make([]string, 0, min(len(hits), maxEvidencePerFile)+1)
		for _, h := range hits[:min(len(hits), maxEvidencePerFile)] {
			evidence = append(evidence, fmt.Sprintf("line %d, column %d: %s", h.line, h.column, h.what))
		}
		if extra := len(hits) - maxEvidencePerFile; extra > 0 {
			evidence = append(evidence, fmt.Sprintf("and %d more", extra))
		}
		findings = append(findings, rule.Finding{Rule: r.Meta(), Subject: f.Path, Evidence: evidence})
	}
	return findings
}

type hit struct {
	line, column int
	what         string
}

func findHidden(text string) []hit {
	runes := []rune(text)
	var hits []hit
	line, column := 1, 1
	advance := func(from, to int) {
		for _, r := range runes[from:to] {
			if r == '\n' {
				line, column = line+1, 1
			} else {
				column++
			}
		}
	}

	for i := 0; i < len(runes); {
		r := runes[i]
		end := i + 1
		var what string
		switch {
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
			if !(r == '\uFEFF' && i == 0) {
				what = fmt.Sprintf("zero-width character %U", r)
			}
		case r == '\u200C' || r == '\u200D':
			if nextToASCII(runes, i) {
				what = fmt.Sprintf("zero-width joiner %U next to ASCII text", r)
			}
		}
		if what != "" {
			hits = append(hits, hit{line: line, column: column, what: what})
		}
		advance(i, end)
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
	return r >= 0xE0000 && r <= 0xE007F
}

func isVariationSelector(r rune) bool {
	return (r >= 0xFE00 && r <= 0xFE0F) || (r >= 0xE0100 && r <= 0xE01EF)
}

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

// subdivisionFlag recognises the emoji flags of England, Scotland and Wales:
// a black flag, a short lowercase region code in tag characters, a cancel tag.
func subdivisionFlag(runes []rune, start, end int) bool {
	if start == 0 || runes[start-1] != 0x1F3F4 || runes[end-1] != 0xE007F {
		return false
	}
	code := runes[start : end-1]
	if len(code) < 3 || len(code) > 7 {
		return false
	}
	for _, r := range code {
		c := r - 0xE0000
		if !(c >= 'a' && c <= 'z') && !(c >= '0' && c <= '9') {
			return false
		}
	}
	return true
}

// Tag characters U+E0020 to U+E007E mirror printable ASCII, which is how
// "ASCII smuggling" hides instructions from humans but not from models.
func describeTags(tags []rune) string {
	var hidden []rune
	for _, r := range tags {
		if c := r - 0xE0000; c >= 0x20 && c <= 0x7E {
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
	if utf8.Valid(hidden) && printable(string(hidden)) {
		return fmt.Sprintf("%d variation selectors hiding the text %s", len(selectors), quote(string(hidden)))
	}
	return fmt.Sprintf("%d variation selectors in a row, enough to hide %d bytes", len(selectors), len(selectors))
}

func printable(s string) bool {
	for _, r := range s {
		if !unicode.IsPrint(r) && !unicode.IsSpace(r) {
			return false
		}
	}
	return true
}

func nextToASCII(runes []rune, i int) bool {
	ascii := func(j int) bool {
		return j >= 0 && j < len(runes) && runes[j] < utf8.RuneSelf && runes[j] > ' '
	}
	return ascii(i-1) || ascii(i+1)
}

const maxQuoted = 80

// quote escapes whatever it prints, so hidden text can't carry terminal
// escape sequences into the report.
func quote(s string) string {
	runes := []rune(s)
	if len(runes) > maxQuoted {
		return strconv.Quote(string(runes[:maxQuoted])) + "…"
	}
	return strconv.Quote(s)
}
