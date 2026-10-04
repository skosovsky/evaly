package evaly_test

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/skosovsky/evaly"
)

func TestPairedScheduleMatchesActualDispatchAndRepeats(t *testing.T) {
	// Arrange.
	b, c := config(t, 4), config(t, 4)
	b.ID, c.ID = "baseline", "candidate"
	b.Plan.Concurrency, c.Plan.Concurrency = 1, 1
	b.Plan.Repeats, c.Plan.Repeats = 3, 3
	b.Plan.Seed, c.Plan.Seed = -17, -17
	var actual []string
	var handles []*int
	old := b.Target
	target := evaly.TargetFunc[input, int, *int](
		func(ctx context.Context, i input, tc evaly.TrialContext[*int]) (evaly.TargetResult[int], error) {
			actual = append(actual, tc.ID)
			for _, handle := range handles {
				if handle == tc.Environment {
					t.Error("environment handle reused")
				}
			}
			handles = append(handles, tc.Environment)
			return old.Run(ctx, i, tc)
		},
	)
	b.Target, c.Target = target, target
	// Act.
	firstB, firstC, err := evaly.RunPaired(context.Background(), b, c, "pair")
	firstDispatch := append([]string(nil), actual...)
	actual = nil
	secondB, _, secondErr := evaly.RunPaired(context.Background(), b, c, "pair")
	// Assert.
	if err != nil || secondErr != nil {
		t.Fatal(err, secondErr)
	}
	schedule := firstB.Record().Manifest.PairSchedule
	if schedule == nil || len(schedule.Slots) != 12 ||
		!reflect.DeepEqual(schedule, firstC.Record().Manifest.PairSchedule) ||
		!reflect.DeepEqual(schedule, secondB.Record().Manifest.PairSchedule) ||
		!reflect.DeepEqual(firstDispatch, actual) {
		t.Fatal("non-reproducible schedule", schedule, firstDispatch, actual)
	}
	var expected []string
	firstSides := map[string]bool{}
	for _, slot := range schedule.Slots {
		firstSides[slot.First] = true
		expected = append(
			expected,
			fmt.Sprintf("%s/%s/%d/0", slot.First, slot.CaseID, slot.Repeat),
			fmt.Sprintf("%s/%s/%d/0", slot.Second, slot.CaseID, slot.Repeat),
		)
	}
	if !reflect.DeepEqual(expected, firstDispatch) || len(firstSides) != 2 || len(handles) != 48 {
		t.Fatal("actual dispatch differs from slot interleaving", expected, firstDispatch, firstSides, len(handles))
	}
	if _, restoreErr := evaly.RestoreExperiment(firstB.Record()); restoreErr != nil {
		t.Fatal(restoreErr)
	}
}

func TestPairedPreflightRejectsIncompatibleConfigurationBeforeEffects(t *testing.T) {
	for _, variant := range []string{"shared", "fixture", "reset", "repeats", "seed", "concurrency", "timeout", "capture", "grader", "output", "projection", "pair_id"} {
		t.Run(variant, func(t *testing.T) {
			// Arrange.
			b, c := config(t, 1), config(t, 1)
			b.ID, c.ID = "baseline", "candidate"
			var effects atomic.Int32
			life := c.Lifecycle.(evaly.LifecycleFuncs[*int])
			oldPrepare := life.PrepareFunc
			life.PrepareFunc = func(ctx context.Context, id string) (*int, error) { effects.Add(1); return oldPrepare(ctx, id) }
			b.Lifecycle, c.Lifecycle = life, life
			pairID := "pair"
			switch variant {
			case "shared":
				life.IdentityValue.Isolation = evaly.SerialShared
			case "fixture":
				life.IdentityValue.Fixture = "other"
			case "reset":
				life.IdentityValue.Reset = "other"
			case "repeats":
				c.Plan.Repeats++
			case "seed":
				c.Plan.Seed++
			case "concurrency":
				c.Plan.Concurrency++
			case "timeout":
				c.Plan.Timeout++
			case "capture":
				c.Capture.MaxBytes++
			case "grader":
				g := c.Graders[0].(evaly.GraderFunc[input, int, int])
				g.Identity.Rubric = "other"
				c.Graders[0] = g
			case "output":
				c.OutputCodec = evaly.JSONCodec[int]{ID: "other", Version: "1"}
			case "projection":
				c.ProjectionRevision = "other"
			case "pair_id":
				pairID = "../invalid"
			}
			c.Lifecycle = life
			// Act.
			_, _, err := evaly.RunPaired(context.Background(), b, c, pairID)
			// Assert.
			if !errors.Is(err, evaly.ErrInvalid) || effects.Load() != 0 {
				t.Fatal(variant, err, effects.Load())
			}
		})
	}
}

