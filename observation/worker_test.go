package observation_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/skosovsky/evaly"
	"github.com/skosovsky/evaly/observation"
)

type fakeClock struct {
	mu     sync.Mutex
	now    time.Time
	alarms []alarm
}
type alarm struct {
	at time.Time
	ch chan time.Time
}

func (c *fakeClock) Now() time.Time { c.mu.Lock(); defer c.mu.Unlock(); return c.now }
func (c *fakeClock) After(d time.Duration) <-chan time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	ch := make(chan time.Time, 1)
	c.alarms = append(c.alarms, alarm{c.now.Add(d), ch})
	return ch
}
func (c *fakeClock) advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
	remaining := []alarm{}
	for _, a := range c.alarms {
		if !a.at.After(c.now) {
			a.ch <- c.now
		} else {
			remaining = append(remaining, a)
		}
	}
	c.alarms = remaining
}
func saved(t *testing.T, complete bool) evaly.SavedView[int, string, int] {
	t.Helper()
	capture, e := evaly.NewCapture(
		evaly.CaptureConfig{
			Policy:     evaly.FieldPolicy{ID: "none"},
			KnownKinds: []string{"outcome"},
			MaxEvents:  10,
			MaxBytes:   1024,
		},
	)
	if e != nil {
		t.Fatal(e)
	}
	if !complete {
		capture.MarkIncomplete("pending outcome")
	}
	reference := 1
	view := evaly.View[int, string, int]{
		Case:     evaly.Case[int, int]{ID: "a", Revision: "case-v1", Input: 1, Reference: &reference},
		Output:   "done",
		Evidence: capture.Seal(),
	}
	s, e := evaly.SaveView(
		view,
		"safe",
		evaly.JSONCodec[int]{ID: "i", Version: "1"},
		evaly.JSONCodec[string]{ID: "o", Version: "1"},
		evaly.JSONCodec[int]{ID: "r", Version: "1"},
	)
	if e != nil {
		t.Fatal(e)
	}
	return s
}
func TestBoundedOverloadFakeDeadlineAndFlush(t *testing.T) {
	// Arrange.
	clock := &fakeClock{now: time.Unix(1, 0)}
	started := make(chan struct{})
	grader := evaly.GraderFunc[int, string, int]{
		Identity: evaly.GraderRevision{ID: "judge", Implementation: "scripted", Rubric: "1"},
		Evaluate: func(ctx context.Context, _ evaly.View[int, string, int]) (evaly.Grade, error) {
			close(started)
			<-ctx.Done()
			return evaly.Grade{}, ctx.Err()
		},
	}
	worker, e := observation.Start(
		context.Background(),
		observation.Config[int, string, int]{
			Capacity:    1,
			Concurrency: 1,
			Deadline:    time.Minute,
			Clock:       clock,
			Graders:     []evaly.Grader[int, string, int]{grader},
		},
	)
	if e != nil {
		t.Fatal(e)
	}
	cleanupctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	defer worker.Cancel(cleanupctx)
	obs, e := observation.New(
		"one",
		"",
		observation.Sampling{Rule: "all", Population: "staging", Window: "1"},
		saved(t, false),
	)
	if e != nil {
		t.Fatal(e)
	}
	// Act.
	first, status := worker.Enqueue(obs)
	if status != "accepted" {
		t.Fatal(status)
	}
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("worker did not dispatch")
	}
	second, status := worker.Enqueue(obs)
	if status != "accepted" {
		t.Fatal(status)
	}
	_, overload := worker.Enqueue(obs)
	clock.advance(time.Minute)
	// Assert.
	if overload != "rejected_overload" {
		t.Fatal(overload)
	}
	if e = worker.Flush(cleanupctx); e != nil {
		t.Fatal(e)
	}
	r1 := <-first.Result
	r2 := <-second.Result
	if r1.State != "expired" || r2.State != "expired" || r1.Assessment.Grades[0].Status != evaly.GraderError {
		t.Fatal(r1, r2)
	}
	if e = worker.Cancel(cleanupctx); e != nil {
		t.Fatal(e)
	}
	_, closed := worker.Enqueue(obs)
	if closed != "rejected_closed" {
		t.Fatal(closed)
	}
}
func TestDelayedObservationLineageAndOfflineRegrade(t *testing.T) {
	// Arrange.
	original, e := observation.New(
		"observation",
		"",
		observation.Sampling{Rule: "sampled", Population: "production", Window: "1", OutcomeDelay: time.Hour},
		saved(t, false),
	)
	if e != nil {
		t.Fatal(e)
	}
	grader := evaly.AbsenceGrader[int, string, int](
		evaly.GraderRevision{ID: "outcome", Implementation: "go", Rubric: "1"},
		"outcome",
		func(evaly.Event) bool { return false },
	)
	worker, e := observation.Start(
		context.Background(),
		observation.Config[int, string, int]{
			Capacity:    1,
			Concurrency: 1,
			Deadline:    time.Second,
			Clock:       observation.RealClock{},
			Graders:     []evaly.Grader[int, string, int]{grader},
		},
	)
	if e != nil {
		t.Fatal(e)
	}
	defer worker.Cancel(context.Background())
	// Act.
	first, _ := worker.Enqueue(original)
	r1 := <-first.Result
	later, e := observation.New(
		"observation",
		original.Record().Revision,
		observation.Sampling{Rule: "sampled", Population: "production", Window: "1"},
		saved(t, true),
	)
	if e != nil {
		t.Fatal(e)
	}
	second, _ := worker.Enqueue(later)
	r2 := <-second.Result
	// Assert.
	if r1.Assessment.Grades[0].Status != evaly.InsufficientEvidence || r2.Assessment.Grades[0].Status != evaly.Scored ||
		r2.Assessment.Parent != original.Record().Revision ||
		r1.Assessment.Revision == r2.Assessment.Revision {
		t.Fatal(r1, r2)
	}
}
