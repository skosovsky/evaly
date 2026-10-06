package evaly_test

import (
	"context"
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/skosovsky/evaly"
)

func measurementPolicy() evaly.GatePolicy {
	return evaly.GatePolicy{
		Revision:               "policy-v1",
		MinimumCoverage:        .5,
		MinimumMatchedCases:    2,
		MinimumMatchedCoverage: .02,
		QualityThreshold:       .5,
		MaximumRegression:      .1,
		BootstrapSamples:       100,
	}
}
func TestMatchedDenominatorAndIndependentRepeats(t *testing.T) {
	// Arrange.
	b, c := config(t, 1), config(t, 1)
	cases := make([]evaly.Case[input, int], 100)
	for i := range cases {
		r := 1
		cases[i] = evaly.Case[input, int]{
			ID:        fmt.Sprintf("case-%03d", i),
			Input:     input{Numbers: []int{1}},
			Reference: &r,
		}
	}
	d, e := (evaly.DatasetDraft[input, int]{Cases: cases, Selection: "all"}).Seal(
		evaly.JSONCodec[input]{ID: "i", Version: "1"},
		evaly.JSONCodec[int]{ID: "r", Version: "1"},
	)
	if e != nil {
		t.Fatal(e)
	}
	b.Dataset = d
	c.Dataset = d
	b.ID = "base"
	c.ID = "candidate"
	b.Plan.Repeats = 3
	c.Plan.Repeats = 3
	makeGrader := func(side string) evaly.Grader[input, int, int] {
		return evaly.GraderFunc[input, int, int]{
			Identity: evaly.GraderRevision{ID: "exact", Implementation: "go-v1", Rubric: "equal-v1"},
			Evaluate: func(_ context.Context, v evaly.View[input, int, int]) (evaly.Grade, error) {
				var n int
				fmt.Sscanf(v.Case.ID, "case-%03d", &n)
				present := n < 50
				if side == "c" {
					present = n >= 49 && n < 99
				}
				if !present {
					return evaly.Grade{Status: evaly.InsufficientEvidence}, nil
				}
				return evaly.Grade{Status: evaly.Scored, Assertions: []evaly.Assertion{{Name: "pass", Pass: true}}}, nil
			},
		}
	}
	b.Graders = []evaly.Grader[input, int, int]{makeGrader("b")}
	c.Graders = []evaly.Grader[input, int, int]{makeGrader("c")}
	// Act.
	be, e := evaly.Run(context.Background(), b)
	if e != nil {
		t.Fatal(e)
	}
	ce, e := evaly.Run(context.Background(), c)
	if e != nil {
		t.Fatal(e)
	}
	comparison, e := evaly.Compare(
		be,
		ce,
		evaly.AssertionObjective{ID: "assertion", Revision: "v1", Policy: "all"},
		measurementPolicy(),
	)
	// Assert.
	if e != nil {
		t.Fatal(e)
	}
	if comparison.Verdict != evaly.GateInconclusive || comparison.MatchedEligible != 100 ||
		comparison.MatchedCases != 1 ||
		comparison.MatchedCoverage != .01 ||
		comparison.Uncertainty.Cases != 1 ||
		comparison.BaselineAggregate.Scored != 50 ||
		comparison.CandidateAggregate.Scored != 50 {
		t.Fatalf("%+v", comparison)
	}
	if len(comparison.BaselineAggregate.Excluded) != 50 ||
		comparison.UncertaintyReason != "one_independent_case_degenerate" ||
		!strings.Contains(evaly.Report(comparison), "matched 1/100") {
		t.Fatal(evaly.Report(comparison))
	}
}
func TestNumericObjectiveNativeScaleAndMissingness(t *testing.T) {
	// Arrange.
	descriptor := evaly.ObjectiveIdentity{
		ID:                  "latency",
		Revision:            "v1",
		Unit:                "ms",
		ScaleRevision:       "milliseconds-v1",
		Minimum:             0,
		Maximum:             100,
		Direction:           "lower",
		EligibilityRevision: "all-declared-v1",
		MissingnessRevision: "all-repeats-required-v1",
		AggregationRevision: "repeat-mean-case-mean-v1",
	}
	descriptor.SourceGrader, descriptor.SourceMetric = "metric", "latency"
	o := evaly.NumericObjective{Descriptor: descriptor}
	metric := evaly.Metric{
		Name:          "latency",
		Unit:          "ms",
		ScaleRevision: "milliseconds-v1",
		Value:         42,
		Minimum:       0,
		Maximum:       100,
		Direction:     "lower",
	}
	trial := evaly.TrialRecord{
		Grades: []evaly.Grade{
			{Revision: evaly.GraderRevision{ID: "metric"}, Status: evaly.Scored, Metrics: []evaly.Metric{metric}},
		},
	}
	// Act.
	m, e := o.Measure(trial)
	// Assert.
	if e != nil || !m.Present || m.Value != 42 {
		t.Fatal(m, e)
	}
	for _, mutate := range []func(*evaly.Metric){func(m *evaly.Metric) { m.Unit = "seconds" }, func(m *evaly.Metric) { m.ScaleRevision = "v2" }, func(m *evaly.Metric) { m.Direction = "higher" }, func(m *evaly.Metric) { m.Value = math.NaN() }, func(m *evaly.Metric) { m.Maximum = 200 }} {
		bad := trial
		bad.Grades = []evaly.Grade{
			{Revision: evaly.GraderRevision{ID: "metric"}, Status: evaly.Scored, Metrics: []evaly.Metric{metric}},
		}
		mutate(&bad.Grades[0].Metrics[0])
		if _, eLocal := o.Measure(bad); eLocal == nil {
			t.Fatal("incompatible metric accepted")
		}
	}
	missing, e := o.Measure(evaly.TrialRecord{})
	if e != nil || missing.Present || missing.Reason == "" {
		t.Fatal(missing, e)
	}
}
func TestNumericComparisonDirectionAndMatchedMeans(t *testing.T) {
	for _, direction := range []string{"higher", "lower"} {
		t.Run(direction, func(t *testing.T) {
			checkNumericComparisonDirectionAndMatchedMeans(t, &direction)
		},

		// Act.

		// Assert.

		// Arrange: a worse candidate, with quality permissive so regression is the deciding gate.

		// Act.

		// Assert.

		)
	}
}