func TestPairedBoundedConcurrencyAndWithinSlotOrder(t *testing.T) {
	// Arrange.
	b, c := config(t, 8), config(t, 8)
	b.ID, c.ID = "baseline", "candidate"
	b.Plan.Concurrency, c.Plan.Concurrency = 3, 3
	var active, maximum atomic.Int32
	var mu sync.Mutex
	actual := map[string][]string{}
	old := b.Target
	target := evaly.TargetFunc[input, int, *int](
		func(ctx context.Context, i input, tc evaly.TrialContext[*int]) (evaly.TargetResult[int], error) {
			n := active.Add(1)
			for prior := maximum.Load(); n > prior && !maximum.CompareAndSwap(prior, n); prior = maximum.Load() {
			}
			defer active.Add(-1)
			parts := strings.Split(tc.ID, "/")
			mu.Lock()
			actual[parts[1]] = append(actual[parts[1]], parts[0])
			mu.Unlock()
			time.Sleep(time.Millisecond)
			return old.Run(ctx, i, tc)
		},
	)
	b.Target, c.Target = target, target
	// Act.
	be, _, err := evaly.RunPaired(context.Background(), b, c, "pair")
	// Assert.
	if err != nil || maximum.Load() > 3 || maximum.Load() < 2 {
		t.Fatal(err, maximum.Load())
	}
	for _, slot := range be.Record().Manifest.PairSchedule.Slots {
		if !reflect.DeepEqual(actual[slot.CaseID], []string{slot.First, slot.Second}) {
			t.Fatal(slot, actual)
		}
	}
}

func TestPairedCancellationAndSharedBudgetRetainEverySlot(t *testing.T) {
	for _, stop := range []string{"cancel", "budget", "infrastructure"} {
		t.Run(stop, func(t *testing.T) {
			// Arrange.
			b, c := config(t, 3), config(t, 3)
			b.ID, c.ID = "baseline", "candidate"
			b.Plan.Concurrency, c.Plan.Concurrency = 1, 1
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			var calls int
			old := b.Target
			target := evaly.TargetFunc[input, int, *int](
				func(ctx context.Context, i input, tc evaly.TrialContext[*int]) (evaly.TargetResult[int], error) {
					calls++
					if stop == "cancel" && calls == 3 {
						cancel()
					}
					return old.Run(ctx, i, tc)
				},
			)
			b.Target, c.Target = target, target
			if stop == "budget" {
				budget, err := evaly.NewMemoryBudget(2)
				if err != nil {
					t.Fatal(err)
				}
				b.Budget, c.Budget = budget, budget
			}
			if stop == "infrastructure" {
				b.Plan.StopOnInfrastructure, c.Plan.StopOnInfrastructure = true, true
				life := b.Lifecycle.(evaly.LifecycleFuncs[*int])
				life.CleanupFunc = func(context.Context, *int) error { return errors.New("cleanup unavailable") }
				b.Lifecycle, c.Lifecycle = life, life
			}
			// Act.
			be, ce, err := evaly.RunPaired(ctx, b, c, "pair")
			// Assert.
			if err != nil {
				t.Fatal(err)
			}
			br, cr := be.Record(), ce.Record()
			if len(br.Trials) != 3 || len(cr.Trials) != 3 || len(br.Manifest.PairSchedule.Slots) != 3 {
				t.Fatal("lost scheduled slots", br, cr)
			}
			wantCalls := map[string]int{"cancel": 3, "budget": 2, "infrastructure": 1}[stop]
			if calls != wantCalls {
				t.Fatal(stop, calls)
			}
			known := 0
			for _, record := range []evaly.ExperimentRecord{br, cr} {
				if _, restoreErr := evaly.RestoreExperiment(record); restoreErr != nil {
					t.Fatal(restoreErr)
				}
				for _, trial := range record.Trials {
					if trial.TargetUsage.Known {
						known++
					}
				}
			}
			if known != wantCalls {
				t.Fatal("lost known paid usage", known, wantCalls)
			}
		})
	}
}
