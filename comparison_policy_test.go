package evaly_test

import (
	"errors"
	"math"
	"testing"

	"github.com/skosovsky/evaly"
)

func TestSerializableComparisonPolicyRejectsUnsupportedSemantics(t *testing.T) {
	// Arrange.
	assertion := evaly.AssertionObjective{ID: "checks", Revision: "1", Policy: "all"}
	policy := evaly.ComparisonPolicy{
		Version:       evaly.ComparisonPolicyVersion,
		ObjectiveKind: "assertion",
		Objective:     assertion.Identity(),
		Gate: evaly.GatePolicy{
			Revision:               "gate-v1",
			MinimumMatchedCases:    1,
			MinimumCoverage:        1,
			MinimumMatchedCoverage: 1,
			QualityThreshold:       1,
			BootstrapSamples:       100,
		},
	}
	cases := []struct {
		name   string
		mutate func(*evaly.ComparisonPolicy)
		want   error
	}{
		{"zero-version", func(p *evaly.ComparisonPolicy) { p.Version = 0 }, evaly.ErrUnsupported},
		{"old-version", func(p *evaly.ComparisonPolicy) { p.Version = 1 }, evaly.ErrUnsupported},
		{"callback-kind", func(p *evaly.ComparisonPolicy) { p.ObjectiveKind = "callback" }, evaly.ErrUnsupported},
		{"forged-scale", func(p *evaly.ComparisonPolicy) { p.Objective.Maximum = 2 }, evaly.ErrInvalid},
		{"null-size", func(p *evaly.ComparisonPolicy) { p.Gate.MinimumMatchedCases = 0 }, evaly.ErrInvalid},
		{"nan-quality", func(p *evaly.ComparisonPolicy) { p.Gate.QualityThreshold = math.NaN() }, evaly.ErrInvalid},
		{
			"infinite-regression",
			func(p *evaly.ComparisonPolicy) { p.Gate.MaximumRegression = math.Inf(1) },
			evaly.ErrInvalid,
		},
		{
			"unsupported-assertion-selection",
			func(p *evaly.ComparisonPolicy) { p.Objective.EligibilityRevision = "host-callback-v1" },
			evaly.ErrUnsupported,
		},
		{"unsupported-numeric-selection", func(p *evaly.ComparisonPolicy) {
			p.ObjectiveKind = "numeric"
			p.Objective.AssertionPolicy = ""
			p.Objective.SourceGrader = "g"
			p.Objective.SourceMetric = "m"
			p.Objective.EligibilityRevision = "host-callback-v1"
		}, evaly.ErrUnsupported},
		{"unsupported-numeric-missingness", func(p *evaly.ComparisonPolicy) {
			p.ObjectiveKind = "numeric"
			p.Objective.AssertionPolicy = ""
			p.Objective.SourceGrader = "g"
			p.Objective.SourceMetric = "m"
			p.Objective.MissingnessRevision = "impute-zero-v1"
		}, evaly.ErrUnsupported},
		{"missing-numeric-grader", func(p *evaly.ComparisonPolicy) {
			p.ObjectiveKind = "numeric"
			p.Objective.AssertionPolicy = ""
			p.Objective.SourceMetric = "m"
		}, evaly.ErrInvalid},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange.
			p := policy
			tc.mutate(&p)
			// Act.
			_, err := p.Resolve()
			// Assert.
			if !errors.Is(err, tc.want) {
				t.Fatalf("got %v want %v", err, tc.want)
			}
		})
	}
	// Act.
	objective, err := policy.Resolve()
	// Assert.
	if err != nil || objective.Identity() != assertion.Identity() {
		t.Fatalf("resolve: %v %v", objective, err)
	}
}
