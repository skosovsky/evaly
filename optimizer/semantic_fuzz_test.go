package optimizer_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/skosovsky/evaly/optimizer"
)

func FuzzResultRejectsRehashedAuditMutation(f *testing.F) {
	// Arrange: one immutable honest artifact; each iteration owns a fresh copy.
	c := searchConfig(f, 30)
	result, err := optimizer.Search(context.Background(), c)
	if err != nil {
		f.Fatal(err)
	}
	wire, err := json.Marshal(result)
	if err != nil {
		f.Fatal(err)
	}
	for kind := range byte(6) {
		f.Add(kind, uint16(1))
	}
	f.Fuzz(func(t *testing.T, kind byte, offset uint16) {
		var mutated optimizer.Result
		if err := json.Unmarshal(wire, &mutated); err != nil {
			t.Fatal(err)
		}
		switch kind % 6 {
		case 0:
			mutated.ProposalUsage.Units = float64(offset) + 1
		case 1:
			mutated.History[0].DispatchID = "foreign"
		case 2:
			mutated.History[0].Round = len(mutated.RoundHistory) + int(offset)
		case 3:
			mutated.RoundHistory[0].Received = mutated.RoundHistory[0].Received[:0]
		case 4:
			mutated.States = nil
		case 5:
			mutated.Holdout = nil
			mutated.HoldoutComparison = nil
		}
		mutated = resealSearch(t, mutated)
		// Act.
		validationErr := optimizer.ValidateResult(mutated)
		_, restoreErr := optimizer.RestoreResult(mutated, c.Codec)
		// Assert: valid checksums never authorize contradictory metadata.
		if validationErr == nil || restoreErr == nil {
			t.Fatal(kind, offset, validationErr, restoreErr)
		}
	})
}
