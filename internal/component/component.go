// Package component finds the skills, subagents, commands and hooks in a
// file or directory, from where Claude Code looks for them, and reads what
// each one declares.
package component

import (
	"bytes"
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/Maxime-Quesnel/groma/internal/frontmatter"
)

type Kind int

const (
	Skill Kind = iota + 1
	Agent
	Command
	Hooks
	// Plugin is a plugin as a whole, declared by its manifest.
	Plugin
)

var kindNames = [...]string{Skill: "skill", Agent: "agent", Command: "command", Hooks: "hooks", Plugin: "plugin"}

func (k Kind) String() string { return kindNames[k] }

type File struct {
	// Path is relative to the tree's root, with forward slashes.
	Path       string
	Content    []byte
	Executable bool
}

type Component struct {
	Kind Kind
	// Path is the file that declares the component.
	Path    string
	Content []byte
	// Header is the frontmatter of a skill, agent or command.
	Header frontmatter.Header
	// Hooks is what a hooks file, settings file or manifest declares.
	Hooks HookFile
	// Manifest is what a plugin's manifest declares.
	Manifest Manifest
	// Dir is a skill's directory, or the directory of the file.
	Dir string
	// InPlugin is false for a user or project component under .claude.
	InPlugin bool
	// Plugin is the plugin's root, and Project the directory above .claude,
	// or "" when that directory lies outside the tree.
	Plugin, Project string
	// PluginName prefixes the component's name in Claude Code, as in
	// /shop:pdf or shop:reviewer: the manifest's name, or else the plugin
	// directory's.
	PluginName string
	// Guessed reports that the file's location didn't say what it is, so
	// its kind comes from its content.
	Guessed bool

	dirName string
}

// Name is what Claude Code calls the component: the name a skill or agent
// declares, or else its directory or file name; for a command, its path
// under commands/, with : between directories.
func (c *Component) Name() string {
	switch c.Kind {
	case Skill:
		if n := c.Header.Value("name"); n != "" {
			return n
		}
		return c.dirName
	case Agent:
		if n := c.Header.Value("name"); n != "" {
			return n
		}
	case Command:
		parts := strings.Split(strings.TrimSuffix(c.Path, ".md"), "/")
		if i := slices.Index(parts, "commands"); i >= 0 && i < len(parts)-1 {
			parts = parts[i+1:]
		} else {
			parts = parts[len(parts)-1:]
		}
		return strings.Join(parts, ":")
	}
	return strings.TrimSuffix(path.Base(c.Path), path.Ext(c.Path))
}

// Disabled returns the rules a Markdown component turns off for itself, with
// a comment such as "# groma:disable rule-a, rule-b" in its frontmatter,
// which Claude never reads, or <!-- groma:disable rule-a --> in its body.
func (c *Component) Disabled() []string {
	if c.Kind == Hooks || c.Kind == Plugin {
		return nil
	}
	var rules []string
	for _, m := range disableComment.FindAllStringSubmatch(string(c.Content), -1) {
		list := strings.TrimSuffix(strings.TrimSpace(m[1]), "--")
		for _, id := range strings.FieldsFunc(list, func(r rune) bool { return r == ',' || r == ' ' || r == '\t' }) {
			rules = append(rules, id)
		}
	}
	return rules
}

var disableComment = regexp.MustCompile(`(?m)(?:^\s*#|<!--)\s*groma:disable\s+([a-z0-9][a-z0-9, \t-]*)`)

// DirName is the name of the component's directory on disk, even when the
// tree starts inside it.
func (c *Component) DirName() string { return c.dirName }

// ID is how Claude Code tells the component apart within its plugin or
// project: a plugin agent's name is prefixed with the subfolders of agents/
// it sits in, as in review:security.
func (c *Component) ID() string {
	if c.Kind != Agent || !c.InPlugin {
		return c.Name()
	}
	parts := strings.Split(c.Path, "/")
	i := slices.Index(parts, "agents")
	if i < 0 || i >= len(parts)-2 {
		return c.Name()
	}
	return strings.Join(append(slices.Clone(parts[i+1:len(parts)-1]), c.Name()), ":")
}

