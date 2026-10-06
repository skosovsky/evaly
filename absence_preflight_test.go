package evaly_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/skosovsky/evaly"
	"github.com/skosovsky/evaly/observation"
)

type noDispatchClock struct{}

func (noDispatchClock) Now() time.Time { panic("clock dispatched during invalid preflight") }
func (noDispatchClock) After(time.Duration) <-chan time.Time {
	panic("clock dispatched during invalid preflight")
}

func absenceEvidence(t *testing.T, events int, complete bool) evaly.EvidenceRecord {
	t.Helper()
	capture, err := evaly.NewCapture(
		evaly.CaptureConfig{
			Policy:     evaly.FieldPolicy{ID: "none"},
			KnownKinds: []string{"tool"},
			MaxEvents:  2,
			MaxBytes:   1024,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	for i := range events {
		if err = capture.Record(t.Context(), evaly.Event{Version: 1, Sequence: i + 1, Kind: "tool"}); err != nil {
			t.Fatal(err)
		}
	}
	if !complete {
		capture.MarkIncomplete()
	}
	return capture.Seal()
}

func TestAbsenceInvalidConfigRejectedBeforeAllEffects(t *testing.T) {
	revision := evaly.GraderRevision{ID: "absence", Implementation: "v1", Rubric: "v1"}
	for _, variant := range []string{"nil_predicate", "empty_kind", "invalid_utf8_kind", "invalid_revision"} {
		t.Run(variant, func(t *testing.T) {
			checkAbsenceInvalidConfig(t, variant, revision)
		})
	}
}

func checkAbsenceInvalidConfig(t *testing.T, variant string, revision evaly.GraderRevision) {
	t.Helper()
	// Arrange: every callback counts an effect; setup itself is valid except grader.
	var effects atomic.Int64
	predicate := func(evaly.Event) bool { effects.Add(1); return true }
	kind := "tool"
	switch variant {
	case "nil_predicate":
		predicate = nil
	case "empty_kind":
		kind = ""
	case "invalid_utf8_kind":
		kind = string([]byte{0xff})
	case "invalid_revision":
		revision = evaly.GraderRevision{}
	}
	direct := evaly.AbsenceGrader[int, int, int](revision, kind, predicate)
	cfg := config(t, 1)
	cfg.Graders = []evaly.Grader[input, int, int]{evaly.AbsenceGrader[input, int, int](revision, kind, predicate)}
	cfg.Target = evaly.TargetFunc[input, int, *int](
		func(context.Context, input, evaly.TrialContext[*int]) (evaly.TargetResult[int], error) {
			effects.Add(1)
			return evaly.TargetResult[int]{}, nil
		},
	)
	cfg.Lifecycle = evaly.LifecycleFuncs[*int]{
		IdentityValue: cfg.Lifecycle.Identity(),
		PrepareFunc:   func(context.Context, string) (*int, error) { effects.Add(1); return new(int), nil },
		ResetFunc:     func(context.Context, *int) error { effects.Add(1); return nil },
		CleanupFunc:   func(context.Context, *int) error { effects.Add(1); return nil },
	}
	cfg.Project = func(context.Context, evaly.Case[input, int], int, evaly.EvidenceRecord) (evaly.View[input, int, int], error) {
		effects.Add(1)
		return evaly.View[input, int, int]{}, nil
	}
	// Act: evidence size cannot make invalid setup pass or panic.
	validation := evaly.ValidatePort(direct)
	for _, events := range []int{0, 1} {
		evidence := absenceEvidence(t, events, true)
		_, directErr := direct.Grade(t.Context(), evaly.View[int, int, int]{Evidence: evidence})
		grades := evaly.Assess(
			t.Context(),
			[]evaly.Grader[int, int, int]{direct},
			func() (evaly.View[int, int, int], error) {
				effects.Add(1)
				return evaly.View[int, int, int]{Evidence: evidence}, nil
			},
		)
		// Assert
		if !errors.Is(directErr, evaly.ErrInvalid) || len(grades) != 1 || grades[0].Dispatched ||
			grades[0].Status != evaly.GraderError {
			t.Fatal(directErr, grades)
		}
	}
	preflight := evaly.ValidateRunConfig(cfg)
	_, runErr := evaly.Run(t.Context(), cfg)
	worker, startErr := observation.Start(
		t.Context(),
		observation.Config[int, int, int]{
			Capacity:    1,
			Concurrency: 1,
			Deadline:    time.Second,
			Clock:       noDispatchClock{},
			Graders:     []evaly.Grader[int, int, int]{direct},
		},
	)
	// Assert
	if !errors.Is(validation, evaly.ErrInvalid) || !errors.Is(preflight, evaly.ErrInvalid) ||
		!errors.Is(runErr, evaly.ErrInvalid) ||
		!errors.Is(startErr, evaly.ErrInvalid) ||
		worker != nil ||
		effects.Load() != 0 {
		t.Fatal(validation, preflight, runErr, startErr, effects.Load())
	}
}

func TestAbsenceValidPredicateCoverage(t *testing.T) {
	// Arrange
	revision := evaly.GraderRevision{ID: "absence", Implementation: "v1", Rubric: "v1"}
	grader := evaly.AbsenceGrader[int, int, int](revision, "tool", func(evaly.Event) bool { return true })
	for _, variant := range []struct {
		name     string
		events   int
		complete bool
		status   evaly.GradeStatus
		pass     bool
	}{
		{"empty_complete", 0, true, evaly.Scored, true},
		{"forbidden", 1, true, evaly.Scored, false},
		{"empty_incomplete", 0, false, evaly.InsufficientEvidence, false},
		{"event_incomplete", 1, false, evaly.InsufficientEvidence, false},
	} {
		t.Run(variant.name, func(t *testing.T) {
			evidence := absenceEvidence(t, variant.events, variant.complete)
			// Act
			grades := evaly.Assess(
				t.Context(),
				[]evaly.Grader[int, int, int]{grader},
				func() (evaly.View[int, int, int], error) { return evaly.View[int, int, int]{Evidence: evidence}, nil },
			)
			// Assert
			if len(grades) != 1 || grades[0].Status != variant.status || !grades[0].Dispatched {
				t.Fatal(grades)
			}
			if variant.status == evaly.Scored && grades[0].Assertions[0].Pass != variant.pass {
				t.Fatal(grades)
			}
		})
	}
}
