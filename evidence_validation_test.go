package evaly

import (
	"context"
	"testing"
	"time"
)

func rehashEvidence(r EvidenceRecord) EvidenceRecord {
	r.Revision = ""
	b, _ := canonical(r)
	r.Revision = digest(b)
	return r
}

func TestRestoredEvidenceRejectsFalseCompleteness(t *testing.T) {
	// Arrange: portable evidence has a valid content hash but contradictory semantic claims.
	for _, variant := range []string{"gap", "error", "event_version", "event_kind", "sequence", "reference"} {
		t.Run(variant, func(t *testing.T) {
			r := EvidenceRecord{
				Version:         1,
				State:           "sealed",
				Policy:          "p1",
				Coverage:        map[string]bool{"tool": true},
				Events:          []Event{},
				ReplayAvailable: true,
			}
			switch variant {
			case "gap":
				r.Gaps = []int{1}
			case "error":
				r.Errors = []string{"capture_error"}
			case "event_version":
				r.Events = []Event{{Version: 2, Sequence: 1, Kind: "tool"}}
			case "event_kind":
				r.Events = []Event{{Version: 1, Sequence: 1, Kind: "unknown_required"}}
			case "sequence":
				r.Events = []Event{{Version: 1, Sequence: 3, Kind: "tool"}}
			case "reference":
				r.Events = []Event{
					{Version: 1, Sequence: 1, Kind: "tool", References: []string{"https://user:secret@host/trace"}},
				}
			}
			r = rehashEvidence(r)
			// Act.
			err := ValidateEvidence(r)
			g := AbsenceGrader[int, int, int](
				GraderRevision{ID: "absence", Implementation: "1", Rubric: "1"},
				"tool",
				func(Event) bool { return false },
			)
			grade := Assess(context.Background(), []Grader[int, int, int]{g}, func() (View[int, int, int], error) { return View[int, int, int]{Evidence: r}, nil })[0]
			// Assert: checksums cannot confer validity on malformed capture claims.
			if err == nil || grade.Status == Scored {
				t.Fatalf("malformed %s accepted: validate=%v grade=%+v", variant, err, grade)
			}
		})
	}
}

func TestCaptureDiagnosticsAreBounded(t *testing.T) {
	// Arrange.
	c, err := NewCapture(
		CaptureConfig{Policy: FieldPolicy{ID: "drop"}, KnownKinds: []string{"tool"}, MaxEvents: 1, MaxBytes: 512},
	)
	if err != nil {
		t.Fatal(err)
	}
	// Act: a target keeps reporting rejected events after retained capacity is exhausted.
	for i := 1; i <= 10000; i++ {
		_ = c.Record(context.Background(), Event{Version: 1, Sequence: i, Kind: "tool"})
	}
	r := c.Seal()
	// Assert: bounded capture includes diagnostics as well as event payloads.
	if len(r.Errors) > 64 || len(r.Gaps) > 64 {
		t.Fatalf("unbounded diagnostics: errors=%d gaps=%d", len(r.Errors), len(r.Gaps))
	}
}

type deadlineStep struct{}

func (deadlineStep) Revision() string { return "deadline-step-v1" }
func (deadlineStep) Step(ctx context.Context, s int, execution ScenarioContext) (int, int, bool, error) {
	<-ctx.Done()
	return s + 1, 1, true, nil
}
func TestScenarioFinalStepCannotHideDeadline(t *testing.T) {
	// Arrange / Act: callback acknowledges cancellation but returns a final answer.
	r, err := Drive(context.Background(), deadlineStep{}, 0, 1, time.Millisecond)
	// Assert.
	if err == nil || r.Stop == "completed" {
		t.Fatalf("deadline hidden: stop=%q error=%v", r.Stop, err)
	}
}

func TestDatasetRestoreChecksCaseRevision(t *testing.T) {
	// Arrange.
	ic, rc := JSONCodec[int]{ID: "i", Version: "1"}, JSONCodec[int]{ID: "r", Version: "1"}
	d, err := (DatasetDraft[int, int]{Selection: "all", Cases: []Case[int, int]{{ID: "c", Input: 1}}}).Seal(ic, rc)
	if err != nil {
		t.Fatal(err)
	}
	r := d.Record()
	r.Cases[0].Revision = "corrupted"
	// Act.
	_, err = RestoreDataset(r, ic, rc)
	// Assert.
	if err == nil {
		t.Fatal("corrupted stored case revision was silently repaired instead of rejected")
	}
}

func TestUnknownReconciliationCannotReleaseDispatchedLiability(t *testing.T) {
	// Arrange: caller explicitly reports actual usage unknown after external dispatch.
	budget, _ := NewMemoryBudget(1)
	reservation, _ := budget.Reserve(context.Background(), "paid", 1)
	if err := budget.Reconcile(context.Background(), reservation, Usage{}); err != nil {
		t.Fatal(err)
	}
	// Act.
	err := budget.Release(context.Background(), reservation)
	// Assert: Release is for undispatched reservations, not unknown completed effects.
	if err == nil || budget.Used() != 1 {
		t.Fatalf("unknown dispatched liability released: err=%v used=%v", err, budget.Used())
	}
}

func TestAssessNilGraderReturnsTypedError(t *testing.T) {
	// Arrange: public grader protocol must handle invalid grader capabilities.
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("nil grader panics instead of grader_error: %v", r)
		}
	}()
	// Act.
	grades := Assess(
		context.Background(),
		[]Grader[int, int, int]{nil},
		func() (View[int, int, int], error) { return View[int, int, int]{}, nil },
	)
	// Assert.
	if len(grades) != 1 || grades[0].Status != GraderError {
		t.Fatalf("invalid grader outcome: %+v", grades)
	}
}

func TestGenerationRejectsUnknownMode(t *testing.T) {
	// Arrange: generation modes are a versioned enum, not arbitrary text.
	ic, rc := JSONCodec[int]{ID: "i", Version: "1"}, JSONCodec[int]{ID: "r", Version: "1"}
	draft := DatasetDraft[int, int]{
		Selection: "all",
		Cases: []Case[int, int]{
			{
				ID:         "generated",
				Input:      1,
				Generation: &Generation{Generator: "g1", Mode: "unknown", LabelValidated: true},
			},
		},
	}
	// Act.
	_, err := draft.Seal(ic, rc)
	// Assert.
	if err == nil {
		t.Fatal("unknown generation mode sealed")
	}
}
