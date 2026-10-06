package evaly_test

import (
	"context"
	"errors"
	"math"
	"strconv"
	"sync/atomic"
	"testing"

	"github.com/skosovsky/evaly"
)

type countingInputCodec struct {
	evaly.JSONCodec[input]

	decodes *atomic.Int64
}

func (c countingInputCodec) Decode(b []byte) (input, error) {
	c.decodes.Add(1)
	return c.JSONCodec.Decode(b)
}

func TestRunPreflightBeforeEffects(t *testing.T) {
	for _, variant := range []string{"output_identity", "unsupported_case_evidence", "grader_identity", "paired_candidate", "nil_budget", "zero_budget"} {
		t.Run(variant, func(t *testing.T) {
			// Arrange.
			c := config(t, 1)
			var prepare, reset, target, judge atomic.Int32
			life := c.Lifecycle.(evaly.LifecycleFuncs[*int])
			oldPrepare, oldReset := life.PrepareFunc, life.ResetFunc
			life.PrepareFunc = func(ctx context.Context, id string) (*int, error) { prepare.Add(1); return oldPrepare(ctx, id) }
			life.ResetFunc = func(ctx context.Context, e *int) error { reset.Add(1); return oldReset(ctx, e) }
			c.Lifecycle = life
			oldTarget := c.Target
			c.Target = evaly.TargetFunc[input, int, *int](
				func(ctx context.Context, i input, tc evaly.TrialContext[*int]) (evaly.TargetResult[int], error) {
					target.Add(1)
					return oldTarget.Run(ctx, i, tc)
				},
			)
			g := c.Graders[0].(evaly.GraderFunc[input, int, int])
			oldGrade := g.Evaluate
			g.Evaluate = func(ctx context.Context, v evaly.View[input, int, int]) (evaly.Grade, error) {
				judge.Add(1)
				return oldGrade(ctx, v)
			}
			c.Graders[0] = g
			candidate := c
			candidate.ID = "candidate"
			switch variant {
			case "nil_budget":
				var budget *evaly.MemoryBudget
				c.Budget = budget
			case "zero_budget":
				c.Budget = &evaly.MemoryBudget{}
			case "output_identity":
				c.OutputCodec = evaly.JSONCodec[int]{}
			case "unsupported_case_evidence":
				cs, _ := c.Dataset.Cases()
				cs[0].RequiredEvidence = []string{"unsupported"}
				c.Dataset, _ = (evaly.DatasetDraft[input, int]{Selection: "all", Cases: cs}).Seal(
					evaly.JSONCodec[input]{ID: "calculation", Version: "1"},
					evaly.JSONCodec[int]{ID: "integer", Version: "1"},
				)
			case "grader_identity":
				g.Identity.Rubric = ""
				c.Graders[0] = g
			case "paired_candidate":
				candidate.OutputCodec = evaly.JSONCodec[int]{}
			}
			// Act.
			var err error
			if variant == "paired_candidate" {
				_, _, err = evaly.RunPaired(context.Background(), c, candidate, "pair")
			} else {
				_, err = evaly.Run(context.Background(), c)
			}
			// Assert.
			if err == nil || prepare.Load() != 0 || reset.Load() != 0 || target.Load() != 0 || judge.Load() != 0 {
				t.Fatalf(
					"err=%v prepare=%d reset=%d target=%d judge=%d",
					err,
					prepare.Load(),
					reset.Load(),
					target.Load(),
					judge.Load(),
				)
			}
		})
	}
}

