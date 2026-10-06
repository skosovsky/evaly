package fixtures

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/skosovsky/evaly"
)

type Refund struct {
	Customer string `json:"customer"`
	Amount   int    `json:"amount"`
}
type CRMOutput struct {
	Text string `json:"text"`
}
type CRMEnvironment struct {
	Namespace string        `json:"Namespace"`
	Store     *OutcomeStore `json:"Store"`
}

// OutcomeStore is independent of the target's response text.
type OutcomeStore struct {
	mu      sync.Mutex
	refunds map[string]int
}

func (s *OutcomeStore) Set(namespace string, amount int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.refunds == nil {
		s.refunds = map[string]int{}
	}
	s.refunds[namespace] = amount
}
func (s *OutcomeStore) Get(namespace string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.refunds[namespace]
}
func (s *OutcomeStore) Delete(namespace string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.refunds, namespace)
}
func CRMConfig(id string, performRefund bool) (evaly.RunConfig[Refund, CRMOutput, int, *CRMEnvironment], error) {
	var zeroGraderRevision evaly.GraderRevision
	var zeroUsage evaly.Usage
	reference := 100
	d, e := (evaly.DatasetDraft[Refund, int]{Selection: assertionAll, Cases: []evaly.Case[Refund, int]{{ID: refundOperation, Input: Refund{"customer-1", 100}, Reference: &reference, RequiredEvidence: []string{outcomeEvidenceKind, toolEvidenceKind}, Revision: "", Metadata: nil, Generation: nil}}, ParentRevision: ""}).Seal(
		evaly.JSONCodec[Refund]{ID: refundOperation, Version: "1"},
		ReferenceCodec(),
	)
	if e != nil {
		return evaly.RunConfig[Refund, CRMOutput, int, *CRMEnvironment]{}, e
	}
	store := new(OutcomeStore)
	c := evaly.RunConfig[Refund, CRMOutput, int, *CRMEnvironment]{
		ID:          id,
		OutputCodec: evaly.JSONCodec[CRMOutput]{ID: "crm-output", Version: "1"},
		Dataset:     d,
		Plan: evaly.RunPlan{
			Repeats:         1,
			Concurrency:     1,
			Timeout:         time.Second,
			CleanupTimeout:  time.Second,
			MaxAttempts:     1,
			AssertionPolicy: assertionAll, Seed: 0, StopOnInfrastructure: false, DispatchUnits: 0, GraderUnits: 0,
		},
		Provenance: evaly.Provenance{
			Target: "crm-v1",
			Model:  "scripted",
			Prompt: "refund-v1",
			Tools:  "refund-v1",
			Policy: "approved-only-v1", Provider: nil, Unknown: nil,
		},
		Capture: evaly.CaptureConfig{
			Policy: evaly.FieldPolicy{
				ID: "crm-safe-v1",
				Allowed: map[string][]string{
					toolEvidenceKind:    {"action"},
					outcomeEvidenceKind: {"amount"},
				},
				KeepReferences: false,
			},
			KnownKinds:    []string{outcomeEvidenceKind, toolEvidenceKind},
			RequiredKinds: []string{outcomeEvidenceKind, toolEvidenceKind},
			MaxEvents:     workflowMaxEvents,
			MaxBytes:      fixtureMaxBytes,
		},
		ProjectionRevision: "crm-safe-v1",
		Target:             nil,
		Lifecycle:          nil,
		Graders:            nil,
		Project:            nil,
		Budget:             nil,
		CriticalEvidence:   false,
	}
	c.Lifecycle = evaly.LifecycleFuncs[*CRMEnvironment]{
		IdentityValue: evaly.LifecycleIdentity{Fixture: "crm-v1", Reset: "namespace-v1", Isolation: evaly.Isolated},
		PrepareFunc: func(ctx context.Context, namespace string) (*CRMEnvironment, error) {
			return &CRMEnvironment{Namespace: namespace, Store: store}, ctx.Err()
		},
		ResetFunc: func(ctx context.Context, e *CRMEnvironment) error { e.Store.Delete(e.Namespace); return ctx.Err() },
		CleanupFunc: func(ctx context.Context, e *CRMEnvironment) error {
			if e != nil {
				e.Store.Delete(e.Namespace)
			}
			return ctx.Err()
		},
	}
	c.Target = evaly.TargetFunc[Refund, CRMOutput, *CRMEnvironment](
		func(ctx context.Context, i Refund, t evaly.TrialContext[*CRMEnvironment]) (evaly.TargetResult[CRMOutput], error) {
			return crmTarget(ctx, i, t, &performRefund)
		},
	)
	c.Project = func(ctx context.Context, cs evaly.Case[Refund, int], o CRMOutput, e evaly.EvidenceRecord) (evaly.View[Refund, CRMOutput, int], error) {
		cs.Input.Customer = "redacted"
		return evaly.View[Refund, CRMOutput, int]{Case: cs, Output: o, Evidence: e}, ctx.Err()
	}
	c.Graders = []evaly.Grader[Refund, CRMOutput, int]{
		evaly.GraderFunc[Refund, CRMOutput, int]{
			Identity: evaly.GraderRevision{
				ID:             outcomeEvidenceKind,
				Implementation: goImplementationRevision,
				Rubric:         "refund-in-store-v1",
				Model:          "",
				Prompt:         "",
				Configuration:  "",
			},
			Evaluate: func(ctx context.Context, v evaly.View[Refund, CRMOutput, int]) (evaly.Grade, error) {
				return gradeRefundOutcome(ctx, v, &zeroGraderRevision, &zeroUsage)
			},
		},
	}
	return c, nil
}

