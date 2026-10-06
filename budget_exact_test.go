package evaly_test

import (
	"errors"
	"math"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/skosovsky/evaly"
)

func TestMemoryBudgetExhaustedTinyReservation(t *testing.T) {
	// Arrange
	ctx := t.Context()
	b, err := evaly.NewMemoryBudget(1)
	if err != nil {
		t.Fatal(err)
	}
	full, err := b.Reserve(ctx, "full", 1)
	if err != nil {
		t.Fatal(err)
	}
	if err = b.Claim(ctx, full); err != nil {
		t.Fatal(err)
	}
	// Act
	_, err = b.Reserve(ctx, "tiny", 1e-17)
	// Assert
	if !errors.Is(err, evaly.ErrBudget) || b.Used() != 1 {
		t.Fatal(err, b.Used())
	}
}

func TestMemoryBudgetLargeReconciliationRetainsActual(t *testing.T) {
	// Arrange
	ctx := t.Context()
	b, err := evaly.NewMemoryBudget(1e16)
	if err != nil {
		t.Fatal(err)
	}
	large, err := b.Reserve(ctx, "large", 1e16)
	if err != nil {
		t.Fatal(err)
	}
	if err = b.Claim(ctx, large); err != nil {
		t.Fatal(err)
	}
	// Act
	err = b.Reconcile(ctx, large, evaly.Usage{Known: true, Units: 1})
	// Assert
	if err != nil || b.Used() != 1 {
		t.Fatal(err, b.Used())
	}
}

func TestMemoryBudgetMixedMagnitudeOrders(t *testing.T) {
	for _, reverse := range []bool{false, true} {
		t.Run(strconv.FormatBool(reverse), func(t *testing.T) {
			checkMixedBudgetOrder(t, reverse)
		})
	}
}

func checkMixedBudgetOrder(t *testing.T, reverse bool) {
	t.Helper()
	// Arrange: all three fit exactly below the next float after the large value.
	ctx := t.Context()
	b, err := evaly.NewMemoryBudget(math.Nextafter(1e16, math.Inf(1)))
	if err != nil {
		t.Fatal(err)
	}
	large, err := b.Reserve(ctx, "large", 1e16)
	if err != nil {
		t.Fatal(err)
	}
	unit, err := b.Reserve(ctx, "unit", 1)
	if err != nil {
		t.Fatal(err)
	}
	tiny, err := b.Reserve(ctx, "tiny", 1e-17)
	if err != nil {
		t.Fatal(err)
	}
	if b.Used() != math.Nextafter(1e16, math.Inf(1)) {
		t.Fatal(b.Used())
	}
	// Act: reverse the release/settlement order; both must preserve tiny.
	if reverse {
		if err = b.Release(ctx, unit); err != nil {
			t.Fatal(err)
		}
	}
	if err = b.Reconcile(ctx, large, evaly.Usage{Known: true, Units: 1}); err != nil {
		t.Fatal(err)
	}
	if !reverse {
		if err = b.Release(ctx, unit); err != nil {
			t.Fatal(err)
		}
	}
	// Assert: reporting rounds up, while admission still uses the exact total.
	if b.Used() != math.Nextafter(1, math.Inf(1)) {
		t.Fatal(b.Used())
	}
	if err = b.Release(ctx, tiny); err != nil || b.Used() != 1 {
		t.Fatal(err, b.Used())
	}
}

func TestMemoryBudgetFiniteExtremes(t *testing.T) {
	for _, capacity := range []float64{0, math.SmallestNonzeroFloat64, math.MaxFloat64} {
		t.Run(strconv.FormatFloat(capacity, 'g', -1, 64), func(t *testing.T) {
			// Arrange
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
			_, denied := b.Reserve(ctx, "extra", math.SmallestNonzeroFloat64)
			actual := math.Min(capacity, math.SmallestNonzeroFloat64)
			err = b.Reconcile(ctx, r, evaly.Usage{Known: true, Units: actual})
			// Assert
			if !errors.Is(denied, evaly.ErrBudget) || err != nil || b.Used() != actual {
				t.Fatal(denied, err, b.Used())
			}
		})
	}
}

