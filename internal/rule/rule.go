package rule

import (
	"cmp"
	"fmt"
)

type Severity int

const (
	Low Severity = iota + 1
	Medium
	High
	Critical
)

var severityNames = [...]string{Low: "low", Medium: "medium", High: "high", Critical: "critical"}

func (s Severity) String() string {
	if s >= Low && s <= Critical {
		return severityNames[s]
	}
	return fmt.Sprintf("Severity(%d)", int(s))
}

type Meta struct {
	ID             string
	Severity       Severity
	Title          string
	Description    string
	Remediation    string
	FalsePositives []string
	References     []string
}

type Finding struct {
	Rule    Meta
	Subject string
	// Evidence must never hold a secret in clear.
	Evidence    []string
	Remediation string
}

func (f Finding) Fix() string {
	return cmp.Or(f.Remediation, f.Rule.Remediation)
}
