package conformance

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/skosovsky/evaly"
	"github.com/skosovsky/evaly/optimizer"
)

// TargetFailure identifies a host-injected failure; no domain model is prescribed.
type TargetFailure string

const (
	AfterEffect        TargetFailure = "failure_after_effect"
	IncompleteDelivery TargetFailure = "incomplete_delivery"
)

type TargetFault[I, O, R, E any] struct {
	Config        evaly.RunConfig[I, O, R, E]               `json:"Config"`
	ExpectedUsage evaly.Usage                               `json:"ExpectedUsage"`
	Kind          string                                    `json:"Kind"`
	Verify        func(*testing.T, evaly.Experiment, error) `json:"Verify"`
}

// TargetFailures requires fresh adapters and mandatory external-effect verification.
func TargetFailures[I, O, R, E any](t *testing.T, factory func(TargetFailure) (TargetFault[I, O, R, E], error)) {
	t.Helper()
	for _, mode := range []TargetFailure{AfterEffect, IncompleteDelivery} {
		t.Run(string(mode), func(t *testing.T) {
			// Arrange: the host owns the injected fault and domain evidence.
			fault, err := factory(mode)
			if err != nil || fault.Verify == nil || fault.Kind == "" {
				t.Fatal("invalid host fault factory", err)
			}
			// Act.
			experiment, err := evaly.Run(context.Background(), fault.Config)
			// Assert: partial evidence and usage cannot collapse into target success.
			trials := experiment.Record().Trials
			if len(trials) == 0 {
				t.Fatal("fault produced no retained trials", err)
			}
			verifyTargetTrials(t, mode, fault, trials, err)
			fault.Verify(t, experiment, err)
		})
	}
}

type GraderFault[I, O, R any] struct {
	Grader        evaly.Grader[I, O, R]               `json:"Grader"`
	View          func() (evaly.View[I, O, R], error) `json:"View"`
	ExpectedUsage evaly.Usage                         `json:"ExpectedUsage"`
}