func TestRunLinearCaseDecodingAndIsolation(t *testing.T) {
	for _, n := range []int{1, 8, 32} {
		t.Run(strconv.Itoa(n), func(t *testing.T) {
			// Arrange.
			c := config(t, n)
			c.Plan.Concurrency = 1
			var count atomic.Int64
			codec := countingInputCodec{
				ID: "calculation", Version: "1",
				decodes: &count,
			}
			cases, _ := c.Dataset.Cases()
			c.Dataset, _ = (evaly.DatasetDraft[input, int]{Selection: "all", Cases: cases}).Seal(
				codec,
				evaly.JSONCodec[int]{ID: "integer", Version: "1"},
			)
			original := c.Target
			c.Target = evaly.TargetFunc[input, int, *int](
				func(ctx context.Context, i input, tc evaly.TrialContext[*int]) (evaly.TargetResult[int], error) {
					result, err := original.Run(ctx, i, tc)
					i.Numbers[0] = 999
					return result, err
				},
			)
			g := c.Graders[0].(evaly.GraderFunc[input, int, int])
			g.Identity.ID = "mutating"
			g.Evaluate = func(_ context.Context, v evaly.View[input, int, int]) (evaly.Grade, error) {
				v.Case.Input.Numbers[0] = 777
				*v.Case.Reference = 777
				v.Evidence.Coverage["tool"] = false
				return evaly.Grade{
					Status:     evaly.Scored,
					Assertions: []evaly.Assertion{{Name: "mutated", Pass: true}},
				}, nil
			}
			c.Graders = append([]evaly.Grader[input, int, int]{g}, c.Graders...)
			// Act.
			exp, err := evaly.Run(context.Background(), c)
			// Assert.
			if err != nil {
				t.Fatal(err)
			}
			if count.Load() != int64(n*3) {
				t.Fatalf("N=%d decodes=%d want=%d", n, count.Load(), n*3)
			}
			for _, trial := range exp.Record().Trials {
				if !trial.Grades[1].Assertions[0].Pass {
					t.Fatal("target or grader mutation leaked")
				}
			}
		})
	}
}

func TestStopOnInfrastructureRetainsSlotsAndAvoidsContaminatedRetry(t *testing.T) {
	for _, isolation := range []evaly.Isolation{evaly.SerialShared, evaly.Isolated} {
		for _, failure := range []string{"setup", "cleanup", "grader", "budget", "usage"} {
			t.Run(string(isolation)+"/"+failure, func(t *testing.T) {
				checkStopOnInfrastructureRetainsSlotsAndAvoidsContaminatedRetry(t, &isolation, &failure)
			},
			)
		}
	}
}

func BenchmarkDatasetCaseAt(b *testing.B) {
	for _, n := range []int{10, 100, 1000} {
		b.Run(strconv.Itoa(n), func(b *testing.B) {
			// Arrange.
			cases := make([]evaly.Case[input, int], n)
			for i := range n {
				cases[i] = evaly.Case[input, int]{ID: strconv.Itoa(i), Input: input{Numbers: []int{i, 1}}}
			}
			d, err := (evaly.DatasetDraft[input, int]{Selection: "all", Cases: cases}).Seal(
				evaly.JSONCodec[input]{ID: "input", Version: "1"},
				evaly.JSONCodec[int]{ID: "ref", Version: "1"},
			)
			if err != nil {
				b.Fatal(err)
			}
			b.ReportAllocs()
			b.ResetTimer()
			// Act and Assert.
			for i := range b.N {
				for index := range n {
					if _, err := d.CaseAt(index); err != nil {
						b.Fatal(err)
					}
				}
				_ = i
			}
		})
	}
}

type executionSteps struct{ contexts []evaly.ScenarioContext }

func (s *executionSteps) Revision() string { return "execution-context-v2" }
func (s *executionSteps) Step(_ context.Context, state int, execution evaly.ScenarioContext) (int, int, bool, error) {
	s.contexts = append(s.contexts, execution)
	return state + 1, state + 1, state == 1, nil
}

func TestScenarioExecutionContextAndRestore(t *testing.T) {
	// Arrange.
	driver := &executionSteps{}
	plan := evaly.ScenarioPlan{Mode: "search", Seed: 42, MaxSteps: 4, Timeout: 1000000000}
	codec := evaly.JSONCodec[int]{ID: "int", Version: "1"}
	// Act.
	record, err := evaly.RunScenario(context.Background(), driver, 0, plan, codec, codec)
	restored, restoreErr := evaly.RestoreScenario(record, codec, codec)
	// Assert.
	if err != nil || restoreErr != nil || record.Version != 2 || restored.Steps != 2 || len(driver.contexts) != 2 {
		t.Fatalf("%+v %v %v", record, err, restoreErr)
	}
	for i, execution := range driver.contexts {
		if execution.Seed != 42 || execution.Mode != "search" || execution.Step != i {
			t.Fatalf("%+v", execution)
		}
	}
	legacy := record
	legacy.Version = 1
	if _, err := evaly.RestoreScenario(legacy, codec, codec); !errors.Is(err, evaly.ErrUnsupported) {
		t.Fatal(err)
	}
	if len(driver.contexts) != 2 {
		t.Fatal("restore dispatched driver")
	}
}

