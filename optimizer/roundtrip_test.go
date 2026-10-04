package optimizer_test

import (
	"context"
	"strings"
	"testing"

	"github.com/skosovsky/evaly"
	"github.com/skosovsky/evaly/optimizer"
)

func TestSearchCanonicalRoundTrip(t *testing.T) {
	// Arrange: use ordinary experiments and the reference static proposer.
	config := searchConfig(t, 30)
	// Act: restore an actual completed history, including comparisons and holdout.
	result, err := optimizer.Search(context.Background(), config)
	if err != nil {
		t.Fatal(err)
	}
	restored, restoreErr := optimizer.RestoreResult(result, config.Codec)
	// Assert: honest producer output must satisfy its published restore contract.
	if restoreErr != nil || optimizer.ValidateResult(result) != nil || restored.Revision != result.Revision ||
		len(restored.History) != len(result.History) ||
		len(restored.RoundHistory) != len(result.RoundHistory) {
		t.Fatal(restoreErr, result)
	}
	if len(restored.History) == 0 || restored.History[0].Comparison == nil {
		t.Fatal("fixture did not exercise persisted comparisons", result)
	}
	// Arrange / Act: mutate the restored service record, leaving the source intact.
	restored.History[0].Candidate.Description[0] = 'x'
	restored.History[0].Comparison.Reasons = append(restored.History[0].Comparison.Reasons, "mutated")
	restored.RoundHistory[0].Usage.Units = 100
	// Assert: restore returns a detached value, never shared persisted history.
	if optimizer.ValidateResult(result) != nil {
		t.Fatal("restored mutation changed source history")
	}
}

func TestSearchFullLengthIDFitsGeneratedArtifacts(t *testing.T) {
	// Arrange: a valid search ID at the core artifact boundary.
	c := searchConfig(t, 30)
	c.ID = strings.Repeat("s", 128)
	// Act: derive calibration and holdout IDs without truncating identity inputs.
	result, err := optimizer.Search(context.Background(), c)
	// Assert: every generated artifact remains publishable and restorable.
	if err != nil || result.Winner == "" || result.Holdout == nil || optimizer.ValidateResult(result) != nil {
		t.Fatal(err, result)
	}
	for _, entry := range result.History {
		if entry.Experiment != nil {
			if _, err := evaly.NewEnvelope("experiment", entry.Experiment.Manifest.ID, *entry.Experiment); err != nil {
				t.Fatal(err)
			}
		}
	}
}
