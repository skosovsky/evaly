package evaly

import (
	"fmt"
	"math"
	"math/rand"
	"sort"
	"strings"
)

type GateVerdict string

const (
	GatePass         GateVerdict = "pass"
	GateFail         GateVerdict = "fail"
	GateInconclusive GateVerdict = "inconclusive"
	GateInvalid      GateVerdict = "invalid_comparison"
)

type GatePolicy struct {
	Revision          string  `json:"revision"`
	MinimumCoverage   float64 `json:"minimum_coverage"`
	MinimumQuality    float64 `json:"minimum_quality"`
	MaximumRegression float64 `json:"maximum_regression"`
	BootstrapSamples  int     `json:"bootstrap_samples"`
	Seed              int64   `json:"seed"`
}
type Aggregate struct {
	Eligible           int      `json:"eligible"`
	Scored             int      `json:"scored"`
	Coverage           float64  `json:"coverage"`
	MeanPassRate       float64  `json:"mean_pass_rate"`
	SuccessAtLeastOnce int      `json:"success_at_least_once"`
	AllRepeatsSuccess  int      `json:"all_repeats_success"`
	SuccessDenominator int      `json:"success_denominator"`
	SetupFailures      int      `json:"setup_failures"`
	TargetFailures     int      `json:"target_failures"`
	GraderFailures     int      `json:"grader_failures"`
	CleanupFailures    int      `json:"cleanup_failures"`
	Excluded           []string `json:"excluded"`
}
type Interval struct {
	Method       string `json:"method"`
	Unit         string `json:"unit"`
	Cases        int    `json:"cases"`
	Lower, Upper float64
	Confidence   float64 `json:"confidence"`
	Limitation   string  `json:"limitation,omitempty"`
}
type CaseDifference struct {
	CaseID                          string `json:"case_id"`
	Baseline, Candidate, Difference float64
}
type TrialSummary struct {
	ExperimentID, TrialID, CaseID, CaseRevision, Target, Fixture, Reset, EvidenceRevision string
	Repeat, Attempt                                                                       int
	Seed                                                                                  int64
	Status                                                                                TrialStatus
	Cleanup                                                                               CleanupStatus
}
type Comparison struct {
	Trials             []TrialSummary   `json:"trials"`
	Version            int              `json:"version"`
	Revision           string           `json:"revision"`
	Baseline           string           `json:"baseline"`
	Candidate          string           `json:"candidate"`
	Verdict            GateVerdict      `json:"verdict"`
	Reasons            []string         `json:"reasons"`
	Policy             GatePolicy       `json:"policy"`
	BaselineAggregate  Aggregate        `json:"baseline_aggregate"`
	CandidateAggregate Aggregate        `json:"candidate_aggregate"`
	Pairs              []CaseDifference `json:"pairs"`
	Uncertainty        *Interval        `json:"uncertainty,omitempty"`
	Metric             string           `json:"metric"`
	Unit               string           `json:"unit"`
}

