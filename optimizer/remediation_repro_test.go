package optimizer_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"

	"github.com/skosovsky/evaly"
	"github.com/skosovsky/evaly/internal/fixtures"
	"github.com/skosovsky/evaly/optimizer"
)

func TestF08RehashedSemanticMutations(t *testing.T) {
	// Arrange: mutate independent copies of a real completed search, then rehash.
	c := searchConfig(t, 30)
	result, err := optimizer.Search(context.Background(), c)
	if err != nil || result.Winner == "" {
		t.Fatal(err, result)
	}
	wire, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	mutations := map[string]func(*optimizer.Result){
		"winner_without_holdout":      func(r *optimizer.Result) { r.Holdout = nil; r.HoldoutComparison = nil },
		"round_without_candidates":    func(r *optimizer.Result) { r.RoundHistory[0].Candidates = nil; r.RoundHistory[0].Received = nil },
		"invented_aggregate_usage":    func(r *optimizer.Result) { r.ProposalUsage.Units = 999 },
		"foreign_dispatch":            func(r *optimizer.Result) { r.History[0].DispatchID = "unrelated-dispatch" },
		"evaluated_without_artifacts": removeEvaluatedArtifacts,
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			var bad optimizer.Result
			if err := json.Unmarshal(wire, &bad); err != nil {
				t.Fatal(err)
			}
			mutate(&bad)
			bad = resealSearch(t, bad)
			// Act: both entrypoints must reject semantic contradictions despite a valid digest.
			validationErr := optimizer.ValidateResult(bad)
			_, restoreErr := optimizer.RestoreResult(bad, c.Codec)
			// Assert: the baseline mode is exclusively for the disposable old-revision repro.
			if os.Getenv("EVALY_BASELINE_REPRO") == "1" {
				if validationErr != nil || restoreErr != nil {
					t.Fatalf("baseline no longer reproduces: validate=%v restore=%v", validationErr, restoreErr)
				}
			} else if validationErr == nil || restoreErr == nil {
				t.Fatalf("accepted contradiction: validate=%v restore=%v", validationErr, restoreErr)
			}
		})
	}
}

func TestF09ForeignMeasurementsNeverEnterCanonicalSlots(t *testing.T) {
	for _, phase := range []string{"calibration", "holdout"} {
		for _, callbackError := range []bool{false, true} {
			name := phase + map[bool]string{false: "/nil_error", true: "/operational_error"}[callbackError]
			t.Run(name, func(t *testing.T) {
				checkForeignMeasurement(t, phase, callbackError)
			})
		}
	}
}

func TestBoundMeasurementsAlongsideOperationalErrorRemainRestorable(t *testing.T) {
	for _, phase := range []string{"calibration", "holdout"} {
		t.Run(phase, func(t *testing.T) {
			// Arrange: a correctly bound measurement with an independent host failure.
			c := searchConfig(t, 30)
			evaluate := c.Evaluate
			c.Evaluate = func(ctx context.Context, req optimizer.EvaluationRequest[recipe, fixtures.Calculation, int]) (evaly.Experiment, error) {
				experiment, err := evaluate(ctx, req)
				if err == nil && req.Phase == phase {
					err = errors.New("post-measurement failure")
				}
				return experiment, err
			}
			// Act.
			result, err := optimizer.Search(context.Background(), c)
			_, restoreErr := optimizer.RestoreResult(result, c.Codec)
			// Assert: real bound evidence survives, even when it cannot select a winner.
			if err != nil || restoreErr != nil {
				t.Fatal(err, restoreErr, result)
			}
			if phase == "calibration" {
				if result.History[0].Experiment == nil || result.History[0].State != "failed" {
					t.Fatal(result)
				}
			} else if result.Holdout == nil || result.HoldoutComparison != nil || result.State != "stopped" {
				t.Fatal(result)
			}
		})
	}
}

func removeEvaluatedArtifacts(r *optimizer.Result) {
	rejected := r.History[0].Candidate.Revision
	r.History[0].Candidate = optimizer.CandidateRecord{}
	r.History[0].Quality = nil
	r.History[0].Comparison = nil
	r.History[0].Experiment = nil
	r.Ranking = nil
	for _, entry := range r.History {
		if entry.Candidate.Revision != rejected && entry.Quality != nil {
			r.Ranking = append(r.Ranking, entry.Candidate.Revision)
		}
	}
	r.BestMeasured = ""
	if len(r.Ranking) > 0 {
		r.BestMeasured = r.Ranking[0]
	}
	r.Winner = ""
	r.Holdout = nil
	r.HoldoutComparison = nil
}

func checkForeignMeasurement(t *testing.T, phase string, callbackError bool) {
	t.Helper()

	// Arrange: return a valid experiment belonging to another dispatch.
	c := searchConfig(t, 30)
	evaluate := c.Evaluate
	calibrationCalls, holdoutCalls := 0, 0
	c.Evaluate = func(ctx context.Context, req optimizer.EvaluationRequest[recipe, fixtures.Calculation, int]) (evaly.Experiment, error) {
		if req.Phase == "calibration" {
			calibrationCalls++
		} else {
			holdoutCalls++
		}
		if req.Phase != phase {
			return evaluate(ctx, req)
		}
		var err error
		if callbackError {
			err = errors.New("host operational failure")
		}
		if phase == "calibration" {
			return c.CalibrationBaseline, err
		}
		return c.HoldoutBaseline, err
	}
	// Act.
	result, err := optimizer.Search(context.Background(), c)
	validationErr := optimizer.ValidateResult(result)
	_, restoreErr := optimizer.RestoreResult(result, c.Codec)
	// Assert: protocol failure stops dispatch, without legalizing the foreign record.
	if os.Getenv("EVALY_BASELINE_REPRO") == "1" {
		if err != nil || validationErr == nil || restoreErr == nil {
			t.Fatalf("baseline no longer reproduces: %v %v %v", err, validationErr, restoreErr)
		}
		return
	}
	if err != nil || result.State != "stopped" || validationErr != nil || restoreErr != nil {
		t.Fatalf("invalid diagnostic result: %v %v %v %+v", err, validationErr, restoreErr, result)
	}
	assertForeignSlots(t, phase, result, calibrationCalls, holdoutCalls)
}

func assertForeignSlots(t *testing.T, phase string, result optimizer.Result, calibrationCalls, holdoutCalls int) {
	t.Helper()
	if phase == "calibration" {
		if calibrationCalls != 1 || holdoutCalls != 0 || result.History[0].Experiment != nil || result.Holdout != nil ||
			result.Reason != "evaluation_protocol_failure" {
			t.Fatal(calibrationCalls, holdoutCalls, result)
		}
	} else if holdoutCalls != 1 || result.Holdout != nil || result.HoldoutComparison != nil || result.Winner == "" || result.Reason != "holdout_evaluation_failure" {
		t.Fatal(calibrationCalls, holdoutCalls, result)
	}
}
