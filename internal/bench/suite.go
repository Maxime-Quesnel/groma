// Package bench measures how precisely a Claude Code plugin routes work to
// each of its agents. It runs the plugin's eval suite with claude plugin eval
// and reads, in every run, which agents Claude dispatched.
package bench

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/Maxime-Quesnel/groma/internal/frontmatter"
)

type Suite struct {
	Plugin string
	// Agents are the plugin's agents by the name Claude dispatches them
	// with, "plugin:agent".
	Agents []string
	// Models holds the model each agent's frontmatter sets, if any.
	Models map[string]string
	Cases  []Case
}

type Case struct {
	Name string
	// Scenario groups a case with its rephrasings, named <scenario>--<variant>:
	// their runs are not independent, so intervals resample scenarios.
	Scenario string
	Dir      string
	// Expect and Avoid are the agents the case's own graders require and
	// forbid. They are the ground truth the benchmark scores against.
	Expect []string
	Avoid  []string
}

// Load reads the plugin's agents and the eval cases found in caseDirs.
func Load(pluginDir string, caseDirs ...string) (Suite, error) {
	var manifest struct {
		Name string `json:"name"`
	}
	content, err := os.ReadFile(filepath.Join(pluginDir, ".claude-plugin", "plugin.json"))
	if err != nil {
		return Suite{}, err
	}
	if err := json.Unmarshal(content, &manifest); err != nil {
		return Suite{}, err
	}
	s := Suite{Plugin: manifest.Name, Models: map[string]string{}}
	agents, _ := filepath.Glob(filepath.Join(pluginDir, "agents", "*.md"))
	for _, a := range agents {
		name := s.Plugin + ":" + strings.TrimSuffix(filepath.Base(a), ".md")
		s.Agents = append(s.Agents, name)
		if content, err := os.ReadFile(a); err == nil {
			if model, ok := frontmatter.Scalar(content, "model"); ok {
				s.Models[name] = model
			}
		}
	}
	for _, dir := range caseDirs {
		entries, err := os.ReadDir(dir)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return Suite{}, err
		}
		for _, e := range entries {
			if c, ok := readCase(filepath.Join(dir, e.Name())); ok {
				s.Cases = append(s.Cases, c)
			}
		}
	}
	return s, nil
}

var dispatched = regexp.MustCompile(`"([\w.-]+:[\w.-]+)"`)

func readCase(dir string) (Case, bool) {
	_, prompt := os.Stat(filepath.Join(dir, "prompt.md"))
	_, yaml := os.Stat(filepath.Join(dir, "case.yaml"))
	if prompt != nil && yaml != nil {
		return Case{}, false
	}
	name := filepath.Base(dir)
	scenario, _, _ := strings.Cut(name, "--")
	c := Case{Name: name, Scenario: scenario, Dir: dir}
	graders, _ := filepath.Glob(filepath.Join(dir, "graders", "*.md"))
	for _, g := range graders {
		content, err := os.ReadFile(g)
		if err != nil {
			continue
		}
		kind, _ := frontmatter.Scalar(content, "type")
		tool, _ := frontmatter.Scalar(content, "tool")
		match, _ := frontmatter.Scalar(content, "input_match")
		m := dispatched.FindStringSubmatch(match)
		if kind != "tool_used" || tool != "Agent" || m == nil || !strings.Contains(match, "subagent_type") {
			continue
		}
		if bound(content, "max", -1) == 0 {
			c.Avoid = append(c.Avoid, m[1])
		} else if bound(content, "min", 1) >= 1 && !slices.Contains(c.Expect, m[1]) {
			c.Expect = append(c.Expect, m[1])
		}
	}
	return c, true
}

func bound(content []byte, key string, fallback int) int {
	value, ok := frontmatter.Scalar(content, key)
	if !ok {
		return fallback
	}
	n, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return n
}

// About returns the cases that say something about agent: those that
// expect it or forbid it.
func (s Suite) About(agent string) []Case {
	var cases []Case
	for _, c := range s.Cases {
		if slices.Contains(c.Expect, agent) || slices.Contains(c.Avoid, agent) {
			cases = append(cases, c)
		}
	}
	return cases
}
