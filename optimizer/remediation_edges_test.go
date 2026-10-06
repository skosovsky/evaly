package optimizer_test

import (
	"context"
	"errors"
	"math"
	"strings"
	"testing"

	"github.com/skosovsky/evaly"
	"github.com/skosovsky/evaly/internal/fixtures"
	"github.com/skosovsky/evaly/optimizer"
)

func TestStaticProposalMaximumRejectedBeforeDispatch(t *testing.T) {
	// Arrange: two sealed static candidates cannot fit the one-candidate plan.
	c := searchConfig(t, 30)
	c.MaximumCandidates = 1
	evaluations := 0
	c.Evaluate = func(context.Context, optimizer.EvaluationRequest[recipe, fixtures.Calculation, int]) (evaly.Experiment, error) {
		evaluations++
		return evaly.Experiment{}, nil
	}
	// Act.
	_, err := optimizer.Search(context.Background(), c)
	// Assert: preflight rejects rather than truncating or spending a proposal claim.
	if !errors.Is(err, evaly.ErrInvalid) || evaluations != 0 || c.Budget.(*evaly.MemoryBudget).Used() != 0 {
		t.Fatal(err, evaluations)
	}
}

func TestOversizedProposalRetainsBoundedDiagnostics(t *testing.T) {
	// Arrange: an operational proposer exceeds the advertised capacity.
	c := searchConfig(t, 30)
	c.Proposal = optimizer.ProposalFunc[recipe, fixtures.Calculation, int]{
		Identity: "oversized-v1",
		Generate: func(context.Context, optimizer.ProposalRequest[fixtures.Calculation, int]) (optimizer.ProposalResult[recipe], error) {
			return optimizer.ProposalResult[recipe]{
				Candidates: []optimizer.Proposal[recipe]{{ID: "one"}, {ID: "two"}, {ID: "three"}},
				Usage:      evaly.Usage{Known: true},
			}, nil
		},
	}
	// Act.
	result, err := optimizer.Search(context.Background(), c)
	_, restoreErr := optimizer.RestoreResult(result, c.Codec)
	// Assert: count/truncation is explicit, no oversized batch is evaluated.
	if err != nil || restoreErr != nil || result.Reason != "proposal_limit" || len(result.History) != 0 {
		t.Fatal(err, restoreErr, result)
	}
	round := result.RoundHistory[0]
	if round.ReceivedCount != 3 || !round.ReceivedTruncated || len(round.Received) != 2 {
		t.Fatal(round)
	}
}

type encodingRejectCodec struct{ evaly.JSONCodec[recipe] }

func (c encodingRejectCodec) Encode(recipe) ([]byte, error) { return nil, evaly.ErrInvalid }

func TestEncodingRejectedHistoryRestores(t *testing.T) {
	// Arrange: host codec rejects a proposal without sealing its description.
	c := searchConfig(t, 30)
	c.Codec = encodingRejectCodec{evaly.JSONCodec[recipe]{ID: "reject", Version: "1"}}
	c.Proposal = optimizer.ProposalFunc[recipe, fixtures.Calculation, int]{
		Identity: "reject-v1",
		Generate: func(context.Context, optimizer.ProposalRequest[fixtures.Calculation, int]) (optimizer.ProposalResult[recipe], error) {
			return optimizer.ProposalResult[recipe]{
				Candidates: []optimizer.Proposal[recipe]{{ID: "rejected"}},
				Exhausted:  true,
				Usage:      evaly.Usage{Known: true},
			}, nil
		},
	}
	// Act.
	result, err := optimizer.Search(context.Background(), c)
	_, restoreErr := optimizer.RestoreResult(result, c.Codec)
	// Assert: rejection is a valid diagnostic attempt, never an accepted candidate.
	if err != nil || restoreErr != nil || len(result.History) != 1 || result.History[0].State != "invalid" ||
		result.History[0].Reason != "candidate_encoding" ||
		len(result.RoundHistory[0].Candidates) != 0 {
		t.Fatal(err, restoreErr, result)
	}
}

