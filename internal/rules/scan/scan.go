package scan

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/Maxime-Quesnel/groma/internal/rule"
)

type File struct {
	// Path is relative to the scanned root, with forward slashes.
	Path    string
	Content []byte
}

type Rule struct {
	rule.Meta
	// Check returns the evidence found in one file, if any.
	Check func(File) []string
}

var Rules = []Rule{
	hiddenUnicode,
}

// Collect reads every text file under root, skipping .git and binary files.
func Collect(root string) ([]File, error) {
	root, err := filepath.EvalSymlinks(root)
	if err != nil {
		return nil, err
	}
	var files []File
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && d.Name() == ".git" {
			return filepath.SkipDir
		}
		// Symlinks inside root aren't followed, so a link can't pull files
		// such as ~/.ssh into the scan.
		if !d.Type().IsRegular() {
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil || binary(content) {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if rel == "." {
			rel = d.Name()
		}
		files = append(files, File{Path: filepath.ToSlash(rel), Content: content})
		return nil
	})
	return files, err
}

func Check(files []File) []rule.Finding {
	var findings []rule.Finding
	for _, r := range Rules {
		for _, f := range files {
			if evidence := r.Check(f); len(evidence) > 0 {
				findings = append(findings, rule.Finding{Rule: r.Meta, Subject: f.Path, Evidence: evidence})
			}
		}
	}
	return findings
}

func binary(content []byte) bool {
	return bytes.IndexByte(content[:min(len(content), 8000)], 0) >= 0
}
