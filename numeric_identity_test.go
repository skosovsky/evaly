package evaly_test

import (
	"errors"
	"testing"

	"github.com/skosovsky/evaly"
)

func TestNumericObjectiveCanonicalReference(t *testing.T) {
	// Arrange.
	descriptor := evaly.ObjectiveIdentity{
		ID: "latency", Revision: "1", Unit: "ms", ScaleRevision: "milliseconds-v1",
		Minimum: 0, Maximum: 100, Direction: "lower",
		SourceGrader: "metric", SourceMetric: "latency",
		EligibilityRevision: "all-declared-v1", MissingnessRevision: "all-repeats-required-v1",
		AggregationRevision: "repeat-mean-case-mean-v1",
	}
	gate := evaly.GatePolicy{
		Revision: "1", MinimumMatchedCases: 1, MinimumCoverage: 1,
		MinimumMatchedCoverage: 1, QualityThreshold: 50, BootstrapSamples: 100,
	}
	cases := []struct {
		name   string
		change func(*evaly.ObjectiveIdentity)
		want   error
	}{
		{"canonical", func(*evaly.ObjectiveIdentity) {}, nil},
		{"no-grader", func(i *evaly.ObjectiveIdentity) { i.SourceGrader = "" }, evaly.ErrInvalid},
		{"no-metric", func(i *evaly.ObjectiveIdentity) { i.SourceMetric = "" }, evaly.ErrInvalid},
		{"assertion-policy", func(i *evaly.ObjectiveIdentity) { i.AssertionPolicy = "all" }, evaly.ErrInvalid},
		{"host-eligibility", func(i *evaly.ObjectiveIdentity) { i.EligibilityRevision = "host" }, evaly.ErrUnsupported},
		{"host-missingness", func(i *evaly.ObjectiveIdentity) { i.MissingnessRevision = "host" }, evaly.ErrUnsupported},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange.
			identity := descriptor
			tc.change(&identity)
			direct := evaly.NumericObjective{Descriptor: identity}
			policy := evaly.ComparisonPolicy{
				Version: evaly.ComparisonPolicyVersion, ObjectiveKind: "numeric", Objective: identity, Gate: gate,
			}
			// Act.
			validateErr := direct.Validate()
			resolved, resolveErr := policy.Resolve()
			_, eligibilityErr := direct.Eligible(evaly.CaseIdentity{})
			_, measureErr := direct.Measure(evaly.TrialRecord{})
			// Assert.
			if !errors.Is(validateErr, tc.want) || !errors.Is(resolveErr, tc.want) ||
				!errors.Is(eligibilityErr, tc.want) || (tc.want != nil && !errors.Is(measureErr, tc.want)) {
				t.Fatal(validateErr, resolveErr, eligibilityErr, measureErr)
			}
			if direct.Identity() != identity || (tc.want == nil && resolved.Identity() != identity) {
				t.Fatal("descriptor changed")
			}
		})
	}
}
