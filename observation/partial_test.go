package observation_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/skosovsky/evaly"
	"github.com/skosovsky/evaly/observation"
)

type failedReconcile struct{ *evaly.MemoryBudget }

func TestWorkerRejectsTypedNilInfrastructure(t *testing.T) {
	for _, missing := range []string{"clock", "budget"} {
		t.Run(missing, func(t *testing.T) {
			// Arrange: interface presence is insufficient for a configured port.
			c := observation.Config[int, string, int]{
				Capacity: 1, Concurrency: 1, Deadline: time.Second,
				Clock: observation.RealClock{},
				Graders: []evaly.Grader[int, string, int]{evaly.GraderFunc[int, string, int]{
					Identity: evaly.GraderRevision{ID: "g", Implementation: "1", Rubric: "1"},
					Evaluate: func(context.Context, evaly.View[int, string, int]) (evaly.Grade, error) {
						t.Fatal("unexpected grader dispatch")
						return evaly.Grade{}, nil
					},
				}},
			}
			if missing == "clock" {
				var clock *fakeClock
				c.Clock = clock
			} else {
				var budget *evaly.MemoryBudget
				c.Budget = budget
			}
			// Act.
			worker, err := observation.Start(context.Background(), c)
			// Assert: rejected before workers or clock/grader callbacks start.
			if err == nil || worker != nil {
				t.Fatal(worker, err)
			}
		})
	}
}

func (b failedReconcile) Reconcile(context.Context, evaly.Reservation, evaly.Usage) error {
	return evaly.ErrConflict
}
func TestPartialAssessmentRetainsPaidResults(t *testing.T) {
	for _, mode := range []string{"budget", "reconcile", "deadline"} {
		t.Run(mode, func(t *testing.T) {
			// Arrange.
			calls := 0
			clock := &fakeClock{now: time.Unix(1, 0)}
			memory, _ := evaly.NewMemoryBudget(1)
			var budget evaly.Budget = memory
			if mode == "reconcile" {
				budget = failedReconcile{memory}
			}
			makeGrader := func(id string) evaly.Grader[int, string, int] {
				return evaly.GraderFunc[int, string, int]{
					Identity: evaly.GraderRevision{ID: id, Implementation: "1", Rubric: "1"},
					Evaluate: func(context.Context, evaly.View[int, string, int]) (evaly.Grade, error) {
						calls++
						if mode == "deadline" {
							clock.advance(2 * time.Minute)
						}
						return evaly.Grade{
							Status:     evaly.Scored,
							Assertions: []evaly.Assertion{{Name: "ok", Pass: true}},
							Usage:      evaly.Usage{Known: true, Units: 1},
						}, nil
					},
				}
			}
			graders := []evaly.Grader[int, string, int]{makeGrader("a"), makeGrader("b")}
			if mode == "reconcile" {
				graders = graders[:1]
			}
			w, e := observation.Start(
				context.Background(),
				observation.Config[int, string, int]{
					Capacity:    1,
					Concurrency: 1,
					Deadline:    time.Minute,
					Clock:       clock,
					Graders:     graders,
					Budget:      budget,
					GraderUnits: 1,
				},
			)
			if e != nil {
				t.Fatal(e)
			}
			defer w.Cancel(context.Background())
			obs, _ := observation.New(
				"partial",
				"parent",
				observation.Sampling{Rule: "all", Population: "test", Window: "1"},
				saved(t, true),
			)
			// Act.
			future, status := w.Enqueue(obs)
			if status != "accepted" {
				t.Fatal(status)
			}
			r := <-future.Result
			// Assert.
			if calls != 1 || len(r.Assessment.Grades) != 1 || r.Assessment.State != "partial" ||
				r.Assessment.StopReason == "" ||
				r.Assessment.Grades[0].Usage.Units != 1 ||
				evaly.ValidateAssessment(r.Assessment) != nil ||
				r.Version != 2 {
				t.Fatal(calls, r)
			}
			if mode == "reconcile" && len(r.Assessment.Skipped) != 0 {
				t.Fatal(r)
			}
			if mode != "reconcile" && len(r.Assessment.Skipped) != 1 {
				t.Fatal(r)
			}
		})
	}
}