func TestMemoryBudgetClaimUnknownAndIdempotency(t *testing.T) {
	// Arrange
	ctx := t.Context()
	b, err := evaly.NewMemoryBudget(2)
	if err != nil {
		t.Fatal(err)
	}
	r, err := b.Reserve(ctx, "one", 1)
	if err != nil {
		t.Fatal(err)
	}
	// Act
	same, retry := b.Reserve(ctx, "one", 1)
	_, changed := b.Reserve(ctx, "one", 2)
	claim := b.Claim(ctx, r)
	duplicate := b.Claim(ctx, r)
	unknown := b.Reconcile(ctx, r, evaly.Usage{Known: false, Units: 0})
	release := b.Release(ctx, r)
	// Assert
	if same != r || retry != nil || !errors.Is(changed, evaly.ErrConflict) || claim != nil ||
		!errors.Is(
			duplicate,
			evaly.ErrConflict,
		) || unknown != nil || !errors.Is(release, evaly.ErrConflict) || b.Used() != 1 {
		t.Fatal(same, retry, changed, claim, duplicate, unknown, release, b.Used())
	}
	if err = b.Reconcile(ctx, r, evaly.Usage{Known: true, Units: .5}); err != nil {
		t.Fatal(err)
	}
	if err = b.Reconcile(ctx, r, evaly.Usage{Known: true, Units: .5}); err != nil || b.Used() != .5 {
		t.Fatal(err, b.Used())
	}
	if err = b.Reconcile(ctx, r, evaly.Usage{Known: true, Units: .25}); !errors.Is(err, evaly.ErrConflict) {
		t.Fatal(err)
	}
	if err = b.Release(ctx, r); !errors.Is(err, evaly.ErrConflict) {
		t.Fatal(err)
	}
}

func TestMemoryBudgetConcurrentClaimAndAccounting(t *testing.T) {
	// Arrange
	const workers = 32
	ctx := t.Context()
	b, err := evaly.NewMemoryBudget(workers)
	if err != nil {
		t.Fatal(err)
	}
	shared, err := b.Reserve(ctx, "shared", 0)
	if err != nil {
		t.Fatal(err)
	}
	var claims atomic.Int64
	var wg sync.WaitGroup
	// Act
	for i := range workers {
		wg.Go(func() {
			if claim := b.Claim(ctx, shared); claim == nil {
				claims.Add(1)
			} else if !errors.Is(claim, evaly.ErrConflict) {
				t.Error(claim)
			}
			r, reserve := b.Reserve(ctx, strconv.Itoa(i), 1)
			if reserve != nil {
				t.Error(reserve)
				return
			}
			if claim := b.Claim(ctx, r); claim != nil {
				t.Error(claim)
				return
			}
			if settle := b.Reconcile(ctx, r, evaly.Usage{Known: true, Units: .5}); settle != nil {
				t.Error(settle)
			}
		})
	}
	wg.Wait()
	// Assert
	if claims.Load() != 1 || b.Used() != workers/2 {
		t.Fatal(claims.Load(), b.Used())
	}
}

func TestMemoryBudgetAdmissionIgnoresRoundedDisplay(t *testing.T) {
	// Arrange: the rounded display reaches capacity while exact slack remains.
	ctx := t.Context()
	capacity := math.Nextafter(1, math.Inf(1))
	b, err := evaly.NewMemoryBudget(capacity)
	if err != nil {
		t.Fatal(err)
	}
	large, err := b.Reserve(ctx, "large", 1)
	if err != nil {
		t.Fatal(err)
	}
	tiny, err := b.Reserve(ctx, "tiny", 1e-17)
	if err != nil {
		t.Fatal(err)
	}
	if b.Used() != capacity {
		t.Fatal(b.Used())
	}
	// Act
	another, admitted := b.Reserve(ctx, "another", 1e-17)
	_, denied := b.Reserve(ctx, "too-much", capacity-1)
	released := b.Release(ctx, large)
	// Assert: both small liabilities remain after removing the large reservation.
	if admitted != nil || !errors.Is(denied, evaly.ErrBudget) || released != nil || b.Used() != 2e-17 {
		t.Fatal(admitted, denied, released, b.Used())
	}
	if err = b.Release(ctx, tiny); err != nil || b.Used() != 1e-17 {
		t.Fatal(err, b.Used())
	}
	if err = b.Release(ctx, another); err != nil || b.Used() != 0 {
		t.Fatal(err, b.Used())
	}
	if err = b.Release(ctx, another); err != nil {
		t.Fatal(err)
	}
	if _, err = b.Reserve(ctx, "another", 1e-17); !errors.Is(err, evaly.ErrConflict) {
		t.Fatal(err)
	}
}

func TestMemoryBudgetNearestUpwardReporting(t *testing.T) {
	// Arrange: nearest float is already above the exact sum.
	ctx := t.Context()
	b, err := evaly.NewMemoryBudget(2)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = b.Reserve(ctx, "whole", 1); err != nil {
		t.Fatal(err)
	}
	fraction := math.Ldexp(3, -54)
	// Act
	_, err = b.Reserve(ctx, "fraction", fraction)
	// Assert: do not advance an already conservative nearest float twice.
	if err != nil || b.Used() != math.Nextafter(1, math.Inf(1)) {
		t.Fatal(err, b.Used())
	}
}
