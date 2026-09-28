// Package config reads .groma.yml, which turns rules off, everywhere or for
// some paths, the way .rubocop.yml does for RuboCop:
//
//	disable:
//	  - description-emphatic
//	exclude:
//	  - plugins/legacy/**
//	reference-no-toc:
//	  - plugins/botyglot-dev/**
//
// disable turns rules off everywhere, exclude skips paths for every rule,
// and a rule's ID lists the paths that rule skips. Paths are globs relative
// to the file's directory, where ** matches any number of directories.
package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/Maxime-Quesnel/groma/internal/frontmatter"
)

const FileName = ".groma.yml"

type Config struct {
	// Path is the file the configuration comes from, or "" for none.
	Path     string
	disabled []string
	exclude  []*regexp.Regexp
	perRule  map[string][]*regexp.Regexp
}

// Find reads the .groma.yml nearest to target: in its directory or the
// closest one above. rules are the IDs a configuration may name.
func Find(target string, rules []string) (Config, error) {
	abs, err := filepath.Abs(target)
	if err != nil {
		return Config{}, err
	}
	if info, err := os.Stat(abs); err == nil && !info.IsDir() {
		abs = filepath.Dir(abs)
	}
	for dir := abs; ; dir = filepath.Dir(dir) {
		content, err := os.ReadFile(filepath.Join(dir, FileName))
		switch {
		case err == nil:
			return parse(filepath.Join(dir, FileName), content, rules)
		case !errors.Is(err, fs.ErrNotExist):
			return Config{}, err
		}
		if dir == filepath.Dir(dir) {
			return Config{}, nil
		}
	}
}

func parse(path string, content []byte, rules []string) (Config, error) {
	c := Config{Path: path, perRule: map[string][]*regexp.Regexp{}}
	h := frontmatter.ParseDocument(content)
	for _, p := range h.Problems {
		return c, fmt.Errorf("%s, line %d: %s", path, p.Line, p.Text)
	}
	known := func(id string, line int) error {
		if slices.Contains(rules, id) {
			return nil
		}
		return fmt.Errorf("%s, line %d: groma has no rule %s", path, line, id)
	}
	for _, f := range h.Fields {
		entries := f.Items()
		switch f.Key {
		case "disable":
			for _, id := range entries {
				if err := known(id, f.Line); err != nil {
					return c, err
				}
				c.disabled = append(c.disabled, id)
			}
		case "exclude":
			c.exclude = append(c.exclude, globs(entries)...)
		default:
			if err := known(f.Key, f.Line); err != nil {
				return c, err
			}
			c.perRule[f.Key] = append(c.perRule[f.Key], globs(entries)...)
		}
	}
	return c, nil
}

// Allows reports whether rule applies to the file at path.
func (c Config) Allows(rule, path string) bool {
	if slices.Contains(c.disabled, rule) {
		return false
	}
	if c.Path == "" {
		return true
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return true
	}
	rel, err := filepath.Rel(filepath.Dir(c.Path), abs)
	if err != nil {
		return true
	}
	rel = filepath.ToSlash(rel)
	matches := func(g *regexp.Regexp) bool { return g.MatchString(rel) }
	return !slices.ContainsFunc(c.exclude, matches) && !slices.ContainsFunc(c.perRule[rule], matches)
}

// globs compiles path globs: ** matches any number of directories, * and ?
// stay within one, and a directory matches everything under it.
func globs(patterns []string) []*regexp.Regexp {
	var compiled []*regexp.Regexp
	for _, p := range patterns {
		p = strings.TrimPrefix(strings.TrimSuffix(p, "/"), "./")
		var b strings.Builder
		for i := 0; i < len(p); i++ {
			switch {
			case strings.HasPrefix(p[i:], "**/"):
				b.WriteString("(?:.*/)?")
				i += 2
			case strings.HasPrefix(p[i:], "**"):
				b.WriteString(".*")
				i++
			case p[i] == '*':
				b.WriteString("[^/]*")
			case p[i] == '?':
				b.WriteString("[^/]")
			default:
				b.WriteString(regexp.QuoteMeta(p[i : i+1]))
			}
		}
		compiled = append(compiled, regexp.MustCompile("^"+b.String()+"(?:/.*)?$"))
	}
	return compiled
}