// Roots maps the directory variables a command can use, without their
// CLAUDE_ prefix, to directories of the tree.
func (c *Component) Roots() map[string]string {
	roots := map[string]string{}
	if c.InPlugin && c.Plugin != "" {
		roots["PLUGIN_ROOT"] = c.Plugin
	}
	if c.Project != "" {
		roots["PROJECT_DIR"] = c.Project
	}
	if c.Kind == Skill {
		roots["SKILL_DIR"] = c.Dir
	}
	return roots
}

type Tree struct {
	// Root is the directory the paths of Files and Components start from.
	Root  string
	Files []File
	// Components are all the components the tree holds, and Targets the
	// ones to check: all of them, or the one a file given to Load declares.
	Components []*Component
	Targets    []*Component
	index      map[string]int
}

func (t *Tree) File(p string) (File, bool) {
	i, ok := t.index[p]
	if !ok {
		return File{}, false
	}
	return t.Files[i], true
}

// Owned returns the files a component is made of: a skill's whole
// directory, or the one file of any other component.
func (t *Tree) Owned(c *Component) []File {
	if c.Kind != Skill {
		f, _ := t.File(c.Path)
		return []File{f}
	}
	var files []File
	for _, f := range t.Files {
		if c.Dir == "." || strings.HasPrefix(f.Path, c.Dir+"/") {
			files = append(files, f)
		}
	}
	return files
}

const maxFileSize = 1 << 20

// Load reads the components at p: a skill, agent, command or hooks file, or a
// directory holding any number of them, such as a plugin or a marketplace.
// Given a file, it also reads the plugin around it, so that scripts and
// links resolve, but returns only the component that file declares.
func Load(p string) (*Tree, error) {
	info, err := os.Stat(p)
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		t, err := read(p, ".")
		if err != nil {
			return nil, err
		}
		t.find()
		if len(t.Components) == 0 {
			return nil, fmt.Errorf("found no skill, agent, command or hook in %s", p)
		}
		t.Targets = t.Components
		return t, nil
	}
	abs, err := filepath.Abs(p)
	if err != nil {
		return nil, err
	}
	root, walk := surroundings(abs)
	t, err := read(root, walk)
	if err != nil {
		return nil, err
	}
	rel, _ := filepath.Rel(root, abs)
	rel = filepath.ToSlash(rel)
	t.find()
	for _, c := range t.Components {
		if c.Path == rel {
			t.Targets = append(t.Targets, c)
		}
	}
	if len(t.Targets) == 0 {
		c, err := t.guess(rel)
		if err != nil {
			return nil, err
		}
		t.Components = append(t.Components, c)
		t.Targets = []*Component{c}
	}
	return t, nil
}

// surroundings returns the directory to read around a file, and the part of
// it to walk: the plugin holding the file, the project holding a .claude
// file, a skill's directory, or else the file's own directory.
func surroundings(file string) (root, walk string) {
	dir := filepath.Dir(file)
	for d := dir; ; d = filepath.Dir(d) {
		if _, err := os.Stat(filepath.Join(d, ".claude-plugin", "plugin.json")); err == nil {
			return d, "."
		}
		if filepath.Base(d) == ".claude" {
			return filepath.Dir(d), ".claude"
		}
		if d == filepath.Dir(d) {
			break
		}
	}
	return dir, "."
}

// read loads the text files under root/walk, skipping .git, node_modules,
// binary files and files over 1 MiB. Symlinks aren't followed, so a link
// can't pull files such as ~/.ssh into the check.
func read(root, walk string) (*Tree, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	t := &Tree{Root: root, index: map[string]int{}}
	err = filepath.WalkDir(filepath.Join(root, walk), func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && (d.Name() == ".git" || d.Name() == "node_modules") {
			return filepath.SkipDir
		}
		if !d.Type().IsRegular() {
			return nil
		}
		info, err := d.Info()
		if err != nil || info.Size() > maxFileSize {
			return err
		}
		content, err := os.ReadFile(p)
		if err != nil || bytes.IndexByte(content[:min(len(content), 8000)], 0) >= 0 {
			return err
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		t.index[filepath.ToSlash(rel)] = len(t.Files)
		t.Files = append(t.Files, File{Path: filepath.ToSlash(rel), Content: content, Executable: info.Mode()&0o111 != 0})
		return nil
	})
	return t, err
}

