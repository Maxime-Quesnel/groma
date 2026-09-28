package bench

import (
	"encoding/json"
	"os"
	"slices"
	"strings"
)

// Result is what groma reads from claude plugin eval --json.
type Result struct {
	ClaudeVersion   string  `json:"claudeVersion"`
	CostUSD         float64 `json:"costUsd"`
	DurationSeconds float64 `json:"durationSeconds"`
	Partial         bool    `json:"partial"`
	PartialReason   string  `json:"partialReason"`
	Suite           struct {
		ModelOverride string `json:"modelOverride"`
	} `json:"suite"`
	Cases []struct {
		Name    string `json:"name"`
		Graders []struct {
			Name   string `json:"name"`
			Type   string `json:"type"`
			Config struct {
				Tool string `json:"tool"`
			} `json:"config"`
		} `json:"graders"`
		Arms struct {
			With []struct {
				Error   *string `json:"error"`
				Graders []struct {
					Name   string  `json:"name"`
					Passed bool    `json:"passed"`
					Weight float64 `json:"weight"`
				} `json:"graders"`
			} `json:"with"`
		} `json:"arms"`
	} `json:"cases"`
}

func ReadResult(path string) (Result, error) {
	var r Result
	content, err := os.ReadFile(path)
	if err != nil {
		return r, err
	}
	return r, json.Unmarshal(content, &r)
}

// Run is one run of one case, reduced to what the benchmark scores.
type Run struct {
	Case string
	// Dispatched are the agents Claude dispatched during the run.
	Dispatched []string
	// Work is the weighted share of the case's work graders the run passed,
	// the graders that check what was produced rather than who produced it.
	// HasWork is false when the case has none.
	Work    float64
	HasWork bool
	Failed  bool
	// TimedOut is a run cut short by its time limit, which says nothing
	// about the plugin unless an agent was dispatched before the cut.
	TimedOut bool
}

func (r Result) Runs(s Suite) []Run {
	var runs []Run
	for _, c := range r.Cases {
		routing := map[string]bool{}
		for _, g := range c.Graders {
			routing[g.Name] = g.Type == "tool_used" && g.Config.Tool == "Agent"
		}
		for _, arm := range c.Arms.With {
			run := Run{Case: c.Name, Failed: arm.Error != nil, TimedOut: arm.Error != nil && strings.Contains(*arm.Error, "timed out")}
			var passed, total float64
			for _, g := range arm.Graders {
				if strings.HasPrefix(g.Name, calledPrefix) {
					i := slices.IndexFunc(s.Agents, func(a string) bool { return calledGrader(a) == g.Name })
					if g.Passed && i >= 0 {
						run.Dispatched = append(run.Dispatched, s.Agents[i])
					}
					continue
				}
				if routing[g.Name] {
					continue
				}
				total += g.Weight
				if g.Passed {
					passed += g.Weight
				}
			}
			if total > 0 {
				run.Work, run.HasWork = passed/total, true
			}
			runs = append(runs, run)
		}
	}
	return runs
}