func aggregate(r ExperimentRecord) (Aggregate, map[string]float64) {
	a := Aggregate{Eligible: len(r.Manifest.Cases), SuccessDenominator: len(r.Manifest.Cases), Excluded: []string{}}
	values := map[string]float64{}
	final := map[string]TrialRecord{}
	for _, t := range r.Trials {
		if t.Cleanup.State == "failed" {
			a.CleanupFailures++
		}
		if t.Status == SetupError {
			a.SetupFailures++
		}
		if t.Status == TargetError {
			a.TargetFailures++
		}
		for _, g := range t.Grades {
			if g.Status == GraderError {
				a.GraderFailures++
			}
		}
		key := t.CaseID + "/" + identityPart(t.Repeat)
		if old, ok := final[key]; !ok || old.Attempt < t.Attempt {
			final[key] = t
		}
	}
	for _, cs := range r.Manifest.Cases {
		complete := true
		passes := 0
		for repeat := range r.Manifest.Plan.Repeats {
			t, ok := final[cs.ID+"/"+identityPart(repeat)]
			if !ok || t.Status != Completed || t.UsageError != "" || len(t.Grades) != len(r.Manifest.Graders) {
				complete = false
				continue
			}
			pass, scored := AssertionOutcome(t.Grades, r.Manifest.Plan.AssertionPolicy)
			if !scored {
				complete = false
			}
			if pass {
				passes++
			}
		}
		if passes > 0 {
			a.SuccessAtLeastOnce++
		}
		if complete && passes == r.Manifest.Plan.Repeats {
			a.AllRepeatsSuccess++
		}
		if !complete {
			a.Excluded = append(a.Excluded, cs.ID)
			continue
		}
		v := float64(passes) / float64(r.Manifest.Plan.Repeats)
		values[cs.ID] = v
		a.Scored++
		a.MeanPassRate += v
	}
	if a.Eligible > 0 {
		a.Coverage = float64(a.Scored) / float64(a.Eligible)
	}
	if a.Scored > 0 {
		a.MeanPassRate /= float64(a.Scored)
	}
	return a, values
}
func compatible(a, b ExperimentManifest) bool {
	if a.State != "sealed" || b.State != "sealed" || a.Mode != "controlled" || b.Mode != "controlled" ||
		a.Dataset != b.Dataset ||
		a.Selection != b.Selection ||
		a.Lifecycle != b.Lifecycle ||
		a.Lifecycle.Isolation != Isolated ||
		a.Projection != b.Projection ||
		a.CapturePolicy != b.CapturePolicy ||
		a.Plan != b.Plan ||
		a.OutputCodec != b.OutputCodec ||
		a.CaptureMaxEvents != b.CaptureMaxEvents ||
		a.CaptureMaxBytes != b.CaptureMaxBytes ||
		a.CriticalEvidence != b.CriticalEvidence {
		return false
	}
	reqA, _ := canonical(a.CaptureRequirements)
	reqB, _ := canonical(b.CaptureRequirements)
	if string(reqA) != string(reqB) {
		return false
	}
	x, _ := canonical(a.Cases)
	y, _ := canonical(b.Cases)
	if string(x) != string(y) {
		return false
	}
	x, _ = canonical(a.Graders)
	y, _ = canonical(b.Graders)
	if string(x) != string(y) {
		return false
	}
	x, _ = canonical(a.CaptureKinds)
	y, _ = canonical(b.CaptureKinds)
	if string(x) != string(y) {
		return false
	}
	if a.PairID != b.PairID {
		return false
	}
	if a.PairID != "" {
		x, _ = canonical(a.PairOrder)
		y, _ = canonical(b.PairOrder)
		if string(x) != string(y) {
			return false
		}
	}
	return true
}
func Compare(baseline, candidate Experiment, p GatePolicy) (Comparison, error) {
	c := Comparison{
		Version:   1,
		Baseline:  baseline.Revision(),
		Candidate: candidate.Revision(),
		Policy:    p,
		Metric:    "assertion_pass_rate",
		Unit:      "case",
		Reasons:   []string{},
		Pairs:     []CaseDifference{},
	}
	if p.Revision == "" || !finiteNonnegative(p.MinimumCoverage) || p.MinimumCoverage > 1 ||
		!finiteNonnegative(p.MinimumQuality) ||
		p.MinimumQuality > 1 ||
		!finiteNonnegative(p.MaximumRegression) ||
		p.MaximumRegression > 1 ||
		p.BootstrapSamples < 100 ||
		p.BootstrapSamples > 100000 {
		return c, ErrInvalid
	}
	if _, e := RestoreExperiment(baseline.Record()); e != nil {
		return c, e
	}
	if _, e := RestoreExperiment(candidate.Record()); e != nil {
		return c, e
	}
	b, cv := baseline.Record(), candidate.Record()
	for _, record := range []ExperimentRecord{b, cv} {
		for _, t := range record.Trials {
			c.Trials = append(
				c.Trials,
				TrialSummary{
					ExperimentID:     record.Manifest.ID,
					TrialID:          t.ID,
					CaseID:           t.CaseID,
					CaseRevision:     t.CaseRevision,
					Target:           record.Manifest.Provenance.Target,
					Fixture:          record.Manifest.Lifecycle.Fixture,
					Reset:            record.Manifest.Lifecycle.Reset,
					EvidenceRevision: t.Evidence.Revision,
					Repeat:           t.Repeat,
					Attempt:          t.Attempt,
					Seed:             t.Seed,
					Status:           t.Status,
					Cleanup:          t.Cleanup,
				},
			)
		}
	}
	ba, bvalues := aggregate(b)
	ca, cvalues := aggregate(cv)
	c.BaselineAggregate = ba
	c.CandidateAggregate = ca
	if !compatible(b.Manifest, cv.Manifest) {
		c.Verdict = GateInvalid
		c.Reasons = append(c.Reasons, "incompatible_manifests")
	} else if ba.Coverage < p.MinimumCoverage || ca.Coverage < p.MinimumCoverage || ba.Scored == 0 || ca.Scored == 0 {
		c.Verdict = GateInconclusive
		c.Reasons = append(c.Reasons, "insufficient_coverage")
	} else {
		c.Verdict = GatePass
	}
	differences := []float64{}
	for _, cs := range b.Manifest.Cases {
		bv, bok := bvalues[cs.ID]
		v, cok := cvalues[cs.ID]
		if bok && cok {
			d := v - bv
			c.Pairs = append(c.Pairs, CaseDifference{cs.ID, bv, v, d})
			differences = append(differences, d)
		}
	}
	if len(differences) > 0 {
		c.Uncertainty = bootstrap(differences, p.BootstrapSamples, p.Seed)
	}
	if c.Verdict == GatePass {
		if ca.MeanPassRate < p.MinimumQuality {
			c.Verdict = GateFail
			c.Reasons = append(c.Reasons, "minimum_quality")
		}
		if len(differences) == 0 {
			c.Verdict = GateInconclusive
			c.Reasons = append(c.Reasons, "no_matched_cases")
		} else {
			delta := 0.0
			for _, d := range differences {
				delta += d
			}
			delta /= float64(len(differences))
			if delta < -p.MaximumRegression {
				c.Verdict = GateFail
				c.Reasons = append(c.Reasons, "regression")
			}
		}
	}
	bbytes, e := canonical(c)
	if e != nil {
		return c, e
	}
	c.Revision = digest(bbytes)
	return c, nil
}
func bootstrap(values []float64, samples int, seed int64) *Interval {
	r := &Interval{
		Method:     "paired_case_bootstrap_percentile",
		Unit:       "case",
		Cases:      len(values),
		Confidence: .95,
		Limitation: "independent representative cases assumed; no multiple-comparison correction",
	}
	if len(values) == 1 {
		r.Lower = values[0]
		r.Upper = values[0]
		r.Limitation = "one independent case: interval is degenerate and not inferential"
		return r
	}
	rng := rand.New(rand.NewSource(seed))
	means := make([]float64, samples)
	for i := range means {
		for range values {
			means[i] += values[rng.Intn(len(values))]
		}
		means[i] /= float64(len(values))
	}
	sort.Float64s(means)
	r.Lower = means[int(math.Floor(.025*float64(samples-1)))]
	r.Upper = means[int(math.Ceil(.975*float64(samples-1)))]
	return r
}