var kindDirs = []string{"skills", "agents", "commands", "hooks"}

// find lists the components of the tree where Claude Code looks for them.
// Files inside a skill's directory belong to the skill.
func (t *Tree) find() {
	var skillDirs []string
	for _, f := range t.Files {
		if path.Base(f.Path) == "SKILL.md" {
			skillDirs = append(skillDirs, path.Dir(f.Path))
		}
	}
	// A SKILL.md at the root of a plugin owns only the files outside the
	// plugin's component directories.
	inSkill := func(p string) bool {
		return slices.ContainsFunc(skillDirs, func(d string) bool {
			return d == "." && nearestKindDir(p) == "" || strings.HasPrefix(p, d+"/")
		})
	}
	// hookFiles maps each hooks file to its plugin's root, when a manifest
	// names it.
	hookFiles := map[string]string{}
	for _, f := range t.Files {
		dir, base := path.Split(f.Path)
		dir = strings.TrimSuffix(dir, "/")
		switch {
		case base == "SKILL.md":
			t.add(&Component{Kind: Skill, Path: f.Path, Dir: path.Dir(f.Path)})
		case base == "plugin.json" && path.Base(dir) == ".claude-plugin":
			t.add(&Component{Kind: Plugin, Path: f.Path, Dir: path.Dir(dir), Manifest: parseManifest(f.Content)})
			files, inline := manifestHooks(f.Content)
			root := path.Dir(dir)
			for _, file := range files {
				hookFiles[path.Join(root, file)] = root
			}
			if inline != nil {
				t.add(&Component{Kind: Hooks, Path: f.Path, Hooks: parseEvents(inline)})
			}
		case (base == "settings.json" || base == "settings.local.json") && path.Base(dir) == ".claude":
			if events, ok := settingsHooks(f.Content); ok {
				t.add(&Component{Kind: Hooks, Path: f.Path, Hooks: events})
			}
		case inSkill(f.Path):
		case path.Ext(base) == ".md" && nearestKindDir(f.Path) == "agents":
			// A Markdown file without a header is documentation kept beside
			// the agents.
			if h := frontmatter.Parse(f.Content); h.Found || h.Misplaced {
				t.add(&Component{Kind: Agent, Path: f.Path})
			}
		case path.Ext(base) == ".md" && nearestKindDir(f.Path) == "commands":
			t.add(&Component{Kind: Command, Path: f.Path})
		case base == "hooks.json" && path.Base(dir) == "hooks":
			if _, named := hookFiles[f.Path]; !named {
				hookFiles[f.Path] = ""
			}
		}
	}
	for _, p := range slices.Sorted(maps.Keys(hookFiles)) {
		if f, ok := t.File(p); ok {
			c := &Component{Kind: Hooks, Path: p, Hooks: parseHooksFile(f.Content)}
			if root := hookFiles[p]; root != "" {
				c.InPlugin, c.Plugin = true, root
			}
			t.add(c)
		}
	}
	slices.SortStableFunc(t.Components, func(a, b *Component) int {
		return cmp.Or(strings.Compare(a.Path, b.Path), cmp.Compare(a.Kind, b.Kind))
	})
	t.name()
}

// name gives each plugin component its plugin's name.
func (t *Tree) name() {
	names := map[string]string{}
	for _, c := range t.Components {
		if c.Kind == Plugin {
			c.InPlugin, c.Plugin = true, c.Dir
			names[c.Dir] = cmp.Or(c.Manifest.Name, c.dirName)
		}
	}
	for _, c := range t.Components {
		if c.InPlugin && c.Plugin != "" {
			c.PluginName = cmp.Or(names[c.Plugin], filepath.Base(filepath.Join(t.Root, filepath.FromSlash(c.Plugin))))
		}
	}
}

