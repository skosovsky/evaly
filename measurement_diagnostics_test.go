package evaly_test

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"strings"
	"testing"

	"github.com/skosovsky/evaly"
)

func TestMeasurementFailureDiagnosticIsLocalizedAndPrivate(t *testing.T) {
	// Arrange.
	config := config(t, 2)
	config.Plan.Repeats = 2
	baseline, err := evaly.Run(context.Background(), config)
	if err != nil {
		t.Fatal(err)
	}
	config.ID = "candidate"
	candidate, err := evaly.Run(context.Background(), config)
	if err != nil {
		t.Fatal(err)
	}
	for _, side := range []string{"baseline", "candidate"} {
		for _, category := range []string{"measurement_callback", "measurement_invalid"} {
			t.Run(side+"_"+category, func(t *testing.T) {
				// Arrange.
				base := evaly.AssertionObjective{ID: "checks", Revision: "1", Policy: "all"}
				objective := evaly.ObjectiveFuncs{
					Descriptor: base.Identity(), Select: base.Eligible,
					Evaluate: localizedMeasurementFailure(base, side, category),
				}
				// Act.
				result, compareErr := evaly.Compare(baseline, candidate, objective, measurementPolicy())
				raw, marshalErr := json.Marshal(result)
				// Assert.
				diagnostic := result.MeasurementDiagnostic
				if compareErr != nil || marshalErr != nil || result.Verdict != evaly.GateInvalid ||
					diagnostic == nil || diagnostic.Side != side || diagnostic.CaseID != "b" ||
					diagnostic.Repeat != 1 || diagnostic.Category != category || result.MatchedMeansAvailable ||
					strings.Contains(string(raw), "private callback secret") {
					t.Fatal(result, compareErr, marshalErr)
				}
			})
		}
	}
}

func TestNumericSourcePlanSeparatesTypoFromMissingMetric(t *testing.T) {
	// Arrange.
	experiment, err := evaly.Run(context.Background(), config(t, 2))
	if err != nil {
		t.Fatal(err)
	}
	identity := (evaly.AssertionObjective{ID: "native", Revision: "1", Policy: "all"}).Identity()
	identity.AssertionPolicy = ""
	identity.SourceMetric = "unavailable"
	for _, source := range []string{"exact", "typo"} {
		for _, pointer := range []bool{false, true} {
			t.Run(source+"_"+map[bool]string{true: "pointer", false: "value"}[pointer], func(t *testing.T) {
				// Arrange.
				descriptor := identity
				descriptor.SourceGrader = source
				numeric := evaly.NumericObjective{Descriptor: descriptor}
				var objective evaly.Objective = numeric
				if pointer {
					objective = &numeric
				}
				// Act.
				result, compareErr := evaly.Compare(experiment, experiment, objective, measurementPolicy())
				// Assert.
				if source == "typo" {
					if !errors.Is(compareErr, evaly.ErrInvalid) {
						t.Fatal(compareErr)
					}
				} else if compareErr != nil || result.Verdict != evaly.GateInconclusive ||
					len(
						result.BaselineAggregate.Excluded,
					) != 2 || result.BaselineAggregate.Excluded[0].Reason != "metric_missing" {
					t.Fatal(result, compareErr)
				}
			})
		}
	}
}

func localizedMeasurementFailure(
	base evaly.AssertionObjective,
	side, category string,
) func(evaly.TrialRecord) (evaly.Measurement, error) {
	return func(trial evaly.TrialRecord) (evaly.Measurement, error) {
		isCandidate := strings.HasPrefix(trial.ID, "candidate/")
		if trial.CaseID == "b" && trial.Repeat == 1 && isCandidate == (side == "candidate") {
			if category == "measurement_callback" {
				return evaly.Measurement{}, errors.New("private callback secret")
			}
			return evaly.Measurement{Present: true, Value: math.NaN()}, nil
		}
		return base.Measure(trial)
	}
}
