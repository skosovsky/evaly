package evaly

import (
	"fmt"
	"math"
	"math/big"
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
	Revision               string  `json:"revision"`
	MinimumMatchedCases    int     `json:"minimum_matched_cases"`
	MinimumMatchedCoverage float64 `json:"minimum_matched_coverage"`
	MinimumCoverage        float64 `json:"minimum_coverage"`
	MinimumQuality         float64 `json:"minimum_quality"`
	MaximumRegression      float64 `json:"maximum_regression"`
	BootstrapSamples       int     `json:"bootstrap_samples"`
	Seed                   int64   `json:"seed"`
}
type Aggregate struct {
	Eligible        int         `json:"eligible"`
	Scored          int         `json:"scored"`
	Coverage        float64     `json:"coverage"`
	MeanAvailable   bool        `json:"mean_available"`
	Mean            float64     `json:"mean"`
	SetupFailures   int         `json:"setup_failures"`
	TargetFailures  int         `json:"target_failures"`
	GraderFailures  int         `json:"grader_failures"`
	CleanupFailures int         `json:"cleanup_failures"`
	Excluded        []Exclusion `json:"excluded"`
}
type Exclusion struct {
	CaseID string `json:"case_id"`
	Reason string `json:"reason"`
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
	Trials                []TrialSummary    `json:"trials"`
	Version               int               `json:"version"`
	Revision              string            `json:"revision"`
	Baseline              string            `json:"baseline"`
	Candidate             string            `json:"candidate"`
	Verdict               GateVerdict       `json:"verdict"`
	Reasons               []string          `json:"reasons"`
	Policy                GatePolicy        `json:"policy"`
	BaselineAggregate     Aggregate         `json:"baseline_aggregate"`
	CandidateAggregate    Aggregate         `json:"candidate_aggregate"`
	Pairs                 []CaseDifference  `json:"pairs"`
	Uncertainty           *Interval         `json:"uncertainty,omitempty"`
	Objective             ObjectiveIdentity `json:"objective"`
	MatchedEligible       int               `json:"matched_eligible"`
	MatchedCases          int               `json:"matched_cases"`
	MatchedCoverage       float64           `json:"matched_coverage"`
	MatchedMeansAvailable bool              `json:"matched_means_available"`
	MatchedBaselineMean   float64           `json:"matched_baseline_mean"`
	MatchedCandidateMean  float64           `json:"matched_candidate_mean"`
	Delta                 float64           `json:"delta"`
	UncertaintyReason     string            `json:"uncertainty_reason"`
	Unit                  string            `json:"unit"`
}

