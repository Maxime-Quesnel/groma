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
	// Checks and Judge weigh the case's work graders, those that grade what
	// was produced rather than who produced it: exact checks such as regexes
	// and tool calls, and the verdicts of a judge model, which can refuse a
	// right answer phrased another way.
	Checks, Judge graded
	Failed        bool
	// TimedOut is a run cut short by its time limit, which says nothing
	// about the plugin unless an agent was dispatched before the cut.
	TimedOut bool
}

func (r Result) Runs(s Suite) []Run {
	var runs []Run
	for _, c := range r.Cases {
		routing, kinds := map[string]bool{}, map[string]string{}
		for _, g := range c.Graders {
			routing[g.Name] = g.Type == "tool_used" && g.Config.Tool == "Agent"
			kinds[g.Name] = g.Type
		}
		for _, arm := range c.Arms.With {
			run := Run{Case: c.Name, Failed: arm.Error != nil, TimedOut: arm.Error != nil && strings.Contains(*arm.Error, "timed out")}
			for _, g := range arm.Graders {
				if strings.HasPrefix(g.Name, calledPrefix) {
					i := slices.IndexFunc(s.Agents, func(a string) bool { return calledGrader(a) == g.Name })
					if g.Passed && i >= 0 {
						run.Dispatched = append(run.Dispatched, s.Agents[i])
					}
					continue
				}
				switch {
				case routing[g.Name]:
				case kinds[g.Name] == "llm" || kinds[g.Name] == "baseline":
					run.Judge.add(g.Weight, g.Passed)
				default:
					run.Checks.add(g.Weight, g.Passed)
				}
			}
			runs = append(runs, run)
		}
	}
	return runs
}

// graded is the weight of a run's work graders, and how much of it passed.
type graded struct{ passed, total float64 }

func (g *graded) add(weight float64, passed bool) {
	g.total += weight
	if passed {
		g.passed += weight
	}
}

func (g graded) share() (float64, bool) {
	if g.total == 0 {
		return 0, false
	}
	return g.passed / g.total, true
}