type cancelDecode struct {
	enabled *atomic.Bool
	cancel  context.CancelFunc
	evaly.JSONCodec[int]
}

func (c cancelDecode) Decode(b []byte) (int, error) {
	if c.enabled.Load() {
		c.cancel()
	}
	return c.JSONCodec.Decode(b)
}
func TestCancelledDuringRescorePreflightRetainsCanonicalAssessment(t *testing.T) {
	// Arrange: codec cancellation occurs after the worker dispatch precheck, during rescore preflight.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	enabled := &atomic.Bool{}
	codec := cancelDecode{enabled: enabled, cancel: cancel, JSONCodec: evaly.JSONCodec[int]{ID: "input", Version: "1"}}
	base := saved(t, true)
	view, e := base.View()
	if e != nil {
		t.Fatal(e)
	}
	s, e := evaly.SaveView(
		view,
		"safe",
		codec,
		evaly.JSONCodec[string]{ID: "output", Version: "1"},
		evaly.JSONCodec[int]{ID: "ref", Version: "1"},
	)
	if e != nil {
		t.Fatal(e)
	}
	calls := 0
	grader := evaly.GraderFunc[int, string, int]{
		Identity: evaly.GraderRevision{ID: "g", Implementation: "1", Rubric: "1"},
		Evaluate: func(context.Context, evaly.View[int, string, int]) (evaly.Grade, error) {
			calls++
			return evaly.Grade{}, nil
		},
	}
	budget, _ := evaly.NewMemoryBudget(1)
	w, e := observation.Start(
		ctx,
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
	if e != nil {
		t.Fatal(e)
	}
	defer w.Cancel(context.Background())
	obs, e := observation.New("cancel", "", observation.Sampling{Rule: "all", Population: "test", Window: "1"}, s)
	if e != nil {
		t.Fatal(e)
	}
	enabled.Store(true)
	// Act.
	future, status := w.Enqueue(obs)
	if status != "accepted" {
		t.Fatal(status)
	}
	result := <-future.Result
	// Assert.
	if calls != 0 || result.State != "cancelled" || len(result.Assessment.Grades) != 0 ||
		len(result.Assessment.Skipped) != 1 ||
		evaly.ValidateAssessment(result.Assessment) != nil ||
		budget.Used() != 1 {
		t.Fatal(calls, result, budget.Used())
	}
}

func TestStartRejectsInvalidBuiltInGradingPorts(t *testing.T) {
	for _, kind := range []string{"function", "llm", "scripted"} {
		t.Run(kind, func(t *testing.T) {
			// Arrange.
			calls := 0
			rev := evaly.GraderRevision{ID: "invalid", Implementation: "1", Rubric: "1"}
			first := evaly.GraderFunc[int, string, int]{
				Identity: evaly.GraderRevision{ID: "first", Implementation: "1", Rubric: "1"},
				Evaluate: func(context.Context, evaly.View[int, string, int]) (evaly.Grade, error) {
					calls++
					return evaly.Grade{}, nil
				},
			}
			var invalid evaly.Grader[int, string, int]
			switch kind {
			case "function":
				invalid = evaly.GraderFunc[int, string, int]{Identity: rev}
			case "llm":
				invalid = evaly.LLMGrader[int, string, int]{Identity: rev, Instructions: "rubric"}
			case "scripted":
				invalid = evaly.LLMGrader[int, string, int]{
					Identity:     rev,
					Instructions: "rubric",
					Port:         evaly.ScriptedJudge[int, string, int]{},
				}
			}
			// Act.
			w, e := observation.Start(
				context.Background(),
				observation.Config[int, string, int]{
					Capacity:    1,
					Concurrency: 1,
					Deadline:    time.Second,
					Clock:       observation.RealClock{},
					Graders:     []evaly.Grader[int, string, int]{first, invalid},
				},
			)
			// Assert.
			if e == nil || w != nil || calls != 0 {
				if w != nil {
					w.Cancel(context.Background())
				}
				t.Fatal(w, e, calls)
			}
		})
	}
}
