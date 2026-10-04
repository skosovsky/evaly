package evaly

import (
	"context"
	"fmt"
	"math"
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
type MemoryBudget struct {
	mu             sync.Mutex
	capacity, used float64
	entries        map[string]budgetEntry
}

func NewMemoryBudget(capacity float64) (*MemoryBudget, error) {
	if !finiteNonnegative(capacity) {
		return nil, ErrInvalid
	}
	return &MemoryBudget{capacity: capacity, entries: map[string]budgetEntry{}}, nil
}
func finiteNonnegative(v float64) bool { return !math.IsNaN(v) && !math.IsInf(v, 0) && v >= 0 }
func (b *MemoryBudget) Reserve(ctx context.Context, id string, units float64) (Reservation, error) {
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
	if b.used+units > b.capacity {
		return Reservation{}, ErrBudget
	}
	r := Reservation{id, units}
	b.entries[id] = budgetEntry{reservation: r}
	b.used += units
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
	b.used -= r.Units - u.Units
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
	b.used -= r.Units
	old.released = true
	b.entries[r.ID] = old
	return nil
}
func (b *MemoryBudget) Used() float64 { b.mu.Lock(); defer b.mu.Unlock(); return b.used }

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
