package rules

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/Maxime-Quesnel/groma/internal/claudecode"
	"github.com/Maxime-Quesnel/groma/internal/component"
	"github.com/Maxime-Quesnel/groma/internal/rule"
)

var nonPortablePath = Rule{
	Meta: rule.Meta{
		ID:    "non-portable-path",
		Level: rule.Warning,
		Title: "Path that only works on the author's machine",
		Description: "A path into someone's home directory, such as /Users/alice/ or /home/alice/, or a Windows path with backslashes, " +
			"points nowhere on another machine. Anthropic's guidance is forward slashes, and paths through ${CLAUDE_PLUGIN_ROOT}, ${CLAUDE_SKILL_DIR} or ${CLAUDE_PROJECT_DIR}.",
		Remediation: "Write the path relative to the component, with forward slashes, through the directory variables Claude Code provides.",
		FalsePositives: []string{
			"An example path with a placeholder user such as /home/user/ or /Users/you/, which groma skips.",
			"A path the component tells the user to adapt to their machine.",
		},
		References: []string{claudecode.BestPractices},
	},
	Kinds: everything,
	Check: func(c *component.Component, t *component.Tree) []string {
		var evidence []string
		if c.Kind == component.Hooks {
			for _, cmd := range commands(c) {
				if p := nonPortable(cmd.run); p != "" {
					evidence = append(evidence, fmt.Sprintf("%s uses %s", cmd.where, p))
				}
			}
			return evidence
		}
		for i, line := range strings.Split(string(c.Content), "\n") {
			if p := nonPortable(line); p != "" {
				evidence = append(evidence, at(i+1, "%s", p))
			}
		}
		return evidence
	},
}

var (
	homePath      = regexp.MustCompile(`(?:^|[\s"'(=:])(/(?:Users|home)/([A-Za-z0-9._-]+)/\S*|[A-Za-z]:\\Users\\([A-Za-z0-9._-]+)\\\S*)`)
	backslashPath = regexp.MustCompile(`\b[\w.-]+\\[\w.-]+\.(?:md|py|sh|js|ts|rb|json|ya?ml|txt)\b`)
	placeholders  = []string{"user", "username", "you", "me", "name", "yourname", "your-name", "example", "foo"}
)

func nonPortable(line string) string {
	for _, m := range homePath.FindAllStringSubmatch(line, -1) {
		if user := m[2] + m[3]; !slices.Contains(placeholders, strings.ToLower(user)) {
			return m[1]
		}
	}
	return backslashPath.FindString(line)
}
