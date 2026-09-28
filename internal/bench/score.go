package bench

import (
	"maps"
	"math"
	"math/rand/v2"
	"slices"
)

// Score is how precisely Claude routes work to one agent, and how well the
// agent then does it.
type Score struct {
	Agent string
	// Runs of cases that expect the agent, and how many dispatched it.
	Expected, Hit int
	// Runs of every other case, and how many dispatched the agent anyway.
	Other, Wrong int
	// Instead counts, for expected runs that missed the agent, what Claude
	// dispatched in its place; "" stands for no agent at all.
	Instead map[string]int
	// WrongCases counts the wrong dispatches per case.
	WrongCases map[string]int
	// Work sums the work share of the hit runs that have work graders.
	Work     float64
	WorkRuns int
	// Undecided counts runs that timed out before dispatching any agent.
	// They are left out of precision and recall.
	Undecided int
	// scenarios holds the same counts per scenario, for the bootstrap.
	scenarios map[string]*tally
}

type tally struct{ expected, hit, wrong int }

// Precision is the share of the agent's dispatches that were right.
func (s Score) Precision() (float64, bool) { return ratio(s.Hit, s.Hit+s.Wrong) }

// Recall is the share of the runs that needed the agent and got it.
func (s Score) Recall() (float64, bool) { return ratio(s.Hit, s.Expected) }

// WorkScore is the mean share of work graders passed when the agent was
// rightly dispatched, on the cases that grade the work, leaving out runs
// whose work a time limit cut short.
func (s Score) WorkScore() (float64, bool) {
	if s.WorkRuns == 0 {
		return 0, false
	}
	return s.Work / float64(s.WorkRuns), true
}

// Scenarios counts the scenarios that expect the agent.
func (s Score) Scenarios() int {
	n := 0
	for _, t := range s.scenarios {
		if t.expected > 0 {
			n++
		}
	}
	return n
}

// PrecisionInterval and RecallInterval return a 95% interval that is the
// wider of two: a Wilson interval over runs, and a bootstrap that resamples
// whole scenarios. Runs of one scenario share a prompt and a workspace, so
// they are not independent; counting them as such would overstate how much
// the benchmark knows.
func (s Score) PrecisionInterval() (lo, hi float64) {
	wLo, wHi := wilson(s.Hit, s.Hit+s.Wrong)
	bLo, bHi := s.bootstrap(func(t tally) (int, int) { return t.hit, t.hit + t.wrong })
	return math.Min(wLo, bLo), math.Max(wHi, bHi)
}

func (s Score) RecallInterval() (lo, hi float64) {
	wLo, wHi := wilson(s.Hit, s.Expected)
	bLo, bHi := s.bootstrap(func(t tally) (int, int) { return t.hit, t.expected })
	return math.Min(wLo, bLo), math.Max(wHi, bHi)
}

const resamples = 10000

func (s Score) bootstrap(count func(tally) (k, n int)) (lo, hi float64) {
	var clusters []tally
	for _, name := range slices.Sorted(maps.Keys(s.scenarios)) {
		if _, n := count(*s.scenarios[name]); n > 0 {
			clusters = append(clusters, *s.scenarios[name])
		}
	}
	// With fewer than two scenarios there is nothing to resample: an empty
	// interval leaves the Wilson one to stand alone.
	if len(clusters) < 2 {
		return 1, 0
	}
	// A fixed seed keeps a report reproducible from the same result file.
	rng := rand.New(rand.NewPCG(1, uint64(len(clusters))))
	rates := make([]float64, 0, resamples)
	for range resamples {
		var k, n int
		for range clusters {
			ck, cn := count(clusters[rng.IntN(len(clusters))])
			k, n = k+ck, n+cn
		}
		rates = append(rates, float64(k)/float64(n))
	}
	slices.Sort(rates)
	return rates[resamples*25/1000], rates[resamples*975/1000-1]
}

func ratio(k, n int) (float64, bool) {
	if n == 0 {
		return 0, false
	}
	return float64(k) / float64(n), true
}

// wilson returns the 95% Wilson score interval of k successes in n trials,
// which stays inside [0, 1] and holds up for small n and rates near 0 or 1,
// unlike the normal approximation.
func wilson(k, n int) (lo, hi float64) {
	if n == 0 {
		return 0, 1
	}
	const z = 1.959964
	p, fn := float64(k)/float64(n), float64(n)
	d := 1 + z*z/fn
	center := (p + z*z/(2*fn)) / d
	margin := z * math.Sqrt(p*(1-p)/fn+z*z/(4*fn*fn)) / d
	return math.Max(0, center-margin), math.Min(1, center+margin)
}

func ScoreAgents(s Suite, agents []string, runs []Run) []Score {
	cases := map[string]Case{}
	for _, c := range s.Cases {
		cases[c.Name] = c
	}
	var scores []Score
	for _, agent := range agents {
		sc := Score{Agent: agent, Instead: map[string]int{}, WrongCases: map[string]int{}, scenarios: map[string]*tally{}}
		for _, run := range runs {
			c := cases[run.Case]
			t := sc.scenarios[c.Scenario]
			if t == nil {
				t = &tally{}
				sc.scenarios[c.Scenario] = t
			}
			if run.TimedOut && len(run.Dispatched) == 0 {
				sc.Undecided++
				continue
			}
			dispatched := slices.Contains(run.Dispatched, agent)
			if slices.Contains(c.Expect, agent) {
				sc.Expected++
				t.expected++
				if dispatched {
					sc.Hit++
					t.hit++
					if run.HasWork && !run.TimedOut {
						sc.Work += run.Work
						sc.WorkRuns++
					}
					continue
				}
				if len(run.Dispatched) == 0 {
					sc.Instead[""]++
				}
				for _, other := range run.Dispatched {
					sc.Instead[other]++
				}
				continue
			}
			sc.Other++
			if dispatched {
				sc.Wrong++
				t.wrong++
				sc.WrongCases[run.Case]++
			}
		}
		scores = append(scores, sc)
	}
	return scores
}
