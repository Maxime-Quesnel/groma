// Package rule describes groma's checks and what they find.
package rule

import "fmt"

type Level int

const (
	// Warning: the component works, but isn't written the way Claude Code's
	// documentation recommends.
	Warning Level = iota + 1
	// RedFlag: a security risk, or something Claude Code won't load, won't
	// run, or silently ignores.
	RedFlag
)

func (l Level) String() string {
	switch l {
	case Warning:
		return "warning"
	case RedFlag:
		return "red flag"
	}
	return fmt.Sprintf("Level(%d)", int(l))
}

type Meta struct {
	// ID names the risk in kebab-case: hook-unknown-event.
	ID             string
	Level          Level
	Title          string
	Description    string
	Remediation    string
	FalsePositives []string
	References     []string
}

type Finding struct {
	Rule Meta
	// Path is the file that declares the component.
	Path string
	// Evidence must never hold a secret in clear.
	Evidence []string
}