func TestMeasurementEmptyEligibilityAndContractConflict(t *testing.T) {
	// Arrange.
	c := config(t, 2)
	experiment, e := evaly.Run(context.Background(), c)
	if e != nil {
		t.Fatal(e)
	}
	base := evaly.AssertionObjective{ID: "assertions", Revision: "v1", Policy: "all"}
	o := evaly.ObjectiveFuncs{Descriptor: base.Identity(), Select: func(evaly.CaseIdentity) (evaly.Eligibility, error) {
		return evaly.Eligibility{Reason: "host_excluded"}, nil
	}, Evaluate: base.Measure}
	p := measurementPolicy()
	// Act.
	r, e := evaly.Compare(experiment, experiment, o, p)
	// Assert.
	if e != nil || r.Verdict != evaly.GateInconclusive || r.MatchedEligible != 0 || r.MatchedCases != 0 ||
		r.Uncertainty != nil ||
		r.UncertaintyReason != "no_matched_cases" ||
		r.BaselineAggregate.MeanAvailable ||
		r.MatchedMeansAvailable ||
		r.BaselineAggregate.Excluded[0].Reason != "host_excluded" {
		t.Fatal(r, e)
	}
	// Arrange.
	o.Select = func(evaly.CaseIdentity) (evaly.Eligibility, error) { return evaly.Eligibility{Eligible: true}, nil }
	o.Evaluate = func(evaly.TrialRecord) (evaly.Measurement, error) {
		return evaly.Measurement{Present: true, Value: math.NaN()}, nil
	}
	// Act.
	r, e = evaly.Compare(experiment, experiment, o, p)
	// Assert.
	if e != nil || r.Verdict != evaly.GateInvalid || r.Revision == "" ||
		r.Reasons[0] != "measurement_contract_invalid" {
		t.Fatal(r, e)
	}
}
func TestObjectiveProvenanceAndUnsupportedAggregation(t *testing.T) {
	// Arrange.
	all := evaly.AssertionObjective{ID: "assertions", Revision: "v1", Policy: "all"}
	anyObjective := all
	anyObjective.Policy = "any"
	descriptor := all.Identity()
	descriptor.AssertionPolicy = ""
	descriptor.SourceGrader, descriptor.SourceMetric = "g", "metric"
	numeric := evaly.NumericObjective{Descriptor: descriptor}
	// Act.
	identity := numeric.Identity()
	bad := identity
	bad.AggregationRevision = "median-v1"
	overflow := identity
	overflow.Minimum = -1e308
	overflow.Maximum = 1e308
	// Assert.
	if all.Identity() == anyObjective.Identity() || identity.SourceGrader != "g" || identity.SourceMetric != "metric" ||
		evaly.ValidateObjectiveIdentity(bad) == nil ||
		evaly.ValidateObjectiveIdentity(overflow) == nil {
		t.Fatal(identity)
	}
}

