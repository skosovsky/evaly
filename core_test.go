package evaly_test

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/skosovsky/evaly"
)

type input struct {
	Numbers []int             `json:"numbers"`
	Labels  map[string]string `json:"labels,omitempty"`
}

func dataset(t *testing.T, n int) evaly.Dataset[input, int] {
	t.Helper()
	cs := []evaly.Case[input, int]{}
	for i := range n {
		r := i + 1
		cs = append(
			cs,
			evaly.Case[input, int]{ID: string(rune('a' + i)), Input: input{Numbers: []int{i, 1}}, Reference: &r},
		)
	}
	d, e := (evaly.DatasetDraft[input, int]{Cases: cs, Selection: "all"}).Seal(
		evaly.JSONCodec[input]{ID: "calculation", Version: "1"},
		evaly.JSONCodec[int]{ID: "integer", Version: "1"},
	)
	if e != nil {
		t.Fatal(e)
	}
	return d
}
func config(t *testing.T, n int) evaly.RunConfig[input, int, int, *int] {
	t.Helper()
	return evaly.RunConfig[input, int, int, *int]{
		ID:          "run",
		OutputCodec: evaly.JSONCodec[int]{ID: "integer-output", Version: "1"},
		Dataset:     dataset(t, n),
		Target: evaly.TargetFunc[input, int, *int](
			func(ctx context.Context, i input, tc evaly.TrialContext[*int]) (evaly.TargetResult[int], error) {
				sum := 0
				for _, v := range i.Numbers {
					sum += v
				}
				return evaly.TargetResult[int]{Output: sum, Usage: evaly.Usage{Known: true, Units: 1}}, nil
			},
		),
		Lifecycle: evaly.LifecycleFuncs[*int]{
			IdentityValue: evaly.LifecycleIdentity{Fixture: "calc-v1", Reset: "clear-v1", Isolation: evaly.Isolated},
			PrepareFunc:   func(context.Context, string) (*int, error) { v := 0; return &v, nil },
			ResetFunc:     func(context.Context, *int) error { return nil },
			CleanupFunc:   func(context.Context, *int) error { return nil },
		},
		Plan: evaly.RunPlan{
			Repeats:         1,
			Concurrency:     2,
			Timeout:         time.Second,
			CleanupTimeout:  time.Second,
			MaxAttempts:     1,
			DispatchUnits:   1,
			GraderUnits:     0,
			AssertionPolicy: "all",
		},
		Provenance: evaly.Provenance{
			Target:  "sum-v1",
			Unknown: []string{"model", "prompt", "tools", "policy"},
		},
		Capture: evaly.CaptureConfig{
			Policy:     evaly.FieldPolicy{ID: "none-v1"},
			KnownKinds: []string{"tool", "outcome"},
			MaxEvents:  10,
			MaxBytes:   4096,
		},
		ProjectionRevision: "safe-v1",
		Project: func(ctx context.Context, c evaly.Case[input, int], o int, e evaly.EvidenceRecord) (evaly.View[input, int, int], error) {
			return evaly.View[input, int, int]{Case: c, Output: o, Evidence: e}, nil
		},
		Graders: []evaly.Grader[input, int, int]{
			evaly.GraderFunc[input, int, int]{
				Identity: evaly.GraderRevision{ID: "exact", Implementation: "go-v1", Rubric: "equal-v1"},
				Evaluate: func(ctx context.Context, v evaly.View[input, int, int]) (evaly.Grade, error) {
					return evaly.Grade{
						Status: evaly.Scored,
						Assertions: []evaly.Assertion{
							{Name: "equal", Pass: v.Case.Reference != nil && v.Output == *v.Case.Reference},
						},
					}, nil
				},
			},
		},
	}
}
func TestDatasetSnapshotAndCanonicalIdentity(t *testing.T) {
	// Arrange.
	reference := 3
	draft := evaly.DatasetDraft[input, int]{
		Selection: "all",
		Cases: []evaly.Case[input, int]{
			{
				ID:        "a",
				Input:     input{Numbers: []int{1, 2}, Labels: map[string]string{"z": "last", "a": "first"}},
				Reference: &reference,
			},
			{ID: "b", Input: input{Numbers: []int{2, 1}}, Reference: &reference},
		},
	}
	ic := evaly.JSONCodec[input]{ID: "input", Version: "1"}
	rc := evaly.JSONCodec[int]{ID: "ref", Version: "1"}
	// Act.
	sealed, e := draft.Seal(ic, rc)
	if e != nil {
		t.Fatal(e)
	}
	rev := sealed.Revision()
	draft.Cases[0].Input.Numbers[0] = 99
	draft.Cases[0].Input.Labels["z"] = "changed"
	reference = 100
	cs, e := sealed.Cases()
	if e != nil {
		t.Fatal(e)
	}
	cs[0].Input.Numbers[0] = 77
	again, _ := sealed.Cases()
	changed, e := draft.Seal(ic, rc)
	// Assert.
	if e != nil || changed.Revision() == rev || again[0].Input.Numbers[0] != 1 || *again[0].Reference != 3 {
		t.Fatalf("mutable snapshot: %+v %v", again, e)
	}
	a, e := evaly.CanonicalJSON([]byte(`{"z":1,"a":{"z":3,"a":2}}`))
	b, e2 := evaly.CanonicalJSON([]byte(`{"a":{"a":2,"z":3},"z":1}`))
	if e != nil || e2 != nil || string(a) != string(b) {
		t.Fatal("noncanonical map")
	}
	draft.Cases[1].ID = "a"
	if _, e = draft.Seal(ic, rc); !errors.Is(e, evaly.ErrConflict) {
		t.Fatal(e)
	}
	for _, bad := range []string{`{"a":1,"a":2}`, `{} {}`, `{"x":NaN}`} {
		if _, e = evaly.CanonicalJSON([]byte(bad)); e == nil {
			t.Fatal("accepted", bad)
		}
	}
	if _, e = (evaly.JSONCodec[float64]{ID: "f", Version: "1"}).Encode(math.Inf(1)); e == nil {
		t.Fatal("accepted infinity")
	}
	restored, e := evaly.RestoreDataset(sealed.Record(), ic, rc)
	if e != nil || restored.Revision() != rev {
		t.Fatal(e)
	}
}
func TestRunResetFailureAndCleanup(t *testing.T) {
	// Arrange.
	c := config(t, 1)
	c.Plan.Repeats = 3
	c.Plan.Concurrency = 1
	var reset, called, clean atomic.Int32
	c.Lifecycle = evaly.LifecycleFuncs[*int]{
		IdentityValue: c.Lifecycle.Identity(),
		PrepareFunc:   func(context.Context, string) (*int, error) { v := 0; return &v, nil },
		ResetFunc: func(context.Context, *int) error {
			if reset.Add(1) == 2 {
				return errors.New("reset failed")
			}
			return nil
		},
		CleanupFunc: func(ctx context.Context, e *int) error { clean.Add(1); return nil },
	}
	original := c.Target
	c.Target = evaly.TargetFunc[input, int, *int](
		func(ctx context.Context, i input, tc evaly.TrialContext[*int]) (evaly.TargetResult[int], error) {
			called.Add(1)
			return original.Run(ctx, i, tc)
		},
	)
	// Act.
	experiment, e := evaly.Run(context.Background(), c)
	// Assert.
	if e != nil {
		t.Fatal(e)
	}
	r := experiment.Record()
	if called.Load() != 2 || clean.Load() != 3 || r.Trials[1].Status != evaly.SetupError ||
		r.Trials[1].Cleanup.State != "completed" {
		t.Fatalf("%+v calls=%d cleanup=%d", r, called.Load(), clean.Load())
	}
	r.Trials[0].Grades[0].Assertions[0].Pass = false
	if !experiment.Record().Trials[0].Grades[0].Assertions[0].Pass {
		t.Fatal("mutable experiment")
	}
}
func TestEvidencePrivacyGapAndJudgeError(t *testing.T) {
	// Arrange.
	capture, e := evaly.NewCapture(
		evaly.CaptureConfig{
			Policy:        evaly.FieldPolicy{ID: "allow-action", Allowed: map[string][]string{"tool": {"action"}}},
			KnownKinds:    []string{"tool"},
			RequiredKinds: []string{"tool"},
			MaxEvents:     4,
			MaxBytes:      1024,
		},
	)
	if e != nil {
		t.Fatal(e)
	}
	// Act.
	if e = capture.Record(
		context.Background(),
		evaly.Event{
			Version:  1,
			Sequence: 1,
			Kind:     "tool",
			Payload:  json.RawMessage(`{"secret":"secret-token","action":"read"}`),
		},
	); e != nil {
		t.Fatal(e)
	}
	if e = capture.Record(
		context.Background(),
		evaly.Event{Version: 1, Sequence: 3, Kind: "tool", Payload: json.RawMessage(`{"action":"read"}`)},
	); e != nil {
		t.Fatal(e)
	}
	record := capture.Seal()
	rev := evaly.GraderRevision{ID: "absence", Implementation: "1", Rubric: "no-delete"}
	g := evaly.AbsenceGrader[input, int, int](
		rev,
		"tool",
		func(e evaly.Event) bool { return strings.Contains(string(e.Payload), "delete") },
	)
	grades := evaly.Assess(
		context.Background(),
		[]evaly.Grader[input, int, int]{g},
		func() (evaly.View[input, int, int], error) { return evaly.View[input, int, int]{Evidence: record}, nil },
	)
	timeout := evaly.GraderFunc[input, int, int]{
		Identity: rev,
		Evaluate: func(ctx context.Context, v evaly.View[input, int, int]) (evaly.Grade, error) {
			return evaly.Grade{}, context.DeadlineExceeded
		},
	}
	judge := evaly.Assess(context.Background(), []evaly.Grader[input, int, int]{timeout}, func() (evaly.View[input, int, int], error) { return evaly.View[input, int, int]{Evidence: record}, nil })[0]
	// Assert.
	bytes, _ := json.Marshal(record)
	if strings.Contains(string(bytes), "secret-token") || record.State != "incomplete" ||
		grades[0].Status != evaly.InsufficientEvidence ||
		judge.Status != evaly.GraderError ||
		len(judge.Metrics) != 0 {
		t.Fatalf("%s %+v %+v", bytes, grades, judge)
	}
	if e = capture.Record(context.Background(), evaly.Event{}); !errors.Is(e, evaly.ErrClosed) {
		t.Fatal(e)
	}
}
func TestBudgetUnknownUsageAndCancellation(t *testing.T) {
	// Arrange.
	budget, e := evaly.NewMemoryBudget(2)
	if e != nil {
		t.Fatal(e)
	}
	ctx := context.Background()
	// Act.
	first, e := budget.Reserve(ctx, "one", 1)
	if e != nil {
		t.Fatal(e)
	}
	if e = budget.Reconcile(ctx, first, evaly.Usage{}); e != nil {
		t.Fatal(e)
	}
	_, e = budget.Reserve(ctx, "two", 1)
	if e != nil {
		t.Fatal(e)
	}
	_, third := budget.Reserve(ctx, "three", 1)
	_, retry := budget.Reserve(ctx, "one", 1)
	// Assert.
	if !errors.Is(third, evaly.ErrBudget) || retry != nil || budget.Used() != 2 {
		t.Fatal(third, retry, budget.Used())
	}
	if e = budget.Reconcile(ctx, first, evaly.Usage{Known: true, Units: .5}); e != nil || budget.Used() != 1.5 {
		t.Fatal(e, budget.Used())
	}
	c := config(t, 4)
	var calls, clean atomic.Int32
	cancelctx, cancel := context.WithCancel(ctx)
	c.Plan.Concurrency = 1
	c.Target = evaly.TargetFunc[input, int, *int](
		func(ctx context.Context, i input, tc evaly.TrialContext[*int]) (evaly.TargetResult[int], error) {
			calls.Add(1)
			cancel()
			return evaly.TargetResult[int]{Output: 1}, nil
		},
	)
	life := c.Lifecycle.(evaly.LifecycleFuncs[*int])
	life.CleanupFunc = func(ctx context.Context, e *int) error {
		if ctx.Err() != nil {
			t.Error("cleanup inherits cancellation")
		}
		clean.Add(1)
		return nil
	}
	c.Lifecycle = life
	exp, e := evaly.Run(cancelctx, c)
	if e != nil {
		t.Fatal(e)
	}
	if calls.Load() != 1 || clean.Load() != 1 || exp.Record().Manifest.State != "incomplete" {
		t.Fatal(calls.Load(), clean.Load(), exp.Record())
	}
}
func TestComparisonCoverageAndCaseDenominator(t *testing.T) {
	// Arrange.
	b := config(t, 4)
	b.ID = "baseline"
	c := config(t, 4)
	c.ID = "candidate"
	life := c.Lifecycle.(evaly.LifecycleFuncs[*int])
	life.PrepareFunc = func(ctx context.Context, id string) (*int, error) {
		v := 0
		if strings.Contains(id, "/c/") || strings.Contains(id, "/d/") {
			return &v, errors.New("offline")
		}
		return &v, nil
	}
	c.Lifecycle = life
	p := evaly.GatePolicy{
		Revision:               "gate-v1",
		MinimumCoverage:        1,
		MinimumMatchedCases:    1,
		MinimumMatchedCoverage: 1,
		MinimumQuality:         .7,
		MaximumRegression:      .3,
		BootstrapSamples:       200,
		Seed:                   3,
	}
	// Act.
	baseline, e := evaly.Run(context.Background(), b)
	if e != nil {
		t.Fatal(e)
	}
	candidate, e := evaly.Run(context.Background(), c)
	if e != nil {
		t.Fatal(e)
	}
	comp, e := evaly.Compare(
		baseline,
		candidate,
		evaly.AssertionObjective{ID: "assertions", Revision: "v1", Policy: "all"},
		p,
	)
	// Assert.
	if e != nil || comp.Verdict != evaly.GateInconclusive || comp.CandidateAggregate.Scored != 2 ||
		comp.CandidateAggregate.Eligible != 4 ||
		comp.CandidateAggregate.SetupFailures != 2 {
		t.Fatal(e, comp)
	}
	c = config(t, 4)
	c.ID = "complete"
	orig := c.Target
	c.Target = evaly.TargetFunc[input, int, *int](
		func(ctx context.Context, i input, tc evaly.TrialContext[*int]) (evaly.TargetResult[int], error) {
			o, e := orig.Run(ctx, i, tc)
			if i.Numbers[0] == 3 {
				o.Output = 99
			}
			return o, e
		},
	)
	complete, e := evaly.Run(context.Background(), c)
	if e != nil {
		t.Fatal(e)
	}
	comp, e = evaly.Compare(
		baseline,
		complete,
		evaly.AssertionObjective{ID: "assertions", Revision: "v1", Policy: "all"},
		p,
	)
	if e != nil || comp.CandidateAggregate.Mean != .75 || comp.CandidateAggregate.Eligible != 4 {
		t.Fatal(e, comp)
	}
	b.Plan.Repeats = 3
	c.Plan.Repeats = 3
	b.ID = "repeated-baseline"
	c.ID = "repeated-candidate"
	be, e := evaly.Run(context.Background(), b)
	if e != nil {
		t.Fatal(e)
	}
	ce, e := evaly.Run(context.Background(), c)
	if e != nil {
		t.Fatal(e)
	}
	comp, e = evaly.Compare(be, ce, evaly.AssertionObjective{ID: "assertions", Revision: "v1", Policy: "all"}, p)
	if e != nil || comp.Uncertainty.Cases != 4 || comp.Unit != "case" {
		t.Fatal(e, comp)
	}
}
func TestArtifactReopenConflictCorruptionAndPartialPublication(t *testing.T) {
	// Arrange.
	dir := t.TempDir()
	store, e := evaly.OpenFileStore(dir)
	if e != nil {
		t.Fatal(e)
	}
	exp, e := evaly.Run(context.Background(), config(t, 2))
	if e != nil {
		t.Fatal(e)
	}
	// Act.
	if e = evaly.SaveExperiment(context.Background(), store, exp); e != nil {
		t.Fatal(e)
	}
	reopened, e := evaly.OpenFileStore(dir)
	if e != nil {
		t.Fatal(e)
	}
	got, e := evaly.LoadExperiment(context.Background(), reopened, exp.ID())
	if e != nil {
		t.Fatal(e)
	}
	// Assert.
	if got.Revision() != exp.Revision() {
		t.Fatal("changed revision")
	}
	if e = evaly.SaveExperiment(context.Background(), store, exp); e != nil {
		t.Fatal("non-idempotent", e)
	}
	conflict, _ := evaly.NewEnvelope("evidence", exp.ID(), map[string]string{"state": "sealed"})
	if e = store.Put(context.Background(), conflict); !errors.Is(e, evaly.ErrConflict) {
		t.Fatal(e)
	}
	store.Fault = func(stage string) error {
		if stage == "validated" {
			return errors.New("power loss")
		}
		return nil
	}
	partial, _ := evaly.NewEnvelope("evidence", "partial", map[string]string{"state": "sealed"})
	if e = store.Put(context.Background(), partial); e == nil {
		t.Fatal("fault ignored")
	}
	if _, e = store.Get(context.Background(), "partial"); !errors.Is(e, os.ErrNotExist) {
		t.Fatal("partial readable", e)
	}
	path := filepath.Join(dir, exp.ID()+".json")
	if e = os.WriteFile(path, []byte(`{"version":1}`), 0600); e != nil {
		t.Fatal(e)
	}
	if _, e = evaly.LoadExperiment(context.Background(), store, exp.ID()); e == nil {
		t.Fatal("corrupt baseline accepted")
	}
}
func TestExportDeliveryIsSeparateAndMappingLoss(t *testing.T) {
	// Arrange.
	exp, e := evaly.Run(context.Background(), config(t, 1))
	if e != nil {
		t.Fatal(e)
	}
	env, e := evaly.NewEnvelope("experiment", exp.ID(), exp.Record())
	if e != nil {
		t.Fatal(e)
	}
	record := evaly.DeliveryRecord{ObservationID: "observation-v1", Artifact: env}
	sink := &evaly.MemoryExport{Dedup: true, Fail: true}
	// Act.
	failed := evaly.Export(context.Background(), sink, record)
	sink.Fail = false
	delivered := evaly.Export(context.Background(), sink, record)
	evaly.Export(context.Background(), sink, record)
	// Assert.
	if failed.State != "failed" || delivered.State != "delivered" || len(sink.Records()) != 1 {
		t.Fatal(failed, delivered, sink.Records())
	}
	raw := &evaly.MemoryExport{}
	evaly.Export(context.Background(), raw, record)
	evaly.Export(context.Background(), raw, record)
	if len(raw.Records()) != 2 {
		t.Fatal("undeclared dedup")
	}
	loss := evaly.CheckMapping(evaly.InteropCapabilities{Version: 2})
	if loss.Supported || len(loss.Losses) != 6 {
		t.Fatal(loss)
	}
}
