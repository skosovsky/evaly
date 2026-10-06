// Package conformance exposes behavioral contract suites for reference and host
// adapters. It is a test dependency, not a runtime core dependency.
package conformance

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/skosovsky/evaly"
)

// Structural verifies repeatable local validation of configured and invalid ports.
// The host supplies capabilities with known validity, without invoking paid work.
func Structural(t *testing.T, valid evaly.StructuralValidator, invalid ...any) {
	t.Helper()
	for range 2 {
		if err := evaly.ValidatePort(valid); err != nil {
			t.Fatal("configured port rejected", err)
		}
		for _, port := range invalid {
			if err := evaly.ValidatePort(port); err == nil {
				t.Fatal("invalid port accepted")
			}
		}
	}
}

// Artifact checks publication, immutable identity, checksum rejection and reopen.
func Artifact(t *testing.T, open func() (evaly.ArtifactStore, error)) {
	t.Helper()
	ctx := context.Background()
	s, e := open()
	if e != nil {
		t.Fatal(e)
	}
	env, e := evaly.NewEnvelope("evidence", "conformance", map[string]string{valueState: "sealed"})
	if e != nil {
		t.Fatal(e)
	}
	t.Run("atomic_identity", func(t *testing.T) {
		var wg sync.WaitGroup
		errs := make(chan error, concurrentBudgetClaims)
		for range concurrentBudgetClaims {
			wg.Go(func() { ; errs <- s.Put(ctx, env) })
		}
		wg.Wait()
		close(errs)
		for e := range errs {
			if e != nil {
				t.Fatal(e)
			}
		}
		other, _ := evaly.NewEnvelope("evidence", env.ID, map[string]string{valueState: "incomplete"})
		if e = s.Put(ctx, other); !errors.Is(e, evaly.ErrConflict) {
			t.Fatal(e)
		}
	})
	t.Run("reopen", func(t *testing.T) {
		reopened, eLocal := open()
		if eLocal != nil {
			t.Fatal(eLocal)
		}
		got, eLocal := reopened.Get(ctx, env.ID)
		if eLocal != nil || got.Checksum != env.Checksum {
			t.Fatal(eLocal, got)
		}
	})
	t.Run("invalid_checksum", func(t *testing.T) {
		bad := env
		bad.ID = "bad"
		bad.Checksum = "bad"
		if e = s.Put(ctx, bad); !errors.Is(e, evaly.ErrCorrupt) {
			t.Fatal(e)
		}
	})
}