func GraderFailures[I, O, R any](t *testing.T, factory func() GraderFault[I, O, R]) {
	t.Helper()
	// Arrange.
	fault := factory()
	// Act.
	grades := evaly.Assess(context.Background(), []evaly.Grader[I, O, R]{fault.Grader}, fault.View)
	// Assert.
	if len(grades) != 1 {
		t.Fatal(grades)
	}
	grade := grades[0]
	if !grade.Dispatched || grade.Status != evaly.GraderError || grade.Usage != fault.ExpectedUsage ||
		len(grade.Assertions) != 0 ||
		len(grade.Metrics) != 0 ||
		evaly.ValidateGrade(grade) != nil {
		t.Fatal("partial grader response promoted or usage lost", grade)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	cancelled := evaly.Assess(ctx, []evaly.Grader[I, O, R]{fault.Grader}, fault.View)[0]
	if cancelled.Dispatched || cancelled.Status != evaly.GraderError ||
		cancelled.Usage != (evaly.Usage{Known: false, Units: 0}) {
		t.Fatal("cancelled grader dispatched", cancelled)
	}
}

type PairFault[T any] struct {
	Judge           evaly.PairJudge[T] `json:"Judge"`
	A               evaly.Snapshot[T]  `json:"A"`
	B               evaly.Snapshot[T]  `json:"B"`
	ExpectedForward evaly.Usage        `json:"ExpectedForward"`
	ExpectedReverse evaly.Usage        `json:"ExpectedReverse"`
}

func PairJudgeFailures[T any](t *testing.T, factory func() PairFault[T]) {
	t.Helper()
	// Arrange.
	fault := factory()
	// Act.
	result := evaly.CheckPair(context.Background(), fault.Judge, "Trusted host rubric", fault.A, fault.B)
	// Assert: one failed order is not a two-order judgment.
	if result.Reviewed != 1 || len(result.Errors) != 1 || !result.ForwardDispatched || !result.ReverseDispatched ||
		result.Reverse.Preferred != "" ||
		result.Disagreement ||
		result.Forward.Usage != fault.ExpectedForward ||
		result.Reverse.Usage != fault.ExpectedReverse {
		t.Fatal("partial pair judgment lost or promoted", result)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	cancelled := evaly.CheckPair(ctx, fault.Judge, "Trusted host rubric", fault.A, fault.B)
	if cancelled.ForwardDispatched || cancelled.ReverseDispatched || cancelled.Reviewed != 0 {
		t.Fatal("cancelled pair dispatched", cancelled)
	}
}

// EvidenceFailures exercises a host-selected capture policy and delivery-fault signal.
func EvidenceFailures(t *testing.T, factory func() (*evaly.Capture, evaly.Event, error)) {
	t.Helper()
	// Arrange.
	capture, event, err := factory()
	if err != nil {
		t.Fatal(err)
	}
	// Act: retain an event, then lose delivery, then attempt a cancelled write.
	if err = capture.Record(context.Background(), event); err != nil {
		t.Fatal(err)
	}
	capture.MarkIncomplete("host_delivery_failure")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	late := event
	late.Sequence++
	if err = capture.Record(ctx, late); !errors.Is(err, context.Canceled) {
		t.Fatal("cancelled evidence accepted", err)
	}
	record := capture.Seal()
	// Assert.
	if evaly.ValidateEvidence(record) != nil || evaly.CompleteFor(record, event.Kind) || len(record.Events) != 1 {
		t.Fatal("partial evidence erased or claimed complete", record)
	}
}

type ExportFault struct {
	Sink   evaly.ExportSink     `json:"Sink"`
	Record evaly.DeliveryRecord `json:"Record"`
	Verify func(*testing.T)     `json:"Verify"`
}

func ExportFailures(t *testing.T, factory func() ExportFault) {
	t.Helper()
	// Arrange.
	fault := factory()
	if fault.Verify == nil {
		t.Fatal("external delivery verification missing")
	}
	encoded, err := json.Marshal(fault.Record.Artifact)
	if err != nil {
		t.Fatal(err)
	}
	var before evaly.Envelope
	if err := json.Unmarshal(encoded, &before); err != nil {
		t.Fatal(err)
	}
	// Act.
	delivery := evaly.Export(context.Background(), fault.Sink, fault.Record)
	// Assert.
	if delivery.State != failedState || delivery.Reason != "delivery_failure" ||
		!reflect.DeepEqual(before, fault.Record.Artifact) {
		t.Fatal("post-effect failure or source ownership lost", delivery)
	}
	fault.Verify(t)
}

type ProposalFault[T, I, R any] struct {
	Proposer           optimizer.Proposer[T, I, R]                          `json:"Proposer"`
	Request            optimizer.ProposalRequest[I, R]                      `json:"Request"`
	ExpectedUsage      evaly.Usage                                          `json:"ExpectedUsage"`
	ExpectedCandidates int                                                  `json:"ExpectedCandidates"`
	Verify             func(*testing.T, optimizer.ProposalResult[T], error) `json:"Verify"`
}

func ProposalFailures[T, I, R any](t *testing.T, factory func() ProposalFault[T, I, R]) {
	t.Helper()
	// Arrange.
	fault := factory()
	if fault.Verify == nil || evaly.ValidatePort(fault.Proposer) != nil {
		t.Fatal("invalid proposal factory")
	}
	// Act.
	result, err := fault.Proposer.Propose(context.Background(), fault.Request)
	// Assert: a paid failure remains distinct from an empty successful proposal.
	if err == nil || result.Usage != fault.ExpectedUsage || len(result.Candidates) != fault.ExpectedCandidates {
		t.Fatal("partial proposal lost", result, err)
	}
	fault.Verify(t, result, err)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, cancelErr := fault.Proposer.Propose(ctx, fault.Request); !errors.Is(cancelErr, context.Canceled) {
		t.Fatal("proposal ignores cancellation", cancelErr)
	}
	fault.Verify(t, result, err)
}

func verifyTargetTrials[I, O, R, E any](
	t *testing.T,
	mode TargetFailure,
	fault TargetFault[I, O, R, E],
	trials []evaly.TrialRecord,
	err error,
) {
	t.Helper()
	for _, trial := range trials {
		if trial.Status != evaly.TargetError || trial.TargetUsage != fault.ExpectedUsage ||
			trial.Cleanup.State != "completed" {
			t.Fatal("failure accounting or cleanup lost", trial, err)
		}
		if mode == AfterEffect && len(trial.Evidence.Events) == 0 {
			t.Fatal("effect evidence lost", trial)
		}
		if mode == IncompleteDelivery && evaly.CompleteFor(trial.Evidence, fault.Kind) {
			t.Fatal("delivery failure proves absence", trial)
		}
	}
}
