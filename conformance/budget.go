package conformance

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/skosovsky/evaly"
)

// Budget requires fresh independently namespaced atomic adapters with exactly two
// units capacity. Factories own external namespace isolation and disposal.
func Budget(t *testing.T, factory func() (evaly.Budget, error)) {
	t.Helper()
	checks := map[string]func(*testing.T, evaly.Budget){
		"capacity_unknown": budgetCapacityUnknown,
		"reserve_identity": budgetReserveIdentity,
		"claim_once":       budgetClaimOnce,
		"concurrent_claim": budgetConcurrentClaim,
		"release":          budgetRelease,
		"settlement":       budgetSettlement,
		"cancelled":        budgetCancelled,
	}
	for name, check := range checks {
		t.Run(name, func(t *testing.T) {
			// Arrange: each check owns a new host namespace via the fixture factory.
			b, err := factory()
			if err != nil {
				t.Fatal("budget factory", err)
			}
			if err = evaly.ValidatePort(b); err != nil {
				t.Fatal("invalid budget", err)
			}
			// Act / Assert: evaluate one independent protocol transition family.
			check(t, b)
		})
	}
}

func reserveBudget(t *testing.T, b evaly.Budget, id string, units float64) evaly.Reservation {
	t.Helper()
	reservation, err := b.Reserve(context.Background(), id, units)
	if err != nil {
		t.Fatal("reserve", id, err)
	}
	if reservation.ID != id || reservation.Units != units {
		t.Fatal("changed reservation", reservation)
	}
	return reservation
}

func budgetCapacityUnknown(t *testing.T, b evaly.Budget) {
	t.Helper()
	// Arrange: three concurrent reservations compete for two units.
	var accepted atomic.Int32
	var wg sync.WaitGroup
	for _, id := range []string{"one", "two", "three"} {
		wg.Go(func() {
			// Act.
			reservation, err := b.Reserve(context.Background(), id, 1)
			if err != nil {
				if !errors.Is(err, evaly.ErrBudget) {
					t.Error("reserve", err)
				}
				return
			}
			accepted.Add(1)
			if err = b.Claim(context.Background(), reservation); err != nil {
				t.Error("claim", err)
				return
			}
			if err = b.Reconcile(context.Background(), reservation, evaly.Usage{Known: false, Units: 0}); err != nil {
				t.Error("unknown reconcile", err)
			}
		})
	}
	wg.Wait()
	// Assert: claimed unknown receipts retain the complete liability.
	if accepted.Load() != 2 {
		t.Fatal("reservation oversubscription", accepted.Load())
	}
	if _, err := b.Reserve(context.Background(), "four", 1); !errors.Is(err, evaly.ErrBudget) {
		t.Fatal("unknown usage released", err)
	}
}

func budgetReserveIdentity(t *testing.T, b evaly.Budget) {
	t.Helper()
	// Arrange.
	first := reserveBudget(t, b, "same", 1)
	// Act / Assert: identical lookup is repeatable but incompatible units conflict.
	second := reserveBudget(t, b, "same", 1)
	if second != first {
		t.Fatal("non-idempotent reservation", first, second)
	}
	if _, err := b.Reserve(context.Background(), "same", 2); !errors.Is(err, evaly.ErrConflict) {
		t.Fatal("changed units accepted", err)
	}
	reserveBudget(t, b, "other", 1)
	if _, err := b.Reserve(context.Background(), "overflow", 1); !errors.Is(err, evaly.ErrBudget) {
		t.Fatal("idempotent reserve consumed capacity", err)
	}
}

func budgetClaimOnce(t *testing.T, b evaly.Budget) {
	t.Helper()
	// Arrange: zero units still carry dispatch authorization.
	reservation := reserveBudget(t, b, "free", 0)
	// Act / Assert.
	if err := b.Claim(context.Background(), reservation); err != nil {
		t.Fatal("first claim", err)
	}
	if err := b.Claim(context.Background(), reservation); !errors.Is(err, evaly.ErrConflict) {
		t.Fatal("duplicate claim authorized", err)
	}
	lookedUp := reserveBudget(t, b, "free", 0)
	if err := b.Claim(context.Background(), lookedUp); !errors.Is(err, evaly.ErrConflict) {
		t.Fatal("lookup authorized redispatch", err)
	}
}