func TestProposalUsageOverflowReturnsError(t *testing.T) {
	// Arrange: finite receipts exceed the representable aggregate across two rounds.
	c := searchConfig(t, 30)
	c.MaximumRounds = 2
	c.Proposal = optimizer.ProposalFunc[recipe, fixtures.Calculation, int]{
		Identity: "overflow-v1",
		Generate: func(_ context.Context, r optimizer.ProposalRequest[fixtures.Calculation, int]) (optimizer.ProposalResult[recipe], error) {
			return optimizer.ProposalResult[recipe]{
				Candidates: []optimizer.Proposal[recipe]{{ID: strings.Repeat("c", r.Round+1)}},
				Exhausted:  r.Round == 1,
				Usage:      evaly.Usage{Known: true, Units: math.MaxFloat64},
			}, nil
		},
	}
	// The budget host deliberately records receipts without a finite global capacity.
	c.Budget = unlimitedReceiptBudget{}
	// Act.
	_, err := optimizer.Search(context.Background(), c)
	// Assert: no nil-error artifact claims an unrepresentable sum.
	if !errors.Is(err, evaly.ErrInvalid) {
		t.Fatal(err)
	}
}

type unlimitedReceiptBudget struct{}

func (unlimitedReceiptBudget) Reserve(_ context.Context, id string, units float64) (evaly.Reservation, error) {
	return evaly.Reservation{ID: id, Units: units}, nil
}
func (unlimitedReceiptBudget) Claim(context.Context, evaly.Reservation) error { return nil }
func (unlimitedReceiptBudget) Reconcile(context.Context, evaly.Reservation, evaly.Usage) error {
	return nil
}
func (unlimitedReceiptBudget) Release(context.Context, evaly.Reservation) error { return nil }

func TestBoundPartialMeasurementsRemainRestorable(t *testing.T) {
	for _, phase := range []string{"calibration", "holdout"} {
		t.Run(phase, func(t *testing.T) {
			// Arrange: the measurement has its own cancelled execution, not a cancelled search.
			c := searchConfig(t, 30)
			evaluate := c.Evaluate
			c.Evaluate = func(ctx context.Context, req optimizer.EvaluationRequest[recipe, fixtures.Calculation, int]) (evaly.Experiment, error) {
				if req.Phase != phase {
					return evaluate(ctx, req)
				}
				cancelled, stop := context.WithCancel(ctx)
				stop()
				experiment, err := evaluate(cancelled, req)
				if err != nil {
					return experiment, err
				}
				return experiment, errors.New("partial host execution")
			}
			// Act.
			result, err := optimizer.Search(context.Background(), c)
			_, restoreErr := optimizer.RestoreResult(result, c.Codec)
			// Assert: a bound incomplete experiment is valid diagnostic evidence.
			if err != nil || restoreErr != nil {
				t.Fatal(err, restoreErr, result)
			}
			record := result.Holdout
			if phase == "calibration" {
				record = result.History[0].Experiment
			}
			if record == nil || record.Manifest.State != "incomplete" {
				t.Fatal(result)
			}
		})
	}
}

type failingEvaluationSettlement struct{ evaly.Budget }

func (b failingEvaluationSettlement) Reconcile(ctx context.Context, r evaly.Reservation, u evaly.Usage) error {
	if err := b.Budget.Reconcile(ctx, r, u); err != nil {
		return err
	}
	if strings.Contains(r.ID, "/evaluation/") {
		return errors.New("settlement host failure")
	}
	return nil
}

func TestCalibrationStopReasonPrecedence(t *testing.T) {
	for _, cancelSearch := range []bool{false, true} {
		t.Run(map[bool]string{false: "settlement", true: "cancellation"}[cancelSearch], func(t *testing.T) {
			// Arrange: one dispatch has a foreign measurement, a settlement failure and optional cancellation.
			c := searchConfig(t, 30)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			c.Budget = failingEvaluationSettlement{c.Budget}
			c.Evaluate = func(context.Context, optimizer.EvaluationRequest[recipe, fixtures.Calculation, int]) (evaly.Experiment, error) {
				if cancelSearch {
					cancel()
				}
				return c.CalibrationBaseline, errors.New("operational failure")
			}
			// Act.
			result, err := optimizer.Search(ctx, c)
			_, restoreErr := optimizer.RestoreResult(result, c.Codec)
			// Assert: later settlement then cancellation take precedence over protocol failure.
			want := "evaluation_usage_failure"
			if cancelSearch {
				want = "deadline"
			}
			if err != nil || restoreErr != nil || result.Reason != want || len(result.History) != 1 ||
				result.History[0].Experiment != nil {
				t.Fatal(err, restoreErr, result)
			}
		})
	}
}