// PluginOf returns the manifest component of c's plugin, if the tree holds it.
func (t *Tree) PluginOf(c *Component) *Component {
	for _, p := range t.Components {
		if p.Kind == Plugin && c.InPlugin && p.Dir == c.Plugin {
			return p
		}
	}
	return nil
}

func (t *Tree) add(c *Component) {
	f, _ := t.File(c.Path)
	c.Content = f.Content
	if c.Kind != Hooks && c.Kind != Plugin {
		c.Header = frontmatter.Parse(f.Content)
	}
	if c.Dir == "" {
		c.Dir = path.Dir(c.Path)
	}
	c.dirName = filepath.Base(filepath.Join(t.Root, filepath.FromSlash(c.Dir)))
	if !c.InPlugin && c.Kind != Plugin {
		t.place(c)
	}
	t.Components = append(t.Components, c)
}

// place works out whether the component ships in a plugin or sits under a
// .claude directory, from its full path, since the tree may start below the
// plugin or project root.
func (t *Tree) place(c *Component) {
	abs := filepath.ToSlash(filepath.Join(t.Root, filepath.FromSlash(c.Path)))
	parts := strings.Split(abs, "/")
	inTree := func(dir []string) string {
		rel, err := filepath.Rel(t.Root, filepath.FromSlash(strings.Join(dir, "/")))
		if err != nil || rel == ".." || strings.HasPrefix(rel, "../") {
			return ""
		}
		return filepath.ToSlash(rel)
	}
	for i := len(parts) - 2; i >= 0; i-- {
		if parts[i] == ".claude" {
			c.Project = inTree(parts[:i])
			return
		}
		if parts[i] == ".claude-plugin" || slices.Contains(kindDirs, parts[i]) {
			if i > 0 && parts[i-1] == ".claude" {
				c.Project = inTree(parts[:i-1])
				return
			}
			c.InPlugin = true
			c.Plugin = inTree(parts[:i])
			return
		}
	}
	// A SKILL.md at a plugin's root.
	if _, err := os.Stat(filepath.Join(filepath.Dir(filepath.FromSlash(abs)), ".claude-plugin", "plugin.json")); err == nil {
		c.InPlugin = true
		c.Plugin = inTree(parts[:len(parts)-1])
	}
}

func nearestKindDir(p string) string {
	parts := strings.Split(p, "/")
	for i := len(parts) - 2; i >= 0; i-- {
		if slices.Contains(kindDirs, parts[i]) {
			return parts[i]
		}
	}
	return ""
}

// guess reads a file found nowhere Claude Code looks, going by its content.
func (t *Tree) guess(rel string) (*Component, error) {
	f, _ := t.File(rel)
	c := &Component{Path: rel, Content: f.Content, Dir: path.Dir(rel), Guessed: true}
	switch path.Ext(rel) {
	case ".json":
		var top map[string]json.RawMessage
		if json.Unmarshal(f.Content, &top) != nil || top["hooks"] == nil {
			return nil, fmt.Errorf("%s holds no hooks", rel)
		}
		c.Kind, c.Hooks = Hooks, parseHooksFile(f.Content)
	case ".md":
		c.Header = frontmatter.Parse(f.Content)
		c.Kind = Command
		if slices.ContainsFunc(c.Header.Fields, func(f frontmatter.Field) bool { return agentOnly[f.Key] }) {
			c.Kind = Agent
		}
	default:
		return nil, errors.New("groma checks skills (SKILL.md), agents and commands (.md) and hooks (.json); " + rel + " is none of them")
	}
	c.dirName = filepath.Base(filepath.Join(t.Root, filepath.FromSlash(c.Dir)))
	t.place(c)
	return c, nil
}

// agentOnly are frontmatter fields that only a subagent has.
var agentOnly = map[string]bool{
	"tools": true, "disallowedTools": true, "permissionMode": true, "maxTurns": true, "skills": true,
	"mcpServers": true, "memory": true, "omitClaudeMd": true, "isolation": true, "color": true, "initialPrompt": true,
}
