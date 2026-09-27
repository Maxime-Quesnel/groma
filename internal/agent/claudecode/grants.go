package claudecode

import (
	"path"
	"strings"

	"github.com/Maxime-Quesnel/groma/internal/plugin"
)

// Grants lists the tools that skills and commands pre-approve with
// allowed-tools. Agents' tools field restricts what they may use instead of
// pre-approving it, so it grants nothing.
func Grants(p plugin.Plugin) []plugin.Grant {
	var grants []plugin.Grant
	for _, f := range p.Files {
		dir, base := path.Split(f.Path)
		skill := base == "SKILL.md"
		command := strings.HasSuffix(base, ".md") && (strings.HasPrefix(dir, "commands/") || strings.Contains(dir, "/commands/"))
		if !skill && !command {
			continue
		}
		for _, tool := range frontmatterList(f.Content, "allowed-tools") {
			grants = append(grants, plugin.Grant{Tool: tool, Source: f.Path})
		}
	}
	return grants
}

// frontmatterList reads a list field from a Markdown file's YAML header, in
// any of the three forms allowed-tools takes: "Read, Bash(git *)", "[Read,
// Grep]" or one "- item" per line. It handles top-level keys only, which is
// all that skills and commands use.
func frontmatterList(content []byte, key string) []string {
	lines := strings.Split(strings.ReplaceAll(string(content), "\r\n", "\n"), "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return nil
	}
	for i, line := range lines[1:] {
		if strings.TrimSpace(line) == "---" {
			return nil
		}
		value, found := strings.CutPrefix(line, key+":")
		if !found {
			continue
		}
		value = strings.TrimSpace(value)
		if strings.HasPrefix(value, "[") {
			for _, next := range lines[i+2:] {
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
		for _, next := range lines[i+2:] {
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
