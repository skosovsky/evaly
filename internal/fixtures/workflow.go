package fixtures

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/skosovsky/evaly"
)

type WorkflowInput struct {
	Account string `json:"account"`
	Amount  int    `json:"amount"`
}
type WorkflowOutput struct {
	Text string `json:"text"`
}
type WorkflowState struct {
	Eligible bool `json:"eligible"`
	Refunded int  `json:"refunded"`
	Credited int  `json:"credited"`
}
type WorkflowEnvironment struct {
	Namespace string
	Store     *WorkflowStore
}

// WorkflowStore is host-owned external state, not a target response projection.
// Audit survives cleanup solely for deterministic fixture verification.
type WorkflowStore struct {
	mu                       sync.Mutex
	active                   map[string]WorkflowState
	audit                    map[string]WorkflowState
	calls, prepared, cleaned int
}

func (s *WorkflowStore) Prepare(ns string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.active == nil {
		s.active = map[string]WorkflowState{}
		s.audit = map[string]WorkflowState{}
	}
	s.active[ns] = WorkflowState{Eligible: true}
	s.prepared++
}
func (s *WorkflowStore) Reset(ns string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.active[ns] = WorkflowState{Eligible: true}
}
func (s *WorkflowStore) Read(ns string) WorkflowState {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.active[ns]
}
func (s *WorkflowStore) Apply(ns, mode string, amount int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v := s.active[ns]
	if mode == "alternative" {
		v.Credited += amount
	} else {
		v.Refunded += amount
	}
	s.active[ns] = v
	s.audit[ns] = v
}
func (s *WorkflowStore) Cleanup(ns string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.active, ns)
	s.cleaned++
}
func (s *WorkflowStore) Snapshot() (map[string]WorkflowState, int, int, int, int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := map[string]WorkflowState{}
	for k, v := range s.audit {
		out[k] = v
	}
	return out, len(s.active), s.calls, s.prepared, s.cleaned
}
func WorkflowInputCodec() evaly.JSONCodec[WorkflowInput] {
	return evaly.JSONCodec[WorkflowInput]{ID: "workflow-input", Version: "1"}
}
func WorkflowOutputCodec() evaly.JSONCodec[WorkflowOutput] {
	return evaly.JSONCodec[WorkflowOutput]{ID: "workflow-output", Version: "1"}
}
func WorkflowReferenceCodec() evaly.JSONCodec[int] {
	return evaly.JSONCodec[int]{ID: "workflow-reference", Version: "1"}
}

// WorkflowInvoke is the same host behavior used by both transport adapters.
func WorkflowInvoke(
	ctx context.Context,
	input WorkflowInput,
	ns string,
	store *WorkflowStore,
	mode string,
) (evaly.TargetResult[WorkflowOutput], []evaly.Event, error) {
	result := evaly.TargetResult[WorkflowOutput]{
		Output: WorkflowOutput{Text: "Refund definitely completed"},
		Usage:  evaly.Usage{Known: true},
	}
	if err := ctx.Err(); err != nil {
		return result, nil, err
	}
	store.mu.Lock()
	store.calls++
	store.mu.Unlock()
	var events []evaly.Event
	record := func(kind string, payload any) {
		b, _ := json.Marshal(payload)
		events = append(
			events,
			evaly.Event{Version: 1, Sequence: len(events) + 1, Kind: kind, CorrelationID: "account-action", Payload: b},
		)
	}
	var targetErr error
	if mode != "text-only" {
		result.Usage.Units++
		record("tool", struct {
			Action  string `json:"action"`
			Success bool   `json:"success"`
		}{"lookup", mode != "tool-error"})
		if mode == "tool-error" {
			targetErr = errors.New("controlled lookup error")
		} else if store.Read(ns).Eligible {
			action := "refund"
			if mode == "alternative" {
				action = "credit"
			}
			store.Apply(ns, mode, input.Amount)
			result.Usage.Units++
			record("tool", struct {
				Action  string `json:"action"`
				Success bool   `json:"success"`
			}{action, true})
			if mode == "effect-error" {
				targetErr = errors.New("controlled error after effect")
			}
		}
	}
	record("outcome", store.Read(ns))
	return result, events, targetErr
}

