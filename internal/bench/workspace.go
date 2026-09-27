package bench

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Prepare copies the plugin into dir, without its own eval suite, and puts
// cases in its place, each with one grader per agent that passes when the
// agent was dispatched. Those graders carry a negligible weight, so they
// leave the case score alone; the benchmark reads their verdicts only.
// The installed plugin is never written to.
func Prepare(s Suite, pluginDir, dir string, cases []Case) (string, error) {
	copyDir := filepath.Join(dir, s.Plugin)
	err := copyTree(pluginDir, copyDir, func(rel string) bool {
		return rel == "evals" || strings.HasPrefix(rel, "evals"+string(filepath.Separator))
	})
	if err != nil {
		return "", err
	}
	for _, c := range cases {
		caseDir := filepath.Join(copyDir, "evals", c.Name)
		if err := copyTree(c.Dir, caseDir, nil); err != nil {
			return "", err
		}
		for _, agent := range s.Agents {
			grader := fmt.Sprintf("---\ntype: tool_used\ntool: Agent\ninput_match: '\"subagent_type\"\\s*:\\s*\"%s\"'\nmin: 1\nweight: 0.001\n---\n", agent)
			if err := os.WriteFile(filepath.Join(caseDir, "graders", calledGrader(agent)+".md"), []byte(grader), 0o644); err != nil {
				return "", err
			}
		}
	}
	return copyDir, nil
}

const calledPrefix = "groma-called-"

func calledGrader(agent string) string {
	return calledPrefix + strings.ReplaceAll(agent, ":", "--")
}

func copyTree(src, dst string, skip func(rel string) bool) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		if skip != nil && skip(rel) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		target := filepath.Join(dst, rel)
		info, err := d.Info()
		if err != nil {
			return err
		}
		switch {
		case d.IsDir():
			return os.MkdirAll(target, 0o755)
		case d.Type()&fs.ModeSymlink != 0:
			link, err := os.Readlink(path)
			if err != nil {
				return err
			}
			return os.Symlink(link, target)
		default:
			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			return os.WriteFile(target, content, info.Mode().Perm())
		}
	})
}
