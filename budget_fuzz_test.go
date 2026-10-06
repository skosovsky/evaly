package evaly_test

import (
	"errors"
	"math"
	"testing"

	"github.com/skosovsky/evaly"
)

func FuzzMemoryBudgetSettlement(f *testing.F) {
	f.Add(math.Float64bits(1e16), math.Float64bits(1))
	f.Add(math.Float64bits(math.MaxFloat64), math.Float64bits(math.SmallestNonzeroFloat64))
	f.Add(math.Float64bits(1), math.Float64bits(.5))
	f.Fuzz(func(t *testing.T, capacityBits, actualBits uint64) {
		// Arrange: arbitrary finite bounds and exactly representable receipt.
		capacity, actual := math.Float64frombits(capacityBits), math.Float64frombits(actualBits)
		if math.IsNaN(capacity) || math.IsInf(capacity, 0) || capacity < 0 ||
			math.IsNaN(actual) || math.IsInf(actual, 0) || actual < 0 || actual > capacity {
			t.Skip()
		}
		ctx := t.Context()
		b, err := evaly.NewMemoryBudget(capacity)
		if err != nil {
			t.Fatal(err)
		}
		r, err := b.Reserve(ctx, "full", capacity)
		if err != nil {
			t.Fatal(err)
		}
		// Act
		_, extra := b.Reserve(ctx, "extra", math.SmallestNonzeroFloat64)
		err = b.Reconcile(ctx, r, evaly.Usage{Known: true, Units: actual})
		// Assert: full capacity never admits a positive extra; actual survives.
		if !errors.Is(extra, evaly.ErrBudget) || err != nil || b.Used() != actual {
			t.Fatal(extra, err, b.Used())
		}
	})
}