// Report renders only permitted artifact fields; raw outputs are never stored.
func Report(c Comparison) string {
	var b strings.Builder
	fmt.Fprintf(
		&b,
		"evaly %s\nbaseline: %s\ncandidate: %s\nmetric: %s; independent unit: %s\n",
		c.Verdict,
		c.Baseline,
		c.Candidate,
		c.Metric,
		c.Unit,
	)
	for _, row := range []struct {
		name string
		a    Aggregate
	}{{"baseline", c.BaselineAggregate}, {"candidate", c.CandidateAggregate}} {
		fmt.Fprintf(
			&b,
			"%s: scored %d/%d; coverage %.3f; observed mean pass %.3f; setup=%d target=%d grader=%d cleanup=%d\n",
			row.name,
			row.a.Scored,
			row.a.Eligible,
			row.a.Coverage,
			row.a.MeanPassRate,
			row.a.SetupFailures,
			row.a.TargetFailures,
			row.a.GraderFailures,
			row.a.CleanupFailures,
		)
	}
	for _, t := range c.Trials {
		fmt.Fprintf(
			&b,
			"trial %s: %s; evidence %s; cleanup %s; replay case=%s target=%s fixture=%s reset=%s seed=%d repeat=%d attempt=%d in a fresh environment\n",
			t.TrialID,
			t.Status,
			t.EvidenceRevision,
			t.Cleanup.State,
			t.CaseID,
			t.Target,
			t.Fixture,
			t.Reset,
			t.Seed,
			t.Repeat,
			t.Attempt,
		)
	}
	for _, p := range c.Pairs {
		fmt.Fprintf(&b, "case %s: %.3f -> %.3f (delta %.3f)\n", p.CaseID, p.Baseline, p.Candidate, p.Difference)
	}
	for _, reason := range c.Reasons {
		fmt.Fprintf(&b, "reason: %s\n", reason)
	}
	if c.Uncertainty != nil {
		fmt.Fprintf(
			&b,
			"%s: [%.3f, %.3f]; n=%d cases; %s\n",
			c.Uncertainty.Method,
			c.Uncertainty.Lower,
			c.Uncertainty.Upper,
			c.Uncertainty.Cases,
			c.Uncertainty.Limitation,
		)
	}
	return b.String()
}
func ExitCode(v GateVerdict) int {
	switch v {
	case GatePass:
		return 0
	case GateFail:
		return 1
	case GateInconclusive:
		return 2
	default:
		return 3
	}
}
