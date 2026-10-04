package optimizer_test

import (
	"context"
	"errors"
	"math"
	"testing"

	"github.com/skosovsky/evaly"
	"github.com/skosovsky/evaly/internal/fixtures"
	"github.com/skosovsky/evaly/optimizer"
)

func TestSearchPreflightPreventsProposalAndEvaluation(t *testing.T) {
	for _, variant := range []string{"codec", "gate", "usage", "baseline", "proposal_callback", "nil_budget", "nil_ledger"} {
		t.Run(variant, func(t *testing.T) {
			// Arrange: invalid structural configuration must fail before paid callbacks.
			c := searchConfig(t, 20)
			calls := 0
			c.Proposal = optimizer.ProposalFunc[recipe, fixtures.Calculation, int]{
				Identity: "proposal-v1",
				Generate: func(context.Context, optimizer.ProposalRequest[fixtures.Calculation, int]) (optimizer.ProposalResult[recipe], error) {
					calls++
					return optimizer.ProposalResult[recipe]{}, nil
				},
			}
			c.Evaluate = func(context.Context, optimizer.EvaluationRequest[recipe, fixtures.Calculation, int]) (evaly.Experiment, error) {
				calls++
				return evaly.Experiment{}, nil
			}
			switch variant {
			case "nil_budget":
				var budget *evaly.MemoryBudget
				c.Budget = budget
			case "nil_ledger":
				var ledger *optimizer.MemoryLedger
				c.Ledger = ledger
			case "codec":
				c.Codec = evaly.JSONCodec[recipe]{}
			case "gate":
				c.Gate.Revision = ""
			case "usage":
				c.ProposalUnits = math.NaN()
			case "proposal_callback":
				c.Proposal = optimizer.ProposalFunc[recipe, fixtures.Calculation, int]{Identity: "empty-proposal"}
			case "baseline":
				c.CalibrationBaseline = evaly.Experiment{}
			}
			// Act.
			_, err := optimizer.Search(context.Background(), c)
			// Assert.
			if err == nil || calls != 0 {
				t.Fatalf("error=%v paid calls=%d", err, calls)
			}
		})
	}
}

func TestSearchRetainsProposalUsageOnFailure(t *testing.T) {
	// Arrange: a failed paid callback can still report known cost.
	c := searchConfig(t, 20)
	c.ProposalUnits = 3
	c.Proposal = optimizer.ProposalFunc[recipe, fixtures.Calculation, int]{
		Identity: "paid-proposal-v1",
		Generate: func(context.Context, optimizer.ProposalRequest[fixtures.Calculation, int]) (optimizer.ProposalResult[recipe], error) {
			return optimizer.ProposalResult[recipe]{
				Usage: evaly.Usage{Known: true, Units: 2},
			}, errors.New(
				"provider error",
			)
		},
	}
	// Act.
	r, err := optimizer.Search(context.Background(), c)
	// Assert: both history and actual accounting survive failure.
	if err != nil || r.State != "stopped" || r.ProposalUsage != (evaly.Usage{Known: true, Units: 2}) ||
		c.Budget.(*evaly.MemoryBudget).Used() != 2 {
		t.Fatal(r, err)
	}
}

func TestOptimizerDoesNotDispatchAfterBudgetStop(t *testing.T) {
	// Arrange: first candidate fits; second exhausts budget.
	c := searchConfig(t, 1)
	original := c.Evaluate
	holdoutCalls := 0
	c.Evaluate = func(ctx context.Context, req optimizer.EvaluationRequest[recipe, fixtures.Calculation, int]) (evaly.Experiment, error) {
		if req.Dataset.Revision() == c.Split.Holdout.Revision() {
			holdoutCalls++
		}
		return original(ctx, req)
	}
	// Act.
	r, err := optimizer.Search(context.Background(), c)
	// Assert: selecting/ranking the completed work is allowed, dispatch after stop is not.
	if err != nil {
		t.Fatal(err)
	}
	if holdoutCalls != 0 {
		t.Fatalf("holdout dispatched %d times after exhaustion: state=%s reason=%s", holdoutCalls, r.State, r.Reason)
	}
}