func TestCaseAtPrivateMetadataAndProvenance(t *testing.T) {
	// Arrange.
	ref := 3
	draft := evaly.DatasetDraft[input, int]{
		Selection: "all",
		Cases: []evaly.Case[input, int]{
			{
				ID:               "one",
				Input:            input{Numbers: []int{1, 2}},
				Reference:        &ref,
				Metadata:         map[string]string{"group": "a"},
				RequiredEvidence: []string{"tool"},
				Generation:       &evaly.Generation{Generator: "host", Mode: "search", LabelValidated: true},
			},
		},
	}
	d, err := draft.Seal(
		evaly.JSONCodec[input]{ID: "input", Version: "1"},
		evaly.JSONCodec[int]{ID: "ref", Version: "1"},
	)
	if err != nil {
		t.Fatal(err)
	}
	// Act.
	first, err := d.CaseAt(0)
	if err != nil {
		t.Fatal(err)
	}
	first.Input.Numbers[0] = 99
	*first.Reference = 99
	first.Metadata["group"] = "changed"
	first.RequiredEvidence[0] = "changed"
	first.Generation.Generator = "changed"
	second, err := d.CaseAt(0)
	// Assert.
	if err != nil || second.Input.Numbers[0] != 1 || *second.Reference != 3 || second.Metadata["group"] != "a" ||
		second.RequiredEvidence[0] != "tool" ||
		second.Generation.Generator != "host" {
		t.Fatalf("%+v %v", second, err)
	}
	for _, index := range []int{-1, 1} {
		if _, err := d.CaseAt(index); !errors.Is(err, evaly.ErrInvalid) {
			t.Fatal(err)
		}
	}
}

type failingStateCodec struct{ evaly.JSONCodec[int] }

func (c failingStateCodec) Encode(state int) ([]byte, error) {
	if state > 0 {
		return nil, errors.New("state serialization")
	}
	return c.JSONCodec.Encode(state)
}
func TestScenarioRetainsCanonicalRecordAfterStateCodecFailure(t *testing.T) {
	// Arrange.
	driver := &executionSteps{}
	sc := failingStateCodec{evaly.JSONCodec[int]{ID: "state", Version: "1"}}
	oc := evaly.JSONCodec[int]{ID: "output", Version: "1"}
	// Act.
	record, err := evaly.RunScenario(
		context.Background(),
		driver,
		0,
		evaly.ScenarioPlan{Mode: "replay", MaxSteps: 3, Timeout: 1000000000},
		sc,
		oc,
	)
	restored, restoreErr := evaly.RestoreScenario(record, sc, oc)
	// Assert.
	if err == nil || restoreErr != nil || record.Revision == "" || record.Stop != "codec_error" || record.Steps != 1 ||
		record.StateStep != 0 ||
		len(record.Outputs) != 1 ||
		restored.State != 0 ||
		restored.StateStep != 0 ||
		len(driver.contexts) != 1 {
		t.Fatalf("%+v %v %v restored=%+v", record, err, restoreErr, restored)
	}
}

