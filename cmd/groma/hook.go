package main

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Maxime-Quesnel/groma/internal/component"
	"github.com/Maxime-Quesnel/groma/internal/config"
	"github.com/Maxime-Quesnel/groma/internal/report"
	"github.com/Maxime-Quesnel/groma/internal/rule"
	"github.com/Maxime-Quesnel/groma/internal/rules"
)

// runHook is Claude Code's PostToolUse hook, which the groma plugin runs after
// every Edit and Write. It checks the component the edited file belongs to,
// and tells Claude what it found: red flags as a block decision Claude must
// answer, warnings as context. It never fails the edit: on anything it can't
// read, it says nothing and exits 0.
func runHook(stdin io.Reader, stdout io.Writer) int {
	var input struct {
		Cwd       string `json:"cwd"`
		ToolInput struct {
			FilePath string `json:"file_path"`
		} `json:"tool_input"`
	}
	if json.NewDecoder(stdin).Decode(&input) != nil || input.ToolInput.FilePath == "" {
		return 0
	}
	file := input.ToolInput.FilePath
	if !filepath.IsAbs(file) {
		file = filepath.Join(input.Cwd, file)
	}
	target := componentFile(file)
	if target == "" {
		return 0
	}
	tree, err := component.Load(target)
	if err != nil || slices.ContainsFunc(tree.Targets, func(c *component.Component) bool { return c.Guessed }) {
		return 0
	}
	cfg, err := config.Find(target, rules.IDs())
	if err != nil {
		return 0
	}
	findings, _ := rules.Check(tree, cfg)
	if len(findings) == 0 {
		return 0
	}
	feedback := report.Feedback(file, findings)
	var output any = contextOutput{specificOutput{"PostToolUse", feedback}}
	if slices.ContainsFunc(findings, func(f rule.Finding) bool { return f.Rule.Level == rule.RedFlag }) {
		output = blockOutput{"block", feedback}
	}
	enc := json.NewEncoder(stdout)
	enc.SetEscapeHTML(false)
	enc.Encode(output)
	return 0
}

// componentFile returns the file declaring the component an edited file
// belongs to: the file itself when Claude Code reads it as a skill, agent,
// command, hooks, settings or manifest file, or the SKILL.md of the skill
// whose directory holds it. It returns "" for any other file, which the
// hook leaves alone.
func componentFile(file string) string {
	slashed := filepath.ToSlash(file)
	dir, base := filepath.Dir(file), filepath.Base(file)
	switch {
	case base == "SKILL.md":
		return file
	case filepath.Ext(base) == ".md" && (strings.Contains(slashed, "/agents/") || strings.Contains(slashed, "/commands/")):
		return file
	case base == "hooks.json" && filepath.Base(dir) == "hooks",
		base == "plugin.json" && filepath.Base(dir) == ".claude-plugin",
		(base == "settings.json" || base == "settings.local.json") && filepath.Base(dir) == ".claude":
		return file
	}
	for d, up := dir, 0; up < 4 && d != filepath.Dir(d); d, up = filepath.Dir(d), up+1 {
		if _, err := os.Stat(filepath.Join(d, "SKILL.md")); err == nil {
			return filepath.Join(d, "SKILL.md")
		}
	}
	return ""
}

// A block decision puts the reason next to the tool's result for Claude;
// additional context only informs it.
type (
	blockOutput struct {
		Decision string `json:"decision"`
		Reason   string `json:"reason"`
	}
	contextOutput struct {
		HookSpecificOutput specificOutput `json:"hookSpecificOutput"`
	}
	specificOutput struct {
		HookEventName     string `json:"hookEventName"`
		AdditionalContext string `json:"additionalContext"`
	}
)