func crmTarget(
	ctx context.Context,
	i Refund,
	t evaly.TrialContext[*CRMEnvironment],
	performRefund *bool,
) (evaly.TargetResult[CRMOutput], error) {
	if *performRefund {
		t.Environment.Store.Set(t.Environment.Namespace, i.Amount)
	}
	if e := t.Evidence.Record(
		ctx,
		evaly.Event{
			Version:       1,
			Sequence:      1,
			Kind:          toolEvidenceKind,
			CorrelationID: refundOperation,
			Payload:       json.RawMessage(`{"action":"refund","secret":"private-api-key"}`), References: nil,
		},
	); e != nil {
		return evaly.TargetResult[CRMOutput]{}, e
	}
	amount := t.Environment.Store.Get(t.Environment.Namespace)
	b, _ := json.Marshal(struct {
		Amount int `json:"amount"`
	}{amount})
	if e := t.Evidence.Record(
		ctx,
		evaly.Event{
			Version:       1,
			Sequence:      2,
			Kind:          outcomeEvidenceKind,
			CorrelationID: refundOperation,
			Payload:       b,
			References:    nil,
		},
	); e != nil {
		return evaly.TargetResult[CRMOutput]{}, e
	}
	return evaly.TargetResult[CRMOutput]{
		Output: CRMOutput{Text: "refund done"},
		Usage:  evaly.Usage{Known: true, Units: 0},
	}, nil
}

func gradeRefundOutcome(ctx context.Context, v evaly.View[Refund, CRMOutput, int], zeroGraderRevision *evaly.
	GraderRevision, zeroUsage *evaly.
	Usage) (evaly.Grade, error) {
	if !evaly.CompleteFor(v.Evidence, outcomeEvidenceKind) {
		return evaly.Grade{
			Status:       evaly.InsufficientEvidence,
			Reasons:      []string{"missing_outcome"},
			Dispatched:   false,
			Revision:     (*zeroGraderRevision),
			Metrics:      nil,
			Assertions:   nil,
			EvidenceRefs: nil,
			Usage:        (*zeroUsage),
		}, nil
	}
	amount := 0
	for _, event := range v.Evidence.Events {
		if event.Kind == outcomeEvidenceKind {
			var payload struct {
				Amount int `json:"amount"`
			}
			if e := json.Unmarshal(event.Payload, &payload); e != nil {
				return evaly.Grade{}, e
			}
			amount = payload.Amount
		}
	}
	return evaly.Grade{
		Status: evaly.Scored,
		Assertions: []evaly.Assertion{
			{
				Name:   "refund_exists",
				Pass:   v.Case.Reference != nil && amount == *v.Case.Reference,
				Reason: "",
			},
		},
		Usage: evaly.Usage{
			Known: true,
			Units: 0,
		},
		Dispatched:   false,
		Revision:     (*zeroGraderRevision),
		Metrics:      nil,
		Reasons:      nil,
		EvidenceRefs: nil,
	}, ctx.Err()
}