func TestBuiltInPortsRejectMissingCallbacksBeforeEffects(t *testing.T) {
	for _, variant := range []string{"prepare", "reset", "cleanup", "target", "typed_nil_lifecycle"} {
		t.Run(variant, func(t *testing.T) {
			// Arrange.
			c := config(t, 1)
			life := c.Lifecycle.(evaly.LifecycleFuncs[*int])
			var calls atomic.Int32
			originalPrepare := life.PrepareFunc
			life.PrepareFunc = func(ctx context.Context, id string) (*int, error) { calls.Add(1); return originalPrepare(ctx, id) }
			originalTarget := c.Target
			c.Target = evaly.TargetFunc[input, int, *int](
				func(ctx context.Context, i input, tc evaly.TrialContext[*int]) (evaly.TargetResult[int], error) {
					calls.Add(1)
					return originalTarget.Run(ctx, i, tc)
				},
			)
			switch variant {
			case "prepare":
				life.PrepareFunc = nil
			case "reset":
				life.ResetFunc = nil
			case "cleanup":
				life.CleanupFunc = nil
			case "target":
				c.Target = evaly.TargetFunc[input, int, *int](nil)
			}
			c.Lifecycle = life
			if variant == "typed_nil_lifecycle" {
				var absent *evaly.LifecycleFuncs[*int]
				c.Lifecycle = absent
			}
			// Act.
			result, err := evaly.Run(context.Background(), c)
			// Assert.
			if !errors.Is(err, evaly.ErrInvalid) || calls.Load() != 0 || result.Revision() != "" {
				t.Fatalf("calls=%d err=%v result=%+v", calls.Load(), err, result.Record())
			}
		})
	}
}
func checkStopOnInfrastructureRetainsSlotsAndAvoidsContaminatedRetry(t *testing.T, isolation *evaly.
	Isolation, failure *string) {
	t.Helper()
	// Arrange.
	c := config(t, 3)
	c.Plan.Concurrency = 1
	c.Plan.MaxAttempts = 2
	c.Plan.StopOnInfrastructure = true
	life := c.Lifecycle.(evaly.LifecycleFuncs[*int])
	life.IdentityValue.Isolation = (*isolation)
	var targetCalls, prepareCalls atomic.Int32
	original := c.Target
	c.Target = evaly.TargetFunc[input, int, *int](
		func(ctx context.Context, i input, tc evaly.TrialContext[*int]) (evaly.TargetResult[int], error) {
			targetCalls.Add(1)
			out, err := original.Run(ctx, i, tc)
			if (*failure) == "usage" {
				out.Usage = evaly.Usage{Known: true, Units: math.NaN()}
			}
			return out, err
		},
	)
	prepare := life.PrepareFunc
	life.PrepareFunc = func(ctx context.Context, id string) (*int, error) {
		prepareCalls.Add(1)
		env, err := prepare(ctx, id)
		if (*failure) == "setup" {
			return env, errors.New("setup")
		}
		return env, err
	}
	if (*failure) == "cleanup" {
		life.CleanupFunc = func(context.Context, *int) error { return errors.New("contaminated") }
	}
	if (*failure) == "grader" {
		g := c.Graders[0].(evaly.GraderFunc[input, int, int])
		g.Evaluate = func(context.Context, evaly.View[input, int, int]) (evaly.Grade, error) {
			return evaly.Grade{Usage: evaly.Usage{Known: true, Units: 2}}, errors.New("judge")
		}
		c.Graders[0] = g
	}
	if (*failure) == "budget" {
		c.Budget, _ = evaly.NewMemoryBudget(0)
	}
	c.Lifecycle = life

	// Act.
	exp, err := evaly.Run(context.Background(), c)

	// Assert.
	if err != nil {
		t.Fatal(err)
	}
	r := exp.Record()
	expectedAttempts := 1
	if (*failure) == "setup" {
		expectedAttempts = 2
	}
	if len(r.Trials) != expectedAttempts+2 || prepareCalls.Load() != int32(expectedAttempts) {
		t.Fatalf("%+v prepares=%d", r.Trials, prepareCalls.Load())
	}
	checkInfrastructureSkippedSlots(t, r, expectedAttempts)
	if (*failure) == "grader" && r.Trials[0].Grades[0].Usage.Units != 2 {
		t.Fatal("lost judge usage")
	}
	if (*failure) == "cleanup" && (targetCalls.Load() != 1 || r.Trials[0].Cleanup.State != "failed") {
		t.Fatal("contaminated retry")
	}
	if (*failure) == "usage" && (targetCalls.Load() != 1 || r.Trials[0].UsageError != "invalid_usage") {
		t.Fatal("usage failure did not stop future dispatch", r.Trials)
	}
}

func checkInfrastructureSkippedSlots(t *testing.T, r evaly.ExperimentRecord, expectedAttempts int) {
	t.Helper()
	for _, trial := range r.Trials[expectedAttempts:] {
		if trial.Status != evaly.InfrastructureStop || trial.Reason != "infrastructure_stop" ||
			len(trial.SkippedGraders) != 1 {
			t.Fatalf("%+v", trial)
		}
	}
}
