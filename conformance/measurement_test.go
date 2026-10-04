package conformance_test

import (
	"testing"

	"github.com/skosovsky/evaly"
	"github.com/skosovsky/evaly/conformance"
)

func TestMeasurementReferenceConformance(t *testing.T) {
	revision := evaly.GraderRevision{ID: "quality", Implementation: "v1", Rubric: "v1"}
	cs := evaly.CaseIdentity{ID: "case", Revision: "case-revision"}
	t.Run("assertion", func(t *testing.T) {
		// Arrange: each call constructs an independent permitted record.
		factory := func() evaly.TrialRecord {
			return evaly.TrialRecord{
				Status: evaly.Completed,
				Grades: []evaly.Grade{
					{
						Revision:   revision,
						Status:     evaly.Scored,
						Assertions: []evaly.Assertion{{Name: "quality", Pass: true}},
					},
				},
			}
		}
		// Act / Assert.
		conformance.Objective(
			t,
			evaly.AssertionObjective{ID: "assertions", Revision: "v1", Policy: "all"},
			cs,
			factory,
			evaly.Measurement{Present: true, Value: 1},
		)
	})
	for _, direction := range []string{"higher", "lower"} {
		t.Run("numeric_"+direction, func(t *testing.T) {
			// Arrange: a numeric grade retains native units and an explicit scale.
			identity := evaly.ObjectiveIdentity{
				ID:                  "numeric",
				Revision:            "v1",
				Unit:                "points",
				ScaleRevision:       "points-v1",
				Minimum:             0,
				Maximum:             10,
				Direction:           direction,
				EligibilityRevision: "all-v1",
				MissingnessRevision: "all-repeats-v1",
				AggregationRevision: "repeat-mean-case-mean-v1",
			}
			factory := func() evaly.TrialRecord {
				return evaly.TrialRecord{
					Status: evaly.Completed,
					Grades: []evaly.Grade{
						{
							Revision: revision,
							Status:   evaly.Scored,
							Metrics: []evaly.Metric{
								{
									Name:          "quality",
									Unit:          "points",
									ScaleRevision: "points-v1",
									Minimum:       0,
									Maximum:       10,
									Value:         7,
									Direction:     direction,
								},
							},
						},
					},
				}
			}
			// Act / Assert.
			conformance.Objective(
				t,
				evaly.NumericObjective{Descriptor: identity, GraderID: revision.ID, MetricName: "quality"},
				cs,
				factory,
				evaly.Measurement{Present: true, Value: 7},
			)
		})
	}
	t.Run("calibration", func(t *testing.T) { conformance.Calibration(t, evaly.Calibrate) })
}