func aggregate(
	r ExperimentRecord,
	objective Objective,
	eligible map[string]bool,
) (Aggregate, map[string]float64, error) {
	a := Aggregate{Excluded: []Exclusion{}}
	identity := objective.Identity()
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
		if !eligible[cs.ID] {
			a.Excluded = append(a.Excluded, Exclusion{cs.ID, "not_eligible"})
			continue
		}
		a.Eligible++
		reason := ""
		sum := 0.0
		for repeat := range r.Manifest.Plan.Repeats {
			t, ok := final[cs.ID+"/"+identityPart(repeat)]
			if !ok {
				reason = "trial_missing"
				break
			}
			if t.Status != Completed {
				reason = string(t.Status)
				break
			}
			if t.UsageError != "" {
				reason = "usage_error"
				break
			}
			if t.GradingState != "complete" || len(t.Grades) != len(r.Manifest.Graders) {
				reason = "grading_incomplete"
				break
			}
			copy, e := cloneJSON(t)
			if e != nil {
				return a, values, e
			}
			m, e := objective.Measure(copy)
			if objective.Identity() != identity {
				return a, values, ErrConflict
			}
			if e != nil {
				return a, values, e
			}
			if e = ValidateMeasurement(identity, m); e != nil {
				return a, values, e
			}
			if !m.Present {
				reason = m.Reason
				break
			}
			sum = meanStep(sum, m.Value, repeat+1)
		}
		if reason != "" {
			a.Excluded = append(a.Excluded, Exclusion{cs.ID, reason})
			continue
		}
		value := sum
		values[cs.ID] = value
		a.Scored++
		a.Mean = meanStep(a.Mean, value, a.Scored)
	}
	if a.Eligible > 0 {
		a.Coverage = float64(a.Scored) / float64(a.Eligible)
	}
	if a.Scored > 0 {
		a.MeanAvailable = true
	}
	return a, values, nil
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
		x, _ = canonical(a.PairSchedule)
		y, _ = canonical(b.PairSchedule)
		if string(x) != string(y) {
			return false
		}
	}
	return true
}
func Compare(baseline, candidate Experiment, objective Objective, p GatePolicy) (Comparison, error) {
	c := Comparison{
		Version:   2,
		Baseline:  baseline.Revision(),
		Candidate: candidate.Revision(),
		Policy:    p,
		Unit:      "case",
		Reasons:   []string{},
		Pairs:     []CaseDifference{},
	}
	if ValidatePort(objective) != nil {
		return c, ErrInvalid
	}
	id := objective.Identity()
	c.Objective = id
	if validateGatePolicy(id, p) != nil {
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

	invalid := func(reason string) (Comparison, error) {
		c.Verdict = GateInvalid
		c.Reasons = append(c.Reasons, reason)
		c.UncertaintyReason = "comparison_invalid"
		raw, e := canonical(c)
		if e != nil {
			return c, e
		}
		c.Revision = digest(raw)
		return c, nil
	}
	if !compatible(b.Manifest, cv.Manifest) {
		return invalid("incompatible_manifests")
	}
	eligible := map[string]bool{}
	eligibilityReasons := map[string]string{}
	for _, cs := range b.Manifest.Cases {
		v, e := objective.Eligible(cs)
		if objective.Identity() != id {
			return invalid("objective_identity_changed")
		}
		if e != nil {
			return invalid("eligibility_contract_error")
		}
		if (!v.Eligible && v.Reason == "") || (v.Eligible && v.Reason != "") {
			return invalid("eligibility_contract_invalid")
		}
		eligible[cs.ID] = v.Eligible
		eligibilityReasons[cs.ID] = v.Reason
		if v.Eligible {
			c.MatchedEligible++
		}
	}
	ba, bvalues, e := aggregate(b, objective, eligible)
	if e != nil {
		return invalid("measurement_contract_invalid")
	}
	ca, cvalues, e := aggregate(cv, objective, eligible)
	if e != nil {
		return invalid("measurement_contract_invalid")
	}
	for _, a := range []*Aggregate{&ba, &ca} {
		for i := range a.Excluded {
			if a.Excluded[i].Reason == "not_eligible" {
				a.Excluded[i].Reason = eligibilityReasons[a.Excluded[i].CaseID]
			}
		}
	}
	c.BaselineAggregate = ba
	c.CandidateAggregate = ca
	differences := []float64{}
	for _, cs := range b.Manifest.Cases {
		bv, bok := bvalues[cs.ID]
		v, cok := cvalues[cs.ID]
		if bok && cok {
			d := v - bv
			c.Pairs = append(c.Pairs, CaseDifference{cs.ID, bv, v, d})
			differences = append(differences, d)
			n := len(differences)
			c.MatchedBaselineMean = meanStep(c.MatchedBaselineMean, bv, n)
			c.MatchedCandidateMean = meanStep(c.MatchedCandidateMean, v, n)
			c.Delta = meanStep(c.Delta, d, n)
		}
	}
	c.MatchedCases = len(differences)
	if c.MatchedEligible > 0 {
		c.MatchedCoverage = float64(c.MatchedCases) / float64(c.MatchedEligible)
	}
	if c.MatchedCases > 0 {
		c.MatchedMeansAvailable = true
		c.Uncertainty = bootstrap(differences, p.BootstrapSamples, p.Seed)
	}
	switch c.MatchedCases {
	case 0:
		c.UncertaintyReason = "no_matched_cases"
	case 1:
		c.UncertaintyReason = "one_independent_case_degenerate"
	default:
		c.UncertaintyReason = "independent_representative_cases_assumed"
		if c.Uncertainty.Lower == c.Uncertainty.Upper {
			c.UncertaintyReason = "constant_sample_degenerate"
		}
	}
	c.Verdict = GatePass
	if !compatible(b.Manifest, cv.Manifest) {
		c.Verdict = GateInvalid
		c.Reasons = append(c.Reasons, "incompatible_manifests")
	} else if ba.Coverage < p.MinimumCoverage || ca.Coverage < p.MinimumCoverage || c.MatchedCases < p.MinimumMatchedCases || c.MatchedCoverage < p.MinimumMatchedCoverage {
		c.Verdict = GateInconclusive
		c.Reasons = append(c.Reasons, "insufficient_matched_coverage")
	} else {
		badQuality := c.MatchedCandidateMean < p.MinimumQuality
		regression := c.Delta < -p.MaximumRegression
		if id.Direction == "lower" {
			badQuality = c.MatchedCandidateMean > p.MinimumQuality
			regression = c.Delta > p.MaximumRegression
		}
		if badQuality {
			c.Verdict = GateFail
			c.Reasons = append(c.Reasons, "minimum_quality")
		}
		if regression {
			c.Verdict = GateFail
			c.Reasons = append(c.Reasons, "regression")
		}
	}
	if objective.Identity() != id {
		return invalid("objective_identity_changed")
	}
	raw, e := canonical(c)
	if e != nil {
		return c, e
	}
	c.Revision = digest(raw)
	return c, nil
}

// meanStep avoids sum overflow and preserves identical subnormal values.
// Opposite-sign finite extremes can overflow subtraction; that rare branch
// computes the convex update in wider arithmetic before rounding to float64.
func meanStep(mean, value float64, count int) float64 {
	if count == 1 {
		return value
	}
	difference := value - mean
	if !math.IsInf(difference, 0) {
		return mean + difference/float64(count)
	}
	accumulator := new(big.Float).SetPrec(256).SetFloat64(mean)
	weight := new(big.Float).SetPrec(256).SetInt64(int64(count - 1))
	accumulator.Mul(accumulator, weight)
	accumulator.Add(accumulator, new(big.Float).SetPrec(256).SetFloat64(value))
	accumulator.Quo(accumulator, new(big.Float).SetPrec(256).SetInt64(int64(count)))
	result, _ := accumulator.Float64()
	return result
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
		for j := range values {
			means[i] = meanStep(means[i], values[rng.Intn(len(values))], j+1)
		}
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
		c.Objective.ID,
		c.Unit,
	)
	for _, row := range []struct {
		name string
		a    Aggregate
	}{{"baseline", c.BaselineAggregate}, {"candidate", c.CandidateAggregate}} {
		fmt.Fprintf(
			&b,
			"%s: scored %d/%d; coverage %.3f; observed mean %s; setup=%d target=%d grader=%d cleanup=%d\n",
			row.name,
			row.a.Scored,
			row.a.Eligible,
			row.a.Coverage,
			availableMean(row.a.Mean, row.a.MeanAvailable),
			row.a.SetupFailures,
			row.a.TargetFailures,
			row.a.GraderFailures,
			row.a.CleanupFailures,
		)
	}
	fmt.Fprintf(
		&b,
		"objective %s@%s: unit=%s direction=%s scale=%s; matched %d/%d coverage %.3f; baseline mean %s; candidate mean %s; delta %s; uncertainty %s\n",
		c.Objective.ID,
		c.Objective.Revision,
		c.Objective.Unit,
		c.Objective.Direction,
		c.Objective.ScaleRevision,
		c.MatchedCases,
		c.MatchedEligible,
		c.MatchedCoverage,
		availableMean(c.MatchedBaselineMean, c.MatchedMeansAvailable),
		availableMean(c.MatchedCandidateMean, c.MatchedMeansAvailable),
		availableMean(c.Delta, c.MatchedMeansAvailable),
		c.UncertaintyReason,
	)
	for _, row := range []struct {
		name string
		a    Aggregate
	}{{"baseline", c.BaselineAggregate}, {"candidate", c.CandidateAggregate}} {
		for _, x := range row.a.Excluded {
			fmt.Fprintf(&b, "%s excluded case %s: %s\n", row.name, x.CaseID, x.Reason)
		}
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
func availableMean(value float64, available bool) string {
	if !available {
		return "unavailable"
	}
	return fmt.Sprintf("%.3f", value)
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

func validateGatePolicy(id ObjectiveIdentity, p GatePolicy) error {
	if ValidateObjectiveIdentity(id) != nil || p.Revision == "" || !finiteNonnegative(p.MinimumCoverage) ||
		p.MinimumCoverage > 1 ||
		p.MinimumMatchedCases < 1 ||
		!finiteNonnegative(p.MinimumMatchedCoverage) ||
		p.MinimumMatchedCoverage > 1 ||
		math.IsNaN(p.MinimumQuality) ||
		math.IsInf(p.MinimumQuality, 0) ||
		p.MinimumQuality < id.Minimum ||
		p.MinimumQuality > id.Maximum ||
		!finiteNonnegative(p.MaximumRegression) ||
		p.BootstrapSamples < 100 ||
		p.BootstrapSamples > 100000 {
		return ErrInvalid
	}
	return nil
}
