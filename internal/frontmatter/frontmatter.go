// Package frontmatter reads the YAML header of Markdown files: top-level
// scalars and lists, which is all that skills, commands, agents and eval
// graders use.
package frontmatter

import "strings"

// header returns the lines of a Markdown file's YAML header, between its
// first two --- lines, or nil when it has none.
func header(content []byte) []string {
	lines := strings.Split(strings.ReplaceAll(string(content), "\r\n", "\n"), "\n")
	if strings.TrimSpace(lines[0]) != "---" {
		return nil
	}
	for i, line := range lines[1:] {
		if strings.TrimSpace(line) == "---" {
			return lines[1 : i+1]
		}
	}
	return nil
}

// Scalar reads a top-level value, unquoting the YAML single- and
// double-quoted forms.
func Scalar(content []byte, key string) (string, bool) {
	for _, line := range header(content) {
		if value, found := strings.CutPrefix(line, key+":"); found {
			value = strings.TrimSpace(value)
			if len(value) >= 2 && value[0] == '\'' && value[len(value)-1] == '\'' {
				return strings.ReplaceAll(value[1:len(value)-1], "''", "'"), true
			}
			return unquote(value), true
		}
	}
	return "", false
}

// List reads a list field from a Markdown file's YAML header, in
// any of the three forms allowed-tools takes: "Read, Bash(git *)", "[Read,
// Grep]" or one "- item" per line. It handles top-level keys only, which is
// all that skills and commands use.
func List(content []byte, key string) []string {
	lines := header(content)
	for i, line := range lines {
		value, found := strings.CutPrefix(line, key+":")
		if !found {
			continue
		}
		value = strings.TrimSpace(value)
		if strings.HasPrefix(value, "[") {
			for _, next := range lines[i+1:] {
				if strings.Contains(value, "]") {
					break
				}
				value += " " + strings.TrimSpace(next)
			}
			value = strings.Trim(value, "[] ")
		}
		if value != "" {
			return splitTools(value)
		}
		var items []string
		for _, next := range lines[i+1:] {
			item, isItem := strings.CutPrefix(strings.TrimSpace(next), "- ")
			if !isItem {
				break
			}
			items = append(items, unquote(item))
		}
		return items
	}
	return nil
}

// splitTools splits on commas and spaces outside parentheses, so that
// "Bash(git add *) Read" gives two tools.
func splitTools(s string) []string {
	var tools []string
	depth, start := 0, 0
	for i, r := range s + " " {
		switch {
		case r == '(':
			depth++
		case r == ')':
			depth--
		case depth == 0 && (r == ',' || r == ' ' || r == '\t'):
			if tool := unquote(s[start:min(i, len(s))]); tool != "" {
				tools = append(tools, tool)
			}
			start = i + 1
		}
	}
	return tools
}

func unquote(s string) string {
	return strings.Trim(strings.TrimSpace(s), `"'`)
}
