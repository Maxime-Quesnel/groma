package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

func TestCheckReportsRedFlagsAndExitsOne(t *testing.T) {
	dir := t.TempDir()
	err := os.CopyFS(dir, fstest.MapFS{
		".claude-plugin/plugin.json": {Data: []byte(`{"name": "shop"}`)},
		"skills/pdf/SKILL.md":        {Data: []byte("---\nname: pdf\ndescription: Extracts text from PDFs. Use when the user mentions a PDF.\n---\n\nRead the PDF.\n")},
		"agents/rails.md":            {Data: []byte("---\nname: rails\ndescription: Reviews Rails code. Use after a Rails change.\ntools: Read, Grep\npermissionMode: plan\n---\n\nYou review Rails code and report each bug with its file and line.\n")},
	})
	if err != nil {
		t.Fatal(err)
	}
	var stdout, stderr strings.Builder

	code := run([]string{"check", dir}, nil, &stdout, &stderr)

	out := stdout.String()
	for _, want := range []string{
		"1 plugin, 1 skill, 1 agent",
		"✔ skill   skills/pdf/SKILL.md",
		"✖ agent   agents/rails.md",
		"Red flags",
		"line 5: permissionMode is ignored on an agent that ships in a plugin",
		"✖ 1 red flag in 3 components",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output lacks %q:\n%s%s", want, out, stderr.String())
		}
	}
	if code != 1 {
		t.Errorf("exit status %d, want 1", code)
	}
}

func TestCheckExitsZeroOnWarningsOnly(t *testing.T) {
	skill := filepath.Join(t.TempDir(), "notes")
	os.MkdirAll(skill, 0o755)
	os.WriteFile(filepath.Join(skill, "SKILL.md"), []byte("---\nname: notes\ndescription: Keeps notes.\n---\n\nKeep notes.\n"), 0o644)
	var stdout, stderr strings.Builder

	code := run([]string{"check", filepath.Join(skill, "SKILL.md")}, nil, &stdout, &stderr)

	if code != 0 || !strings.Contains(stdout.String(), "description-no-trigger") {
		t.Errorf("exit %d:\n%s%s", code, stdout.String(), stderr.String())
	}
}

func TestCheckRejectsWhatItCantRead(t *testing.T) {
	for _, args := range [][]string{{"check"}, {"check", "a", "b"}, {"check", filepath.Join(t.TempDir(), "missing")}, {"scan"}} {
		var stdout, stderr strings.Builder
		if code := run(args, nil, &stdout, &stderr); code != 2 || stderr.Len() == 0 {
			t.Errorf("%q: exit %d, stderr %q", args, code, stderr.String())
		}
	}
}

func TestCheckSilencesWhatTheConfigurationTurnsOff(t *testing.T) {
	skill := filepath.Join(t.TempDir(), "notes")
	os.MkdirAll(skill, 0o755)
	os.WriteFile(filepath.Join(skill, "SKILL.md"), []byte("---\nname: notes\ndescription: Keeps notes.\n---\n\nKeep notes.\n"), 0o644)
	os.WriteFile(filepath.Join(skill, ".groma.yml"), []byte("disable:\n  - description-no-trigger\n"), 0o644)
	var stdout, stderr strings.Builder

	run([]string{"check", skill}, nil, &stdout, &stderr)

	if out := stdout.String(); strings.Contains(out, "description-no-trigger") || !strings.Contains(out, "1 silenced by .groma.yml or groma:disable") {
		t.Errorf("got:\n%s%s", out, stderr.String())
	}
}
