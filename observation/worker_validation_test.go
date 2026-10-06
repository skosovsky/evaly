package observation_test

import (
	"context"
	"testing"
	"time"

	"github.com/skosovsky/evaly"
	"github.com/skosovsky/evaly/observation"
)

func TestDuplicateObservationCannotBypassGraderBudget(t *testing.T) {
	// Arrange: one-unit budget, every grader dispatch costs exactly one unit.
	budget, _ := evaly.NewMemoryBudget(1)
	calls := 0
	grader := evaly.GraderFunc[int, string, int]{
		Identity: evaly.GraderRevision{ID: "g", Implementation: "1", Rubric: "1"},
		Evaluate: func(context.Context, evaly.View[int, string, int]) (evaly.Grade, error) {
			calls++
			return evaly.Grade{
				Status:     evaly.Scored,
				Assertions: []evaly.Assertion{{Name: "ok", Pass: true}},
				Usage:      evaly.Usage{Known: true, Units: 1},
			}, nil
		},
	}
	w, err := observation.Start(
		context.Background(),
		observation.Config[int, string, int]{
			Capacity:    1,
			Concurrency: 1,
			Deadline:    time.Second,
			Clock:       observation.RealClock{},
			Graders:     []evaly.Grader[int, string, int]{grader},
			Budget:      budget,
			GraderUnits: 1,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Cancel(context.Background())
	obs, err := observation.New(
		"duplicate",
		"",
		observation.Sampling{Rule: "all", Population: "test", Window: "1"},
		saved(t, true),
	)
	if err != nil {
		t.Fatal(err)
	}
	// Act: the same immutable observation is delivered twice (common at-least-once input).
	first, status := w.Enqueue(obs)
	if status != "accepted" {
		t.Fatal(status)
	}
	<-first.Result
	second, status := w.Enqueue(obs)
	if status == "accepted" {
		<-second.Result
	}
	// Assert: either deduplicate or reserve a fresh assessment dispatch identity.
	if calls > 1 {
		t.Fatalf("budget bypass: %d paid calls on capacity 1, recorded usage %v", calls, budget.Used())
	}
}

type advanceDuringTimerClock struct{ now time.Time }

func (c *advanceDuringTimerClock) Now() time.Time { return c.now }
func (c *advanceDuringTimerClock) After(d time.Duration) <-chan time.Time {
	// Simulate fake clock advancement after remaining duration was read but before timer registration.
	c.now = c.now.Add(2 * d)
	return make(chan time.Time)
}
func TestWorkerCannotExtendAbsoluteDeadlineDuringTimerRegistration(t *testing.T) {
	// Arrange: absolute observation deadline passes while its relative timer is registered.
	clock := &advanceDuringTimerClock{now: time.Unix(1, 0)}
	calls := 0
	grader := evaly.GraderFunc[int, string, int]{
		Identity: evaly.GraderRevision{ID: "g", Implementation: "1", Rubric: "1"},
		Evaluate: func(context.Context, evaly.View[int, string, int]) (evaly.Grade, error) {
			calls++
			return evaly.Grade{Status: evaly.Scored, Assertions: []evaly.Assertion{{Name: "ok", Pass: true}}}, nil
		},
	}
	w, err := observation.Start(
		context.Background(),
		observation.Config[int, string, int]{
			Capacity:    1,
			Concurrency: 1,
			Deadline:    time.Minute,
			Clock:       clock,
			Graders:     []evaly.Grader[int, string, int]{grader},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Cancel(context.Background())
	obs, _ := observation.New(
		"deadline",
		"",
		observation.Sampling{Rule: "all", Population: "test", Window: "1"},
		saved(t, true),
	)
	// Act.
	future, status := w.Enqueue(obs)
	if status != "accepted" {
		t.Fatal(status)
	}
	r := <-future.Result
	// Assert: timer setup cannot convert an expired observation into another paid evaluation.
	if r.State != observation.Expired || calls != 0 {
		t.Fatalf("absolute deadline extended: result=%s calls=%d", r.State, calls)
	}
}

func TestWorkerAndOfflineUseSameAssessmentDigest(t *testing.T) {
	// Arrange: identical immutable assessment fields created by two public reference paths.
	grader := evaly.GraderFunc[int, string, int]{
		Identity: evaly.GraderRevision{ID: "g", Implementation: "1", Rubric: "1"},
		Evaluate: func(context.Context, evaly.View[int, string, int]) (evaly.Grade, error) {
			return evaly.Grade{Status: evaly.Scored, Assertions: []evaly.Assertion{{Name: "ok", Pass: true}}}, nil
		},
	}
	graders := []evaly.Grader[int, string, int]{grader}
	s := saved(t, true)
	obs, _ := observation.New("same", "", observation.Sampling{Rule: "all", Population: "test", Window: "1"}, s)
	w, err := observation.Start(
		context.Background(),
		observation.Config[int, string, int]{
			Capacity:    1,
			Concurrency: 1,
			Deadline:    time.Second,
			Clock:       observation.RealClock{},
			Graders:     graders,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Cancel(context.Background())
	// Act.
	future, _ := w.Enqueue(obs)
	r := <-future.Result
	offline, err := evaly.Rescore(context.Background(), s, graders, "same", "", "observation")
	if err != nil {
		t.Fatal(err)
	}
	// Assert: content identity cannot depend on which authoring path serialized the same assessment.
	if r.Assessment.Revision != offline.Revision {
		t.Fatalf("noncanonical assessment identity: worker=%s offline=%s", r.Assessment.Revision, offline.Revision)
	}
}
