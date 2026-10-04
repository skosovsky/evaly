package optimizer_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/skosovsky/evaly"
	"github.com/skosovsky/evaly/conformance"
	"github.com/skosovsky/evaly/internal/fixtures"
	"github.com/skosovsky/evaly/optimizer"
)

func TestHostSplitKeysRejectRelatedCasesWithDifferentIDs(t *testing.T) {
	// Arrange: technical IDs differ, but the host declares one shared group.
	c := searchConfig(t, 20)
	validator := optimizer.KeySplitValidator[fixtures.Calculation, int]{
		Identity: "session-v1",
		GroupKey: func(evaly.Case[fixtures.Calculation, int]) (string, error) { return "same-session", nil },
	}
	c.SplitValidator = validator
	calls := 0
	c.Evaluate = func(context.Context, optimizer.EvaluationRequest[recipe, fixtures.Calculation, int]) (evaly.Experiment, error) {
		calls++
		return evaly.Experiment{}, nil
	}
	// Act.
	_, err := optimizer.Search(context.Background(), c)
	// Assert.
	if !errors.Is(err, evaly.ErrConflict) || calls != 0 {
		t.Fatal(err, calls)
	}
}
func TestSplitValidatorRequiresKeysAndHonorsCancellation(t *testing.T) {
	// Arrange.
	c := searchConfig(t, 20)
	empty := optimizer.KeySplitValidator[fixtures.Calculation, int]{Identity: "empty"}
	valid := optimizer.KeySplitValidator[fixtures.Calculation, int]{
		Identity:   "content-v1",
		ContentKey: func(c evaly.Case[fixtures.Calculation, int]) (string, error) { return c.ID, nil },
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	// Act.
	absent := empty.Validate()
	cancelled := valid.ValidateSplit(ctx, c.Split)
	accepted := valid.ValidateSplit(context.Background(), c.Split)
	// Assert.
	if !errors.Is(absent, evaly.ErrInvalid) || !errors.Is(cancelled, context.Canceled) || accepted != nil {
		t.Fatal(absent, cancelled, accepted)
	}
}

func TestNoHostValidatorDoesNotClaimIndependentSplit(t *testing.T) {
	// Arrange.
	c := searchConfig(t, 20)
	// Act.
	result, err := optimizer.Search(context.Background(), c)
	// Assert: different IDs alone establish no content independence.
	if err != nil || result.IndependenceValidated || result.SplitValidationRevision != "" {
		t.Fatal(result, err)
	}
}

func TestKeySplitValidatorConformance(t *testing.T) {
	// Arrange.
	c := searchConfig(t, 20)
	validator := optimizer.KeySplitValidator[fixtures.Calculation, int]{
		Identity: "content-v1",
		ContentKey: func(c evaly.Case[fixtures.Calculation, int]) (string, error) {
			return fmt.Sprintf("%d+%d", c.Input.Left, c.Input.Right), nil
		},
	}
	duplicate := c.Split
	duplicate.Holdout = duplicate.Calibration
	// Act and Assert.
	conformance.SplitValidation(t, validator, c.Split, duplicate)
}
