package evaly

import (
	"context"
	"fmt"
	"math"
	"math/big"
	"sync"
)

type Usage struct {
	Known bool    `json:"known"`
	Units float64 `json:"units"`
}
type Reservation struct {
	ID    string  `json:"id"`
	Units float64 `json:"units"`
}

// Budget is a host atomic reservation port. Unknown usage must retain liability.
type Budget interface {
	Reserve(context.Context, string, float64) (Reservation, error)
	Claim(context.Context, Reservation) error
	Reconcile(context.Context, Reservation, Usage) error
	Release(context.Context, Reservation) error
}
type budgetEntry struct {
	reservation Reservation
	settled     bool
	claimed     bool
	actual      Usage
	released    bool
}

// MemoryBudget is process-local and mutex-atomic; it is not a distributed budget.
// Every finite float64 unit is accounted as its exact binary rational value.
// Admission and settlement never round liabilities; Used rounds upward for display.
type MemoryBudget struct {
	mu       sync.Mutex
	capacity float64
	used     big.Rat
	entries  map[string]budgetEntry
}

func NewMemoryBudget(capacity float64) (*MemoryBudget, error) {
	if !finiteNonnegative(capacity) {
		return nil, ErrInvalid
	}
	var used big.Rat
	return &MemoryBudget{capacity: capacity, entries: map[string]budgetEntry{}, mu: sync.Mutex{}, used: used}, nil
}

// Validate checks constructor invariants without reserving or dispatching work.
func (b *MemoryBudget) Validate() error {
	if b == nil {
		return ErrInvalid
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.entries == nil || !finiteNonnegative(b.capacity) {
		return ErrInvalid
	}
	return nil
}

func finiteNonnegative(v float64) bool { return !math.IsNaN(v) && !math.IsInf(v, 0) && v >= 0 }
func (b *MemoryBudget) Reserve(ctx context.Context, id string, units float64) (Reservation, error) {
	var zeroUsage Usage
	if e := ctx.Err(); e != nil {
		return Reservation{}, e
	}
	if id == "" || !finiteNonnegative(units) {
		return Reservation{}, ErrInvalid
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if e := ctx.Err(); e != nil {
		return Reservation{}, e
	}
	if old, ok := b.entries[id]; ok {
		if old.reservation.Units != units || old.released {
			return Reservation{}, ErrConflict
		}
		return old.reservation, nil
	}
	var requested, total, capacity big.Rat
	requested.SetFloat64(units)
	total.Add(&b.used, &requested)
	capacity.SetFloat64(b.capacity)
	if total.Cmp(&capacity) > 0 {
		return Reservation{}, ErrBudget
	}
	r := Reservation{id, units}
	b.entries[id] = budgetEntry{reservation: r, settled: false, claimed: false, actual: zeroUsage, released: false}
	b.used.Set(&total)
	return r, nil
}
func (b *MemoryBudget) Reconcile(ctx context.Context, r Reservation, u Usage) error {
	if e := ctx.Err(); e != nil {
		return e
	}
	if !finiteNonnegative(u.Units) {
		return ErrInvalid
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	old, ok := b.entries[r.ID]
	if !ok || old.reservation != r || old.released {
		return ErrConflict
	}
	if !u.Known {
		old.claimed = true
		b.entries[r.ID] = old
		return nil
	}
	if u.Units > r.Units {
		return fmt.Errorf("%w: actual usage exceeds reserved bound", ErrInvalid)
	}
	if old.settled {
		if old.actual != u {
			return ErrConflict
		}
		return nil
	}
	var reserved, actual big.Rat
	reserved.SetFloat64(r.Units)
	actual.SetFloat64(u.Units)
	b.used.Sub(&b.used, &reserved)
	b.used.Add(&b.used, &actual)
	old.claimed = true
	old.actual = u
	old.settled = true
	b.entries[r.ID] = old
	return nil
}
func (b *MemoryBudget) Release(ctx context.Context, r Reservation) error {
	if e := ctx.Err(); e != nil {
		return e
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	old, ok := b.entries[r.ID]
	if !ok || old.reservation != r || old.settled || old.claimed {
		return ErrConflict
	}
	if old.released {
		return nil
	}
	var reserved big.Rat
	reserved.SetFloat64(r.Units)
	b.used.Sub(&b.used, &reserved)
	old.released = true
	b.entries[r.ID] = old
	return nil
}

// Used returns the smallest float64 at least as large as the exact liability.
// All finite nonnegative float64 units, including subnormals, are supported.
// Rounded reporting does not affect admission or reconciliation.
func (b *MemoryBudget) Used() float64 {
	b.mu.Lock()
	defer b.mu.Unlock()
	used, exact := b.used.Float64()
	if !exact {
		var reported big.Rat
		reported.SetFloat64(used)
		if reported.Cmp(&b.used) < 0 {
			used = math.Nextafter(used, math.Inf(1))
		}
	}
	return used
}

// Claim atomically authorizes one dispatch. Idempotent reservation lookup does not
// authorize executing the external effect again.
func (b *MemoryBudget) Claim(ctx context.Context, r Reservation) error {
	if e := ctx.Err(); e != nil {
		return e
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if e := ctx.Err(); e != nil {
		return e
	}
	old, ok := b.entries[r.ID]
	if !ok || old.reservation != r || old.released || old.claimed || old.settled {
		return ErrConflict
	}
	old.claimed = true
	b.entries[r.ID] = old
	return nil
}
