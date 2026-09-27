// Package rule defines what every groma check declares and what it reports.
package rule

import "fmt"

type Severity int

const (
	Low Severity = iota + 1
	Medium
	High
	Critical
)

func (s Severity) String() string {
	switch s {
	case Low:
		return "low"
	case Medium:
		return "medium"
	case High:
		return "high"
	case Critical:
		return "critical"
	}
	return fmt.Sprintf("Severity(%d)", int(s))
}

type Meta struct {
	ID          string
	Severity    Severity
	Title       string
	Description string
	Remediation string
	// Fixable reports whether `groma fix` can apply the remediation.
	Fixable        bool
	FalsePositives []string
	References     []string
}

type Finding struct {
	Rule Meta
	// Subject is what the finding is about, such as an agent's name.
	Subject string
	// Evidence is what was observed, one item per line of the report.
	// It must never hold a secret in clear.
	Evidence []string
	// Remediation replaces Rule.Remediation with steps specific to the
	// subject, when the subject has its own.
	Remediation string
}

func (f Finding) Fix() string {
	if f.Remediation != "" {
		return f.Remediation
	}
	return f.Rule.Remediation
}
