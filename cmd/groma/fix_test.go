package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

func fixable(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	err := os.CopyFS(dir, fstest.MapFS{
		".claude-plugin/plugin.json": {Data: []byte(`{"name": "shop"}`)},
		"agents/reviewer.md":         {Data: []byte("---\nname: reviewer\ndescription: Reviews code. MUST BE USED after a change.\ntools: Read, Grep, AskUserQuestion\n---\n\nYou review code and report each bug.\n")},
		"hooks/hooks.json":           {Data: []byte("{\n  \"hooks\": {\n    \"PostToolUse\": [\n      {\"matcher\": \"Edit|Write|MultiEdit\", \"hooks\": [{\"type\": \"command\", \"command\": \"bash ${CLAUDE_PLUGIN_ROOT}/scripts/format.sh\"}]}\n    ]\n  }\n}\n")},
		"scripts/format.sh":          {Data: []byte("#!/bin/sh\nexit 0\n"), Mode: 0o755},
	})
	if err != nil {
		t.Fatal(err)
	}
	return dir
}

func read(t *testing.T, dir, file string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(dir, file))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestFixShowsTheDiffAndWritesOnYes(t *testing.T) {
	dir := fixable(t)
	var stdout, stderr strings.Builder

	code := run([]string{"fix", dir}, strings.NewReader("y\n"), &stdout, &stderr)

	out := stdout.String()
	for _, want := range []string{
		"-tools: Read, Grep, AskUserQuestion",
		"+tools: Read, Grep",
		`-      {"matcher": "Edit|Write|MultiEdit", "hooks": [{"type": "command", "command": "bash ${CLAUDE_PLUGIN_ROOT}/scripts/format.sh"}]}`,
		`+      {"matcher": "Edit|Write", "hooks": [{"type": "command", "command": "bash \"${CLAUDE_PLUGIN_ROOT}/scripts/format.sh\""}]}`,
		"3 fixes in 2 files: hook-matcher-dead-alternative 1, tool-unavailable-to-agents 1, unquoted-path 1",
		"1 more fix changes what runs",
		"✔ Fixed 3 problems in 2 files.",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output lacks %q:\n%s%s", want, out, stderr.String())
		}
	}
	if code != 0 || !strings.Contains(read(t, dir, "agents/reviewer.md"), "tools: Read, Grep\n") ||
		!strings.Contains(read(t, dir, "hooks/hooks.json"), `"matcher": "Edit|Write"`) {
		t.Errorf("exit %d, files not fixed", code)
	}
	if strings.Contains(read(t, dir, "agents/reviewer.md"), "Use proactively") {
		t.Error("an unsafe fix was applied without --unsafe")
	}
}

func TestFixUnsafeRewritesTheDescription(t *testing.T) {
	dir := fixable(t)
	var stdout, stderr strings.Builder

	run([]string{"fix", "--unsafe", dir}, strings.NewReader("yes\n"), &stdout, &stderr)

	if got := read(t, dir, "agents/reviewer.md"); !strings.Contains(got, "description: Reviews code. Use proactively after a change.") {
		t.Errorf("got:\n%s\n%s", got, stdout.String())
	}
}

func TestFixWritesNothingWithoutConsent(t *testing.T) {
	dir := fixable(t)
	before := read(t, dir, "agents/reviewer.md")

	var stdout, stderr strings.Builder
	run([]string{"fix", dir}, strings.NewReader("\n"), &stdout, &stderr)
	if read(t, dir, "agents/reviewer.md") != before || !strings.Contains(stdout.String(), "Nothing written.") {
		t.Errorf("wrote without a yes:\n%s", stdout.String())
	}

	pipe, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	w.WriteString("y\n")
	w.Close()
	stdout.Reset()
	run([]string{"fix", dir}, pipe, &stdout, &stderr)
	if read(t, dir, "agents/reviewer.md") != before || !strings.Contains(stdout.String(), "run it in a terminal") {
		t.Errorf("wrote from a pipe:\n%s", stdout.String())
	}
}

func TestFixRespectsTheConfiguration(t *testing.T) {
	dir := fixable(t)
	os.WriteFile(filepath.Join(dir, ".groma.yml"), []byte("disable: [tool-unavailable-to-agents, hook-matcher-dead-alternative, unquoted-path]\n"), 0o644)
	var stdout, stderr strings.Builder

	run([]string{"fix", dir}, strings.NewReader("y\n"), &stdout, &stderr)

	if !strings.Contains(stdout.String(), "Nothing groma can fix without changing what runs.") {
		t.Errorf("got:\n%s", stdout.String())
	}
}
