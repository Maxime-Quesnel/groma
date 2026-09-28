package fix

import (
	"reflect"
	"strings"
	"testing"
)

func TestPlanAppliesEditsAndDropsOverlaps(t *testing.T) {
	files := map[string]string{
		"hooks/hooks.json": `{"matcher": "Edit|Write|MultiEdit"}`,
		"agents/a.md":      "tools: Read, AskUserQuestion\n",
		"scripts/x.sh":     "echo x\n",
	}
	read := func(p string) ([]byte, bool) { s, ok := files[p]; return []byte(s), ok }

	got, dropped := Plan([]Edit{
		{Path: "hooks/hooks.json", Start: 13, End: 33, New: "Edit|Write", Rule: "dead"},
		{Path: "hooks/hooks.json", Start: 13, End: 33, New: "Edit|Write", Rule: "dead"},
		{Path: "hooks/hooks.json", Start: 20, End: 25, New: "overlap", Rule: "other"},
		{Path: "agents/a.md", Start: 11, End: 28, New: "", Rule: "unavailable"},
		{Path: "scripts/x.sh", Executable: true, Rule: "chmod"},
		{Path: "missing.md", Start: 0, End: 0, New: "x", Rule: "gone"},
	}, read)

	want := []File{
		{Path: "agents/a.md", Old: []byte(files["agents/a.md"]), New: []byte("tools: Read\n"), Rules: map[string]int{"unavailable": 1}},
		{Path: "hooks/hooks.json", Old: []byte(files["hooks/hooks.json"]), New: []byte(`{"matcher": "Edit|Write"}`), Rules: map[string]int{"dead": 1}},
		{Path: "scripts/x.sh", Old: []byte("echo x\n"), New: []byte("echo x\n"), MakeExecutable: true, Rules: map[string]int{"chmod": 1}},
	}
	if !reflect.DeepEqual(got, want) || dropped != 2 {
		t.Errorf("got %+v, dropped %d", got, dropped)
	}
}

func TestDiffShowsHunksWithContext(t *testing.T) {
	old := "a\nb\nc\nd\ne\nf\ng\nh\ni\nj\nk\nl\nm\n"
	new := "a\nB\nc\nd\ne\nf\ng\nh\ni\nj\nk\nL\nm\nn\n"

	got := strings.Join(Diff("x.md", []byte(old), []byte(new)), "\n")

	want := `--- a/x.md
+++ b/x.md
@@ -1,5 +1,5 @@
 a
-b
+B
 c
 d
 e
@@ -9,5 +9,6 @@
 i
 j
 k
-l
+L
 m
+n`
	if got != want {
		t.Errorf("got\n%s\nwant\n%s", got, want)
	}
}

func TestDiffOfIdenticalContentIsEmpty(t *testing.T) {
	if got := Diff("x", []byte("a\n"), []byte("a\n")); got != nil {
		t.Errorf("got %q", got)
	}
}