func TestNumericSubnormalMeansRemainInScale(t *testing.T) {
	// Arrange.
	minimum := math.SmallestNonzeroFloat64
	maximum := 2 * minimum
	b, c := config(t, 2), config(t, 2)
	b.ID = "subnormal-base"
	c.ID = "subnormal-candidate"
	b.Plan.Repeats = 2
	c.Plan.Repeats = 2
	grader := func(value float64) evaly.Grader[input, int, int] {
		return evaly.GraderFunc[input, int, int]{
			Identity: evaly.GraderRevision{ID: "tiny", Implementation: "v1", Rubric: "v1"},
			Evaluate: func(context.Context, evaly.View[input, int, int]) (evaly.Grade, error) {
				return evaly.Grade{
					Status: evaly.Scored,
					Metrics: []evaly.Metric{
						{
							Name:          "tiny",
							Unit:          "tiny-unit",
							ScaleRevision: "tiny-v1",
							Value:         value,
							Minimum:       minimum,
							Maximum:       maximum,
							Direction:     "higher",
						},
					},
				}, nil
			},
		}
	}
	b.Graders = []evaly.Grader[input, int, int]{grader(minimum)}
	c.Graders = []evaly.Grader[input, int, int]{grader(maximum)}
	objective := evaly.NumericObjective{
		Descriptor: evaly.ObjectiveIdentity{
			SourceGrader: "tiny", SourceMetric: "tiny",
			ID:                  "tiny",
			Revision:            "v1",
			Unit:                "tiny-unit",
			ScaleRevision:       "tiny-v1",
			Minimum:             minimum,
			Maximum:             maximum,
			Direction:           "higher",
			EligibilityRevision: "all-declared-v1",
			MissingnessRevision: "all-repeats-required-v1",
			AggregationRevision: "repeat-mean-case-mean-v1",
		},
	}
	p := measurementPolicy()
	p.QualityThreshold = maximum
	p.MaximumRegression = 0
	// Act.
	be, e := evaly.Run(context.Background(), b)
	if e != nil {
		t.Fatal(e)
	}
	ce, e := evaly.Run(context.Background(), c)
	if e != nil {
		t.Fatal(e)
	}
	r, e := evaly.Compare(be, ce, objective, p)
	// Assert.
	if e != nil || r.Verdict != evaly.GatePass || r.BaselineAggregate.Mean != minimum ||
		r.CandidateAggregate.Mean != maximum ||
		r.MatchedBaselineMean != minimum ||
		r.MatchedCandidateMean != maximum ||
		r.Delta != minimum ||
		r.Uncertainty.Lower != minimum ||
		r.Uncertainty.Upper != minimum {
		t.Fatal(r, e)
	}
}
func checkNumericComparisonDirectionAndMatchedMeans(t *testing.T, direction *string) {
	t.Helper()
	// Arrange.
	b, c := config(t, 3), config(t, 3)
	b.ID = "base"
	c.ID = "candidate"
	grader := func(value float64) evaly.Grader[input, int, int] {
		return evaly.GraderFunc[input, int, int]{
			Identity: evaly.GraderRevision{ID: "metric", Implementation: "v1", Rubric: "v1"},
			Evaluate: func(context.Context, evaly.View[input, int, int]) (evaly.Grade, error) {
				return evaly.Grade{
					Status: evaly.Scored,
					Metrics: []evaly.Metric{
						{
							Name:          "native",
							Unit:          "points",
							ScaleRevision: "v1",
							Value:         value,
							Minimum:       0,
							Maximum:       100,
							Direction:     (*direction),
						},
					},
				}, nil
			},
		}
	}
	b.Graders = []evaly.Grader[input, int, int]{grader(50)}
	value := 60.0
	quality := 55.0
	if (*direction) == "lower" {
		value = 40
		quality = 45
	}
	c.Graders = []evaly.Grader[input, int, int]{grader(value)}
	o := evaly.NumericObjective{
		Descriptor: evaly.ObjectiveIdentity{
			SourceGrader: "metric", SourceMetric: "native",
			ID:                  "native",
			Revision:            "v1",
			Unit:                "points",
			ScaleRevision:       "v1",
			Minimum:             0,
			Maximum:             100,
			Direction:           (*direction),
			EligibilityRevision: "all-declared-v1",
			MissingnessRevision: "all-repeats-required-v1",
			AggregationRevision: "repeat-mean-case-mean-v1",
		},
	}
	p := measurementPolicy()
	p.QualityThreshold = quality
	p.MaximumRegression = 5

	be, e := evaly.Run(context.Background(), b)
	if e != nil {
		t.Fatal(e)
	}
	ce, e := evaly.Run(context.Background(), c)
	if e != nil {
		t.Fatal(e)
	}
	r, e := evaly.Compare(be, ce, o, p)

	if e != nil || r.Verdict != evaly.GatePass || r.MatchedCases != 3 || r.MatchedCandidateMean != value ||
		r.Delta != value-50 ||
		r.Uncertainty.Lower != r.Delta ||
		r.Uncertainty.Upper != r.Delta {
		t.Fatal(r, e)
	}
	// Act / Assert: equality passes in both directions; the adjacent float on
	// the failing side crosses the same native-scale threshold.
	p.QualityThreshold = value
	equal, equalErr := evaly.Compare(be, ce, o, p)
	toward := math.Inf(1)
	if (*direction) == "lower" {
		toward = math.Inf(-1)
	}
	p.QualityThreshold = math.Nextafter(value, toward)
	adjacent, adjacentErr := evaly.Compare(be, ce, o, p)
	if equalErr != nil || equal.Verdict != evaly.GatePass || adjacentErr != nil ||
		adjacent.Verdict != evaly.GateFail || len(adjacent.Reasons) != 1 || adjacent.Reasons[0] != "quality_threshold" {
		t.Fatal(equal, equalErr, adjacent, adjacentErr)
	}
	p.QualityThreshold = quality - 20
	if (*direction) == "higher" {
		p.QualityThreshold = quality + 20
	}
	r, e = evaly.Compare(be, ce, o, p)
	if e != nil || r.Verdict != evaly.GateFail {
		t.Fatal(r, e)
	}

	worse := 40.0
	p.QualityThreshold = 0
	if (*direction) == "lower" {
		worse = 60
		p.QualityThreshold = 100
	}
	c.Graders = []evaly.Grader[input, int, int]{grader(worse)}

	worseExperiment, e := evaly.Run(context.Background(), c)
	if e != nil {
		t.Fatal(e)
	}
	r, e = evaly.Compare(be, worseExperiment, o, p)

	if e != nil || r.Verdict != evaly.GateFail || len(r.Reasons) != 1 || r.Reasons[0] != "regression" {
		t.Fatal(r, e)
	}
}
