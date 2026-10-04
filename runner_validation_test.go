package evaly_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"testing"

	"github.com/skosovsky/evaly"
)

func TestRepeatedRunIdentityCannotReuseSpentBudget(t *testing.T) {
	// Arrange: external caller retries a run with an unchanged identity.
	c := config(t, 1)
	budget, _ := evaly.NewMemoryBudget(1)
	c.Budget = budget
	calls := 0
	original := c.Target
	c.Target = evaly.TargetFunc[input, int, *int](
		func(ctx context.Context, i input, tc evaly.TrialContext[*int]) (evaly.TargetResult[int], error) {
			calls++
			return original.Run(ctx, i, tc)
		},
	)
	// Act.
	if _, err := evaly.Run(context.Background(), c); err != nil {
		t.Fatal(err)
	}
	_, _ = evaly.Run(context.Background(), c)
	// Assert: retrying reservation identity must not authorize another paid target dispatch.
	if calls > 1 {
		t.Fatalf("duplicate run charged once: calls=%d budget=%v", calls, budget.Used())
	}
}

func TestRestoreRejectsMalformedTrialAndManifest(t *testing.T) {
	// Arrange: a checksum-valid artifact still must satisfy the typed lifecycle/usage contract.
	c := config(t, 1)
	exp, err := evaly.Run(context.Background(), c)
	if err != nil {
		t.Fatal(err)
	}
	for _, variant := range []string{"usage", "cleanup", "provenance", "states"} {
		t.Run(variant, func(t *testing.T) {
			r := exp.Record()
			switch variant {
			case "usage":
				r.Trials[0].TargetUsage.Units = -1
			case "cleanup":
				r.Trials[0].Cleanup.State = "alien_success"
			case "provenance":
				r.Manifest.Provenance.Target = ""
			case "states":
				r.Trials[0].States = []string{"running", "queued"}
			}
			r.Manifest.Revision = ""
			wire, _ := json.Marshal(r)
			canonical, _ := evaly.CanonicalJSON(wire)
			hash := sha256.Sum256(canonical)
			r.Manifest.Revision = hex.EncodeToString(hash[:])
			// Act.
			_, err := evaly.RestoreExperiment(r)
			// Assert.
			if err == nil {
				t.Fatalf("malformed %s accepted as a sealed experiment", variant)
			}
		})
	}
}