func Codec[T any](t *testing.T, c evaly.Codec[T], value T) {
	t.Helper()
	if c.Identity().ID == "" || c.Identity().Version == "" {
		t.Fatal("missing codec identity")
	}
	bytes, e := c.Encode(value)
	if e != nil {
		t.Fatal(e)
	}
	canonical, e := evaly.CanonicalJSON(bytes)
	if e != nil || string(bytes) != string(canonical) {
		t.Fatal("noncanonical encoding", e)
	}
	restored, e := c.Decode(bytes)
	if e != nil {
		t.Fatal(e)
	}
	again, e := c.Encode(restored)
	if e != nil || string(again) != string(bytes) {
		t.Fatal("non-reversible codec", e)
	}
	if _, e = c.Decode([]byte(`{"a":1,"a":2}`)); e == nil {
		t.Fatal("duplicate-key decode")
	}
}
func Target[I, O, R, E any](t *testing.T, c evaly.RunConfig[I, O, R, E]) {
	t.Helper()
	experiment, e := evaly.Run(context.Background(), c)
	if e != nil {
		t.Fatal(e)
	}
	for _, trial := range experiment.Record().Trials {
		if trial.Status != evaly.Completed || trial.Cleanup.State != "completed" {
			t.Fatal(trial)
		}
		pass, known := evaly.AssertionOutcome(trial.Grades, c.Plan.AssertionPolicy)
		if !known || !pass {
			t.Fatal("known fixture did not pass", trial)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	cancelled, e := evaly.Run(ctx, c)
	if e != nil {
		t.Fatal(e)
	}
	for _, trial := range cancelled.Record().Trials {
		if trial.Status != evaly.Cancelled {
			t.Fatal("dispatch after cancellation", trial)
		}
	}
}
func Grader[I, O, R any](t *testing.T, g evaly.Grader[I, O, R], factory func() (evaly.View[I, O, R], error)) {
	t.Helper()
	grades := evaly.Assess(context.Background(), []evaly.Grader[I, O, R]{g}, factory)
	if len(grades) != 1 || evaly.ValidateGrade(grades[0]) != nil || grades[0].Revision != g.Revision() {
		t.Fatal(grades)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	result := evaly.Assess(ctx, []evaly.Grader[I, O, R]{g}, factory)[0]
	if result.Status != evaly.GraderError || len(result.Metrics) != 0 {
		t.Fatal("pre-cancellation collapsed into score", result)
	}
}

// Lifecycle uses local test defaults; external hosts should choose their own ID
// and cooperative operation/cleanup deadlines with LifecycleWithOptions.
func Lifecycle[E any](t *testing.T, l evaly.Lifecycle[E]) {
	t.Helper()
	LifecycleWithOptions(
		t,
		l,
		LifecycleOptions{ID: "conformance-lifecycle", Timeout: time.Second, CleanupTimeout: time.Second},
	)
}

// LifecycleOptions belongs to the test fixture, not a runtime execution policy.
type LifecycleOptions struct {
	ID             string
	Timeout        time.Duration
	CleanupTimeout time.Duration
}

// LifecycleWithOptions owns every returned Prepare handle, including partial
// failure handles. Cleanup runs once at test exit with a detached bounded context.
// A callback must cooperate with cancellation; this suite does not forcibly abort it.
func LifecycleWithOptions[E any](t *testing.T, l evaly.Lifecycle[E], options LifecycleOptions) {
	t.Helper()
	if options.ID == "" || options.Timeout <= 0 || options.CleanupTimeout <= 0 {
		t.Fatal("invalid lifecycle test options")
	}
	if err := evaly.ValidatePort(l); err != nil {
		t.Fatal("invalid lifecycle", err)
	}
	if l.Identity().Fixture == "" || l.Identity().Reset == "" {
		t.Fatal("unversioned fixture")
	}
	ctx, cancel := context.WithTimeout(context.Background(), options.Timeout)
	defer cancel()
	env, err := l.Prepare(ctx, options.ID)
	t.Cleanup(func() {
		cleanupctx, done := context.WithTimeout(context.WithoutCancel(ctx), options.CleanupTimeout)
		defer done()
		if cleanupErr := l.Cleanup(cleanupctx, env); cleanupErr != nil {
			t.Error("cleanup failure", cleanupErr)
		}
	})
	if err != nil {
		t.Fatal("prepare failure", err)
	}
	if err = l.Reset(ctx, env); err != nil {
		t.Fatal("reset failure", err)
	}
}
func Export(t *testing.T, s evaly.ExportSink, count func() int) {
	t.Helper()
	env, e := evaly.NewEnvelope("evidence", "export-conformance", map[string]string{valueState: "sealed"})
	if e != nil {
		t.Fatal(e)
	}
	r := evaly.DeliveryRecord{ObservationID: "same-observation", Artifact: env}
	for range 2 {
		d := evaly.Export(context.Background(), s, r)
		if d.State != "delivered" {
			t.Fatal(d)
		}
	}
	expected := 2
	if s.Capabilities().Deduplication {
		expected = 1
	}
	if count() != expected {
		t.Fatal("advertised dedup mismatch", count(), expected)
	}
	// Arrange: a cancelled delivery must not reach the sink or alter source data.
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	// Act / Assert: cancellation is classified separately from transient delivery.
	delivery := evaly.Export(cancelled, s, r)
	if delivery.State != failedState || delivery.Reason != "cancelled" || count() != expected {
		t.Fatal("cancelled delivery dispatched or misclassified", delivery, count())
	}
	if s.Capabilities().Deduplication {
		// Arrange: the same delivery identity now carries different artifact bytes.
		other, err := evaly.NewEnvelope("evidence", "export-conformance", map[string]string{valueState: "incomplete"})
		if err != nil {
			t.Fatal(err)
		}
		// Act / Assert: a conflict cannot be treated as a temporary sink outage.
		conflict := evaly.Export(
			context.Background(),
			s,
			evaly.DeliveryRecord{ObservationID: r.ObservationID, Artifact: other},
		)
		if conflict.State != failedState || conflict.Reason != "conflict" || count() != expected {
			t.Fatal("immutable delivery identity conflict not preserved", conflict, count())
		}
	}
}
