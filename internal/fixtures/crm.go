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
	Namespace string
	Store     *OutcomeStore
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
	reference := 100
	d, e := (evaly.DatasetDraft[Refund, int]{Selection: "all", Cases: []evaly.Case[Refund, int]{{ID: "refund", Input: Refund{"customer-1", 100}, Reference: &reference, RequiredEvidence: []string{"outcome", "tool"}}}}).Seal(
		evaly.JSONCodec[Refund]{ID: "refund", Version: "1"},
		ReferenceCodec(),
	)
	if e != nil {
		return evaly.RunConfig[Refund, CRMOutput, int, *CRMEnvironment]{}, e
	}
	store := &OutcomeStore{}
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
			AssertionPolicy: "all",
		},
		Provenance: evaly.Provenance{
			Target: "crm-v1",
			Model:  "scripted",
			Prompt: "refund-v1",
			Tools:  "refund-v1",
			Policy: "approved-only-v1",
		},
		Capture: evaly.CaptureConfig{
			Policy: evaly.FieldPolicy{
				ID:      "crm-safe-v1",
				Allowed: map[string][]string{"tool": {"action"}, "outcome": {"amount"}},
			},
			KnownKinds:    []string{"outcome", "tool"},
			RequiredKinds: []string{"outcome", "tool"},
			MaxEvents:     10,
			MaxBytes:      4096,
		},
		ProjectionRevision: "crm-safe-v1",
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
			if performRefund {
				t.Environment.Store.Set(t.Environment.Namespace, i.Amount)
			}
			if e := t.Evidence.Record(
				ctx,
				evaly.Event{
					Version:       1,
					Sequence:      1,
					Kind:          "tool",
					CorrelationID: "refund",
					Payload:       json.RawMessage(`{"action":"refund","secret":"private-api-key"}`),
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
				evaly.Event{Version: 1, Sequence: 2, Kind: "outcome", CorrelationID: "refund", Payload: b},
			); e != nil {
				return evaly.TargetResult[CRMOutput]{}, e
			}
			return evaly.TargetResult[CRMOutput]{
				Output: CRMOutput{Text: "refund done"},
				Usage:  evaly.Usage{Known: true},
			}, nil
		},
	)
	c.Project = func(ctx context.Context, cs evaly.Case[Refund, int], o CRMOutput, e evaly.EvidenceRecord) (evaly.View[Refund, CRMOutput, int], error) {
		cs.Input.Customer = "redacted"
		return evaly.View[Refund, CRMOutput, int]{Case: cs, Output: o, Evidence: e}, ctx.Err()
	}
	c.Graders = []evaly.Grader[Refund, CRMOutput, int]{
		evaly.GraderFunc[Refund, CRMOutput, int]{
			Identity: evaly.GraderRevision{ID: "outcome", Implementation: "go-v1", Rubric: "refund-in-store-v1"},
			Evaluate: func(ctx context.Context, v evaly.View[Refund, CRMOutput, int]) (evaly.Grade, error) {
				if !evaly.CompleteFor(v.Evidence, "outcome") {
					return evaly.Grade{Status: evaly.InsufficientEvidence, Reasons: []string{"missing_outcome"}}, nil
				}
				amount := 0
				for _, event := range v.Evidence.Events {
					if event.Kind == "outcome" {
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
						{Name: "refund_exists", Pass: v.Case.Reference != nil && amount == *v.Case.Reference},
					},
					Usage: evaly.Usage{Known: true},
				}, ctx.Err()
			},
		},
	}
	return c, nil
}
