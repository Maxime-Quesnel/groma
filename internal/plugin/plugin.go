// Package plugin is the agent-neutral view of what groma scans: the files an
// extension ships and the hooks it declares. Agent adapters fill in the hooks.
package plugin

import (
	"bytes"
	"io/fs"
	"os"
	"path"
	"path/filepath"
)

type Plugin struct {
	Files  []File
	Hooks  []Hook
	Grants []Grant
}

type File struct {
	// Path is relative to the scanned root, with forward slashes.
	Path    string
	Content []byte
}

// A Hook runs a command on an agent event, without asking the user.
type Hook struct {
	// Trigger says what runs the command, as the report shows it:
	// "SessionStart hook", "inline shell".
	Trigger string
	Command string
	// Source is the file that declares the hook.
	Source string
	// Scripts are the files of the scanned tree that the command runs.
	Scripts []string
}

// A Grant lets the agent use a tool without asking the user, such as a
// permission rule like Bash(git status *).
type Grant struct {
	Tool string
	// Source is the file that declares the grant.
	Source string
}

func (p Plugin) File(path string) (File, bool) {
	for _, f := range p.Files {
		if f.Path == path {
			return f, true
		}
	}
	return File{}, false
}

// Read loads every text file under root, skipping .git and binary files.
func Read(root string) (Plugin, error) {
	root, err := filepath.EvalSymlinks(root)
	if err != nil {
		return Plugin{}, err
	}
	var p Plugin
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
		p.Files = append(p.Files, File{Path: filepath.ToSlash(rel), Content: content})
		return nil
	})
	return p, err
}

// ReadAs reads root like Read but names its files under display, so that
// several roots read into one plugin keep distinct paths.
func ReadAs(root, display string) (Plugin, error) {
	p, err := Read(root)
	if err != nil {
		return p, err
	}
	info, err := os.Stat(root)
	if err != nil {
		return p, err
	}
	for i := range p.Files {
		if info.IsDir() {
			p.Files[i].Path = path.Join(display, p.Files[i].Path)
		} else {
			p.Files[i].Path = display
		}
	}
	return p, nil
}

func binary(content []byte) bool {
	return bytes.IndexByte(content[:min(len(content), 8000)], 0) >= 0
}