func WorkflowGraders(rubric string) []evaly.Grader[WorkflowInput, WorkflowOutput, int] {
	return []evaly.Grader[WorkflowInput, WorkflowOutput, int]{
		evaly.GraderFunc[WorkflowInput, WorkflowOutput, int]{
			Identity: evaly.GraderRevision{ID: "outcome", Implementation: "go-v1", Rubric: rubric},
			Evaluate: func(ctx context.Context, v evaly.View[WorkflowInput, WorkflowOutput, int]) (evaly.Grade, error) {
				if !evaly.CompleteFor(v.Evidence, "outcome") {
					return evaly.Grade{Status: evaly.InsufficientEvidence, Reasons: []string{"outcome_incomplete"}}, nil
				}
				state := WorkflowState{}
				for _, event := range v.Evidence.Events {
					if event.Kind == "outcome" {
						if err := json.Unmarshal(event.Payload, &state); err != nil {
							return evaly.Grade{}, err
						}
					}
				}
				pass := v.Case.Reference != nil && state.Refunded+state.Credited == *v.Case.Reference
				return evaly.Grade{
					Status:     evaly.Scored,
					Assertions: []evaly.Assertion{{Name: "external_balance", Pass: pass}},
					Usage:      evaly.Usage{Known: true, Units: 1},
				}, ctx.Err()
			},
		},
		evaly.GraderFunc[WorkflowInput, WorkflowOutput, int]{
			Identity: evaly.GraderRevision{
				ID:             "trajectory",
				Implementation: "go-v1",
				Rubric:         "allowed-alternatives-v1",
			},
			Evaluate: func(ctx context.Context, v evaly.View[WorkflowInput, WorkflowOutput, int]) (evaly.Grade, error) {
				if !evaly.CompleteFor(v.Evidence, "tool") {
					return evaly.Grade{Status: evaly.InsufficientEvidence, Reasons: []string{"absence_unprovable"}}, nil
				}
				lookup, effect, forbidden := false, false, false
				for _, event := range v.Evidence.Events {
					if event.Kind != "tool" {
						continue
					}
					var p struct {
						Action  string `json:"action"`
						Success bool   `json:"success"`
					}
					if err := json.Unmarshal(event.Payload, &p); err != nil {
						return evaly.Grade{}, err
					}
					lookup = lookup || p.Action == "lookup" && p.Success
					effect = effect || (p.Action == "refund" || p.Action == "credit") && p.Success
					forbidden = forbidden || p.Action == "delete"
				}
				return evaly.Grade{
					Status: evaly.Scored,
					Assertions: []evaly.Assertion{
						{Name: "allowed_path", Pass: lookup && effect},
						{Name: "no_delete", Pass: !forbidden},
					},
					Usage: evaly.Usage{Known: true, Units: 1},
				}, ctx.Err()
			},
		},
	}
}

func WorkflowConfig(
	id, mode string,
) (evaly.RunConfig[WorkflowInput, WorkflowOutput, int, *WorkflowEnvironment], *WorkflowStore, error) {
	var c evaly.RunConfig[WorkflowInput, WorkflowOutput, int, *WorkflowEnvironment]
	switch mode {
	case "refund", "alternative", "text-only", "tool-error", "effect-error":
	default:
		return c, nil, evaly.ErrUnsupported
	}
	reference := 100
	dataset, err := (evaly.DatasetDraft[WorkflowInput, int]{Selection: "all", Cases: []evaly.Case[WorkflowInput, int]{{ID: "account-refund", Input: WorkflowInput{Account: "account-1", Amount: 100}, Reference: &reference, RequiredEvidence: []string{"tool", "outcome"}}}}).Seal(
		WorkflowInputCodec(),
		WorkflowReferenceCodec(),
	)
	if err != nil {
		return c, nil, err
	}
	store := &WorkflowStore{}
	c = evaly.RunConfig[WorkflowInput, WorkflowOutput, int, *WorkflowEnvironment]{
		ID:          id,
		Dataset:     dataset,
		OutputCodec: WorkflowOutputCodec(),
		Plan: evaly.RunPlan{
			Repeats:         1,
			Concurrency:     1,
			Timeout:         time.Second,
			CleanupTimeout:  time.Second,
			MaxAttempts:     1,
			AssertionPolicy: "all",
		},
		Provenance: evaly.Provenance{
			Target: "workflow-" + mode,
			Model:  "scripted",
			Prompt: "host-script-v1",
			Tools:  "account-tools-v1",
			Policy: "refund-or-credit-v1",
		},
		Capture: evaly.CaptureConfig{
			Policy: evaly.FieldPolicy{
				ID: "workflow-safe-v1",
				Allowed: map[string][]string{
					"tool":    {"action", "success"},
					"outcome": {"eligible", "refunded", "credited"},
				},
			},
			KnownKinds:    []string{"tool", "outcome"},
			RequiredKinds: []string{"tool", "outcome"},
			MaxEvents:     10,
			MaxBytes:      4096,
		},
		ProjectionRevision: "workflow-safe-v1",
		Graders:            WorkflowGraders("external-balance-v1"),
	}
	c.Lifecycle = evaly.LifecycleFuncs[*WorkflowEnvironment]{
		IdentityValue: evaly.LifecycleIdentity{
			Fixture:   "workflow-v1",
			Reset:     "empty-account-v1",
			Isolation: evaly.Isolated,
		},
		PrepareFunc: func(ctx context.Context, ns string) (*WorkflowEnvironment, error) {
			store.Prepare(ns)
			return &WorkflowEnvironment{ns, store}, ctx.Err()
		},
		ResetFunc:   func(ctx context.Context, e *WorkflowEnvironment) error { store.Reset(e.Namespace); return ctx.Err() },
		CleanupFunc: func(ctx context.Context, e *WorkflowEnvironment) error { store.Cleanup(e.Namespace); return ctx.Err() },
	}
	c.Target = evaly.TargetFunc[WorkflowInput, WorkflowOutput, *WorkflowEnvironment](
		func(ctx context.Context, i WorkflowInput, t evaly.TrialContext[*WorkflowEnvironment]) (evaly.TargetResult[WorkflowOutput], error) {
			r, events, e := WorkflowInvoke(ctx, i, t.Environment.Namespace, store, mode)
			for _, event := range events {
				if err := t.Evidence.Record(ctx, event); err != nil {
					return r, err
				}
			}
			return r, e
		},
	)
	c.Project = func(ctx context.Context, cs evaly.Case[WorkflowInput, int], o WorkflowOutput, e evaly.EvidenceRecord) (evaly.View[WorkflowInput, WorkflowOutput, int], error) {
		return evaly.View[WorkflowInput, WorkflowOutput, int]{Case: cs, Output: o, Evidence: e}, ctx.Err()
	}
	return c, store, nil
}
