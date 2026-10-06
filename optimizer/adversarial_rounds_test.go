package optimizer_test

import (
	"context"
	"testing"

	"github.com/skosovsky/evaly"
	"github.com/skosovsky/evaly/internal/fixtures"
	"github.com/skosovsky/evaly/optimizer"
)

type cancellingClaim struct {
	evaly.Budget

	cancel context.CancelFunc
}

func (b cancellingClaim) Claim(ctx context.Context, r evaly.Reservation) error {
	if err := b.Budget.Claim(ctx, r); err != nil {
		return err
	}
	b.cancel()
	return nil
}

func TestSearchDoesNotDispatchAfterClaimCancellation(t *testing.T) {
	// Arrange: the budget callback expires the search after authorizing identity.
	c := searchConfig(t, 30)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c.Budget = cancellingClaim{Budget: c.Budget, cancel: cancel}
	calls := 0
	c.Proposal = optimizer.ProposalFunc[recipe, fixtures.Calculation, int]{
		Identity: "cancel-proposal-v1",
		Generate: func(context.Context, optimizer.ProposalRequest[fixtures.Calculation, int]) (optimizer.ProposalResult[recipe], error) {
			calls++
			return optimizer.ProposalResult[recipe]{}, nil
		},
	}
	// Act.
	result, err := optimizer.Search(ctx, c)
	// Assert: no callback dispatch and no holdout after the stop boundary.
	if err != nil || calls != 0 || result.State != "stopped" || len(result.History) != 0 || result.Holdout != nil ||
		optimizer.ValidateResult(result) != nil {
		t.Fatal(err, calls, result)
	}
}

func TestRejectedDuplicateCandidateHistoryRestores(t *testing.T) {
	// Arrange: one legitimate candidate and its rejected duplicate share revision.
	c := searchConfig(t, 30)
	c.Proposal = optimizer.ProposalFunc[recipe, fixtures.Calculation, int]{
		Identity: "duplicate-proposal-v1",
		Generate: func(context.Context, optimizer.ProposalRequest[fixtures.Calculation, int]) (optimizer.ProposalResult[recipe], error) {
			return optimizer.ProposalResult[recipe]{
				Candidates: []optimizer.Proposal[recipe]{{ID: "same", Value: recipe{}}, {ID: "same", Value: recipe{}}},
				Usage:      evaly.Usage{Known: true},
				Exhausted:  true,
			}, nil
		},
	}
	// Act.
	result, err := optimizer.Search(context.Background(), c)
	restored, restoreErr := optimizer.RestoreResult(result, c.Codec)
	// Assert: retained rejection does not corrupt an otherwise valid history.
	if err != nil || restoreErr != nil || len(restored.History) != 2 || restored.History[1].State != "invalid" ||
		restored.History[1].Reason != "identity_conflict" ||
		len(restored.Ranking) != 1 {
		t.Fatal(err, restoreErr, result)
	}
}

func TestSearchCallbacksCannotMutateHistoryOrRanking(t *testing.T) {
	// Arrange: malicious callbacks modify the service views they receive.
	c := searchConfig(t, 30)
	c.Constraints = optimizer.ConstraintsFunc[recipe]{
		Identity: "mutation-constraints-v1",
		Assess: func(_ context.Context, _ recipe, s optimizer.EvaluationSummary) (optimizer.Feasibility, error) {
			if s.Quality != nil {
				*s.Quality = -100
			}
			if len(s.Measurement.Excluded) > 0 {
				s.Measurement.Excluded[0].Reason = "mutated"
			}
			return optimizer.Feasibility{Feasible: true}, nil
		},
	}
	c.FeedbackProjector = optimizer.FeedbackProjectionFunc{
		Identity: "mutation-projection-v1",
		ProjectFeedback: func(_ context.Context, e optimizer.Evaluation) ([]string, error) {
			if e.Quality != nil {
				*e.Quality = -200
			}
			if e.Comparison != nil {
				e.Comparison.Verdict = evaly.GateInvalid
			}
			if e.Experiment != nil {
				e.Experiment.Manifest.State = "incomplete"
			}
			return nil, nil
		},
	}
	// Act.
	result, err := optimizer.Search(context.Background(), c)
	// Assert: measured score, sealed experiment and winner survive callback mutation.
	if err != nil || len(result.History) == 0 || result.History[0].Quality == nil || *result.History[0].Quality != 1 ||
		result.History[0].Experiment.Manifest.State != "sealed" ||
		result.History[0].Comparison.Verdict != evaly.GatePass ||
		result.Winner == "" ||
		optimizer.ValidateResult(result) != nil {
		t.Fatal(err, result)
	}
}

func TestSearchLimitsForbidFurtherDispatch(t *testing.T) {
	for _, limit := range []string{"round", "candidate"} {
		t.Run(limit, func(t *testing.T) {
			// Arrange: the proposer remains willing to continue after one evaluation.
			c := searchConfig(t, 30)
			if limit == "candidate" {
				c.MaximumCandidates = 1
				c.MaximumRounds = 2
			}
			proposals, evaluations := 0, 0
			c.Proposal = optimizer.ProposalFunc[recipe, fixtures.Calculation, int]{
				Identity: "never-exhausted-v1",
				Generate: func(context.Context, optimizer.ProposalRequest[fixtures.Calculation, int]) (optimizer.ProposalResult[recipe], error) {
					proposals++
					return optimizer.ProposalResult[recipe]{
						Candidates: []optimizer.Proposal[recipe]{{ID: "one", Value: recipe{}}},
						Usage:      evaly.Usage{Known: true},
					}, nil
				},
			}
			evaluate := c.Evaluate
			c.Evaluate = func(ctx context.Context, request optimizer.EvaluationRequest[recipe, fixtures.Calculation, int]) (evaly.Experiment, error) {
				evaluations++
				return evaluate(ctx, request)
			}
			// Act.
			result, err := optimizer.Search(context.Background(), c)
			// Assert: measured partial history survives, but limit stop forbids holdout.
			if err != nil || result.State != "stopped" || result.Reason != limit+"_limit" || proposals != 1 ||
				evaluations != 1 ||
				result.Holdout != nil ||
				len(result.History) != 1 ||
				result.History[0].Quality == nil ||
				optimizer.ValidateResult(result) != nil {
				t.Fatal(err, proposals, evaluations, result)
			}
		})
	}
}