func budgetConcurrentClaim(t *testing.T, b evaly.Budget) {
	t.Helper()
	// Arrange: one identity is claimed concurrently by independent workers.
	reservation := reserveBudget(t, b, "shared", 1)
	var accepted atomic.Int32
	var wg sync.WaitGroup
	for range concurrentBudgetClaims {
		wg.Go(func() {
			// Act.
			err := b.Claim(context.Background(), reservation)
			if err == nil {
				accepted.Add(1)
			} else if !errors.Is(err, evaly.ErrConflict) {
				t.Error("claim classification", err)
			}
		})
	}
	wg.Wait()
	// Assert.
	if accepted.Load() != 1 {
		t.Fatal("claim must authorize exactly once", accepted.Load())
	}
}

func budgetRelease(t *testing.T, b evaly.Budget) {
	t.Helper()
	// Arrange: all capacity is reserved without dispatch.
	reservation := reserveBudget(t, b, "released", 2)
	// Act / Assert: release is repeatable and makes capacity available.
	for range 2 {
		if err := b.Release(context.Background(), reservation); err != nil {
			t.Fatal("release", err)
		}
	}
	if _, err := b.Reserve(context.Background(), "released", 2); !errors.Is(err, evaly.ErrConflict) {
		t.Fatal("released identity reused", err)
	}
	claimed := reserveBudget(t, b, "claimed", 2)
	if err := b.Claim(context.Background(), claimed); err != nil {
		t.Fatal("claim", err)
	}
	if err := b.Release(context.Background(), claimed); !errors.Is(err, evaly.ErrConflict) {
		t.Fatal("claimed release refunded", err)
	}
	if _, err := b.Reserve(context.Background(), "blocked", 1); !errors.Is(err, evaly.ErrBudget) {
		t.Fatal("claimed liability lost", err)
	}
}

func budgetSettlement(t *testing.T, b evaly.Budget) {
	t.Helper()
	// Arrange.
	reservation := reserveBudget(t, b, "settled", 2)
	if err := b.Claim(context.Background(), reservation); err != nil {
		t.Fatal("claim", err)
	}
	// Act / Assert: unknown then identical known settlement; incompatible changes reject.
	if err := b.Reconcile(context.Background(), reservation, evaly.Usage{Known: false, Units: 0}); err != nil {
		t.Fatal(err)
	}
	receipt := evaly.Usage{Known: true, Units: 1}
	for range 2 {
		if err := b.Reconcile(context.Background(), reservation, receipt); err != nil {
			t.Fatal("known settlement", err)
		}
	}
	if err := b.Reconcile(
		context.Background(),
		reservation,
		evaly.Usage{Known: true, Units: 0},
	); !errors.Is(
		err,
		evaly.ErrConflict,
	) {
		t.Fatal("settlement changed", err)
	}
	if err := b.Release(context.Background(), reservation); !errors.Is(err, evaly.ErrConflict) {
		t.Fatal("settled release", err)
	}
	reserveBudget(t, b, "remaining", 1)
	if _, err := b.Reserve(context.Background(), "overflow", 1); !errors.Is(err, evaly.ErrBudget) {
		t.Fatal("known usage lost", err)
	}
}

func budgetCancelled(t *testing.T, b evaly.Budget) {
	t.Helper()
	// Arrange: contexts cancelled before any method invocation.
	reservation := reserveBudget(t, b, "cancel", 1)
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	// Act / Assert: each port operation classifies pre-cancellation.
	if _, err := b.Reserve(cancelled, "no-dispatch", 0); !errors.Is(err, context.Canceled) {
		t.Fatal("reserve cancellation", err)
	}
	for name, operation := range map[string]func() error{
		"claim":     func() error { return b.Claim(cancelled, reservation) },
		"release":   func() error { return b.Release(cancelled, reservation) },
		"reconcile": func() error { return b.Reconcile(cancelled, reservation, evaly.Usage{Known: true, Units: 0}) },
	} {
		if err := operation(); !errors.Is(err, context.Canceled) {
			t.Fatal(name, "cancellation", err)
		}
	}
	if err := b.Claim(context.Background(), reservation); err != nil {
		t.Fatal("cancelled operation changed authorization", err)
	}
}
