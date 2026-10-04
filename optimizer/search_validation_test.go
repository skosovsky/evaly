package optimizer_test

import (
	"context"
	"testing"

	"github.com/skosovsky/evaly"
	"github.com/skosovsky/evaly/internal/fixtures"
	"github.com/skosovsky/evaly/optimizer"
)

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
