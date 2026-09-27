// Package scan holds the checks behind `groma scan`.
package scan

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/Maxime-Quesnel/groma/internal/rule"
)

// Facts is everything the rules may look at. Collect reads it once, so rules
// stay pure functions over file contents.
type Facts struct {
	Files []File
}

type File struct {
	// Path is relative to the scanned root, with forward slashes.
	Path    string
	Content []byte
}

type Rule interface {
	Meta() rule.Meta
	Check(Facts) []rule.Finding
}

func Rules() []Rule {
	return []Rule{
		hiddenUnicode{},
	}
}

// Collect reads every text file under root. It skips .git and binary files,
// and doesn't follow symlinks, so a link can't pull files from outside root
// into the scan.
func Collect(root string) (Facts, error) {
	var facts Facts
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		if !d.Type().IsRegular() {
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if binary(content) {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if rel == "." {
			rel = d.Name()
		}
		facts.Files = append(facts.Files, File{Path: filepath.ToSlash(rel), Content: content})
		return nil
	})
	return facts, err
}

func Check(facts Facts, rules []Rule) []rule.Finding {
	var findings []rule.Finding
	for _, r := range rules {
		findings = append(findings, r.Check(facts)...)
	}
	return findings
}

func binary(content []byte) bool {
	head := content[:min(len(content), 8000)]
	return bytes.IndexByte(head, 0) >= 0
}
