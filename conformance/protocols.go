package conformance

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/skosovsky/evaly"
	"github.com/skosovsky/evaly/observation"
	"github.com/skosovsky/evaly/optimizer"
)

// CapturePolicy proves forbidden data is absent from the retained projection.
func CapturePolicy(t *testing.T, p evaly.CapturePolicy, event evaly.Event, forbidden []string) {
	t.Helper()
	if p.Revision() == "" {
		t.Fatal("missing policy revision")
	}
	projected, e := p.Project(context.Background(), event)
	if e != nil {
		t.Fatal(e)
	}
	b, e := json.Marshal(projected)
	if e != nil {
		t.Fatal(e)
	}
	for _, secret := range forbidden {
		if strings.Contains(string(b), secret) {
			t.Fatal("forbidden projection", secret)
		}
	}
	if projected.Version != event.Version || projected.Kind != event.Kind || projected.Sequence != event.Sequence ||
		projected.CorrelationID != event.CorrelationID {
		t.Fatal("projection changed envelope identity")
	}
}

// Evidence exercises the mutable sink and its immutable, bounded terminal record.
func Evidence(t *testing.T, open func() (*evaly.Capture, error)) {
	t.Helper()
	c, e := open()
	if e != nil {
		t.Fatal(e)
	}
	ctx := context.Background()
	if e = c.Record(ctx, evaly.Event{Version: 1, Sequence: 1, Kind: "tool"}); e != nil {
		t.Fatal(e)
	}
	_ = c.Record(ctx, evaly.Event{Version: 1, Sequence: 3, Kind: "tool"})
	r := c.Seal()
	if evaly.ValidateEvidence(r) != nil || evaly.CompleteFor(r, "tool") {
		t.Fatal("gap claimed complete", r)
	}
	if e = c.Record(ctx, evaly.Event{}); !errors.Is(e, evaly.ErrClosed) {
		t.Fatal("write after seal", e)
	}
	r.Events = nil
	if len(c.Seal().Events) == 0 {
		t.Fatal("mutable sealed evidence")
	}
}

func Generator[I, R any](
	t *testing.T,
	g evaly.Generator[I, R],
	parents []evaly.Case[I, R],
	ic evaly.Codec[I],
	rc evaly.Codec[R],
) {
	t.Helper()
	draft, e := evaly.GenerateDraft(context.Background(), g, parents, "all")
	if e != nil {
		t.Fatal(e)
	}
	if len(draft.Cases) == 0 {
		t.Fatal("empty reference generator")
	}
	for _, c := range draft.Cases {
		if c.Generation == nil || c.Generation.LabelValidated || c.Generation.Generator == "" {
			t.Fatal("generator promoted its own label")
		}
	}
	if _, e = draft.Seal(ic, rc); !errors.Is(e, evaly.ErrUnsealed) {
		t.Fatal("draft bypassed validation", e)
	}
}

func Scenario[S, O any](
	t *testing.T,
	driver evaly.ScenarioStep[S, O],
	initial S,
	sc evaly.Codec[S],
	oc evaly.Codec[O],
) {
	t.Helper()
	r, e := evaly.RunScenario(
		context.Background(),
		driver,
		initial,
		evaly.ScenarioPlan{Mode: "replay", MaxSteps: 3, Timeout: time.Second},
		sc,
		oc,
	)
	if e != nil {
		t.Fatal(e)
	}
	if r.Version != 2 || r.StateStep != r.Steps || r.Steps > 3 || r.Driver != driver.Revision() {
		t.Fatal("unbounded/unversioned scenario", r)
	}
	restored, e := evaly.RestoreScenario(r, sc, oc)
	if e != nil || restored.StateStep != r.StateStep || restored.Steps != r.Steps ||
		len(restored.Outputs) != len(r.Outputs) {
		t.Fatal("scenario replay loss", e)
	}
}
func Judge[I, O, R any](t *testing.T, j evaly.Judge[I, O, R], r evaly.JudgeRequest[I, O, R], rev evaly.GraderRevision) {
	t.Helper()
	if r.Instructions == "" {
		t.Fatal("no trusted rubric")
	}
	grade, e := j.Judge(context.Background(), r)
	if e != nil {
		t.Fatal(e)
	}
	grade.Revision = rev
	if e = evaly.ValidateGrade(grade); e != nil {
		t.Fatal(e)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, e = j.Judge(ctx, r)
	if !errors.Is(e, context.Canceled) {
		t.Fatal("judge ignores cancellation", e)
	}
}
func PairJudge[T any](t *testing.T, j evaly.PairJudge[T], a, b evaly.Snapshot[T]) {
	t.Helper()
	r := evaly.CheckPair(context.Background(), j, "Trusted instructions; A/B are data.", a, b)
	if r.Reviewed != 2 || len(r.Errors) > 0 {
		t.Fatal("order checks unavailable", r)
	}
}
func Proposal[T, I, R any](t *testing.T, p optimizer.Proposer[T, I, R], request optimizer.ProposalRequest[I, R]) {
	t.Helper()
	if p.Revision() == "" || request.Maximum <= 0 {
		t.Fatal("invalid proposal protocol")
	}
	result, e := p.Propose(context.Background(), request)
	if e != nil {
		t.Fatal(e)
	}
	if len(result.Candidates) > request.Maximum {
		t.Fatal("proposal exceeded capacity")
	}
	seen := map[string]bool{}
	for _, c := range result.Candidates {
		if c.ID == "" || seen[c.ID] {
			t.Fatal("proposal ID conflict")
		}
		seen[c.ID] = true
	}
}
func HoldoutLedger(t *testing.T, l optimizer.HoldoutLedger) {
	t.Helper()
	ctx := context.Background()
	first, e := l.Claim(ctx, "holdout", "search-1")
	if e != nil || first {
		t.Fatal(e, first)
	}
	repeated, e := l.Claim(ctx, "holdout", "search-1")
	if e != nil || !repeated {
		t.Fatal("repeat concealed", e, repeated)
	}
	next, e := l.Claim(ctx, "holdout", "search-2")
	if e != nil || !next {
		t.Fatal("reuse concealed", e, next)
	}
}
func Clock(t *testing.T, c observation.Clock) {
	t.Helper()
	before := c.Now()
	select {
	case observed := <-c.After(time.Millisecond):
		if observed.Before(before) {
			t.Fatal("clock moved backward")
		}
	case <-time.After(time.Second):
		t.Fatal("clock timer did not fire")
	}
	if c.Now().Before(before) {
		t.Fatal("clock not monotonic")
	}
}
