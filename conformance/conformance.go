// Package conformance exposes behavioral contract suites for reference and host
// adapters. It is a test dependency, not a runtime core dependency.
package conformance

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
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

// Budget requires an atomic adapter with capacity exactly 2 units.
func Budget(t *testing.T, makeBudget func() (evaly.Budget, error)) {
	t.Helper()
	ctx := context.Background()
	b, e := makeBudget()
	if e != nil {
		t.Fatal(e)
	}
	var accepted atomic.Int32
	var wg sync.WaitGroup
	for _, id := range []string{"one", "two", "three"} {
		wg.Go(func() {
			r, eLocal := b.Reserve(ctx, id, 1)
			if eLocal == nil {
				accepted.Add(1)
				if eLocal = b.Reconcile(ctx, r, evaly.Usage{Known: false, Units: 0}); eLocal != nil {
					t.Error(eLocal)
				}
			} else if !errors.Is(eLocal, evaly.ErrBudget) {
				t.Error(eLocal)
			}
		})
	}
	wg.Wait()
	if accepted.Load() != 2 {
		t.Fatal("reservation oversubscription", accepted.Load())
	}
	if _, e = b.Reserve(ctx, "four", 1); !errors.Is(e, evaly.ErrBudget) {
		t.Fatal("unknown usage released", e)
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, e = b.Reserve(cancelled, "cancelled", 0); !errors.Is(e, context.Canceled) {
		t.Fatal(e)
	}
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
	ctx, cancel := context.WithTimeout(context.Background(), time.Nanosecond)
	defer cancel()
	<-ctx.Done()
	result := evaly.Assess(ctx, []evaly.Grader[I, O, R]{g}, factory)[0]
	if result.Status != evaly.GraderError || len(result.Metrics) != 0 {
		t.Fatal("timeout collapsed into score", result)
	}
}
func Lifecycle[E any](t *testing.T, l evaly.Lifecycle[E]) {
	t.Helper()
	ctx := context.Background()
	if l.Identity().Fixture == "" || l.Identity().Reset == "" {
		t.Fatal("unversioned fixture")
	}
	env, e := l.Prepare(ctx, "conformance-lifecycle")
	if e != nil {
		t.Fatal(e)
	}
	if e = l.Reset(ctx, env); e != nil {
		t.Fatal(e)
	}
	cleanupctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	if e = l.Cleanup(cleanupctx, env); e != nil {
		t.Fatal(e)
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
