package optimizer_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/skosovsky/evaly/optimizer"
)

func TestRehashedStopPhaseContradictionsRejected(t *testing.T) {
	// Arrange: a real selected winner with its bound holdout.
	c := searchConfig(t, 30)
	result, err := optimizer.Search(context.Background(), c)
	if err != nil || result.HoldoutComparison == nil {
		t.Fatal(err, result)
	}
	wire, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	for _, reason := range []string{"candidate_limit", "round_limit", "proposal_failure", "holdout_ledger_failure", "holdout_budget", "holdout_evaluation_failure", "holdout_comparison_failure", "deadline"} {
		t.Run(reason, func(t *testing.T) {
			// Arrange: retain measurements while inventing an incompatible stop phase.
			var mutated optimizer.Result
			if err := json.Unmarshal(wire, &mutated); err != nil {
				t.Fatal(err)
			}
			mutated.State = "stopped"
			mutated.States = []string{"proposing", "selecting", "stopped"}
			mutated.Reason = reason
			mutated = resealSearch(t, mutated)
			// Act.
			validationErr := optimizer.ValidateResult(mutated)
			_, restoreErr := optimizer.RestoreResult(mutated, c.Codec)
			// Assert: checksum cannot make an unreachable state/artifact relationship valid.
			if validationErr == nil || restoreErr == nil {
				t.Fatal(reason, validationErr, restoreErr)
			}
		})
	}
}
