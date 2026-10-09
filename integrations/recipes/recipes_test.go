package recipes_test

import (
	"context"
	"encoding/json"
	"errors"
	"iter"
	"math"
	"testing"
	"time"

	"github.com/skosovsky/metry/genai"
	"github.com/skosovsky/metry/metrytest"
	"github.com/skosovsky/prompty"

	"github.com/skosovsky/evaly"
	"github.com/skosovsky/evaly/integrations/recipes"
)

const secret = "RAW_SECRET_MARKER"

func conversion() recipes.Conversion {
	return recipes.Conversion{
		Identity:   "billing-total",
		SourceUnit: "token",
		Unit:       "credit",
		Mode:       "total",
		TotalRate:  1,
		Bound:      10,
	}
}
func receipt(n int) prompty.Usage {
	return prompty.Usage{
		TotalTokens: n,
		Known:       []prompty.UsageCounter{prompty.UsageTotal},
		Mode:        prompty.UsageCumulative,
	}
}
func check(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

// source is a deterministic transport through the real lazy streaming SDK.
func source(ctx context.Context, mode string, calls *int, released *bool) *prompty.Stream {
	return prompty.NewStream(
		ctx,
		prompty.StreamNative,
		func(_ context.Context) iter.Seq2[*prompty.ResponseChunk, error] {
			return func(yield func(*prompty.ResponseChunk, error) bool) {
				*calls++
				defer func() { *released = true }()
				frames := []*prompty.ResponseChunk{
					{
						Kind:     prompty.StreamDelta,
						Identity: prompty.StreamIdentity{PartIDs: []string{"text"}},
						Content:  []prompty.ContentPart{prompty.TextPart{Text: "confident answer " + secret}},
					},
					{Kind: prompty.StreamUsage, Usage: receipt(3)},
				}
				for _, frame := range frames {
					if !yield(frame, nil) {
						return
					}
				}
				if mode == "error" {
					yield(nil, errors.New(secret))
					return
				}
				if mode == "eof" {
					return
				}
				outcome := prompty.OutcomeCompleted
				if mode == "incomplete" {
					outcome = prompty.OutcomeIncomplete
				}
				if !yield(
					&prompty.ResponseChunk{Kind: prompty.StreamFinish, Outcome: outcome, FinishReason: "stop"},
					nil,
				) {
					return
				}
				// Trailing cumulative usage replaces earlier total. Never bill both snapshots.
				yield(&prompty.ResponseChunk{Kind: prompty.StreamUsage, Usage: receipt(4)}, nil)
			}
		},
	)
}

func target(mode string, calls *int, released *bool) recipes.StreamingTarget[int, int, *int] {
	return recipes.StreamingTarget[int, int, *int]{
		Identity: "stream-target", Conversion: conversion(),
		Open: func(ctx context.Context, _ int, _ evaly.TrialContext[*int]) (*prompty.Stream, error) {
			return source(ctx, mode, calls, released), nil
		},
		Observe: func(ctx context.Context, f *prompty.ResponseChunk, tr evaly.TrialContext[*int]) error {
			if f.Kind != prompty.StreamDelta || mode == "text-only" {
				return nil
			}
			*tr.Environment = 1 // observable committed domain effect independent of answer
			bridge := recipes.CaptureBridge{Sink: tr.Evidence}
			return bridge.Record(
				ctx,
				evaly.Event{
					Version:       1,
					Sequence:      1,
					Kind:          "action",
					CorrelationID: tr.ID,
					Payload:       json.RawMessage(`{"nested":{"secret":"` + secret + `"}}`),
					References:    []string{"opaque:" + secret},
				},
				true,
			)
		},
		Project: func(_ context.Context, _ *prompty.Response, tr evaly.TrialContext[*int]) (int, error) {
			return *tr.Environment, nil
		},
		Complete: func(status prompty.StreamStatus) bool { return status.State == prompty.StreamCompleted },
	}
}

func config(t *testing.T, mode string, calls *int, released *bool) evaly.RunConfig[int, int, int, *int] {
	t.Helper()
	codec := evaly.JSONCodec[int]{ID: "int", Version: "host"}
	ref := 1
	dataset, err := (evaly.DatasetDraft[int, int]{Selection: "all", Cases: []evaly.Case[int, int]{{ID: "case", Input: 1, Reference: &ref}}}).Seal(
		codec,
		codec,
	)
	check(t, err)
	dispatch, err := conversion().ReservationUnits("token", receipt(6))
	check(t, err)
	budget, err := evaly.NewMemoryBudget(10)
	check(t, err)
	return evaly.RunConfig[int, int, int, *int]{
		ID:          "stream-fixture",
		Dataset:     dataset,
		OutputCodec: codec,
		Target:      target(mode, calls, released),
		Budget:      budget,
		Lifecycle: evaly.LifecycleFuncs[*int]{
			IdentityValue: evaly.LifecycleIdentity{Fixture: "isolated", Reset: "clear", Isolation: evaly.Isolated},
			PrepareFunc:   func(context.Context, string) (*int, error) { return new(int), nil },
			ResetFunc:     func(_ context.Context, state *int) error { *state = 0; return nil },
			CleanupFunc:   func(context.Context, *int) error { return nil },
		},
		Plan: evaly.RunPlan{
			Repeats:         1,
			Concurrency:     1,
			MaxAttempts:     1,
			Timeout:         time.Second,
			CleanupTimeout:  time.Second,
			AssertionPolicy: "all",
			DispatchUnits:   dispatch,
		},
		Provenance: evaly.Provenance{
			Target:  "stream-target",
			Unknown: []string{"model", "prompt", "tools", "policy"},
			Provider: map[string]string{
				"adapter":    "stream-target",
				"conversion": conversion().Identity,
				"export":     "egress",
			},
		},
		Capture: evaly.CaptureConfig{
			Policy:        evaly.FieldPolicy{ID: "drop-all"},
			KnownKinds:    []string{"action"},
			RequiredKinds: []string{"action"},
			MaxEvents:     8,
			MaxBytes:      4096,
		},
		CriticalEvidence:   true,
		ProjectionRevision: "domain-safe",
		Project: func(_ context.Context, c evaly.Case[int, int], o int, e evaly.EvidenceRecord) (evaly.View[int, int, int], error) {
			return evaly.View[int, int, int]{Case: c, Output: o, Evidence: e}, nil
		},
		Graders: []evaly.Grader[int, int, int]{
			evaly.GraderFunc[int, int, int]{
				Identity: evaly.GraderRevision{ID: "domain", Implementation: "host", Rubric: "effect"},
				Evaluate: func(_ context.Context, v evaly.View[int, int, int]) (evaly.Grade, error) {
					return evaly.Grade{
						Status: evaly.Scored,
						Usage:  evaly.Usage{Known: true},
						Assertions: []evaly.Assertion{
							{Name: "effect", Pass: v.Output == *v.Case.Reference, Reason: "effect measured"},
						},
						Metrics: []evaly.Metric{
							{
								Name:          "quality",
								Unit:          "ratio",
								ScaleRevision: "scale",
								Value:         1,
								Minimum:       0,
								Maximum:       1,
								Direction:     "higher",
							},
						},
					}, nil
				},
			},
		},
	}
}

func TestConversionPresenceAndLiability(t *testing.T) {
	for _, tc := range []struct {
		name    string
		usage   prompty.Usage
		mode    string
		known   bool
		units   float64
		invalid bool
	}{
		{name: "unknown-estimate", usage: prompty.Usage{TotalTokens: 8}, mode: "total"},
		{name: "zero", usage: receipt(0), mode: "total", known: true},
		{name: "total", usage: receipt(3), mode: "total", known: true, units: 3},
		{name: "total-only-components", usage: receipt(3), mode: "components"},
		{name: "components", usage: prompty.Usage{PromptTokens: 2, CompletionTokens: 3, Known: []prompty.UsageCounter{prompty.UsagePrompt, prompty.UsageCompletion}}, mode: "components", known: true, units: 5},
		{name: "negative", usage: receipt(-1), mode: "total", invalid: true},
		{name: "over-bound", usage: receipt(11), mode: "total", invalid: true},
		{name: "delta", usage: prompty.Usage{TotalTokens: 3, Known: []prompty.UsageCounter{prompty.UsageTotal}, Mode: prompty.UsageDelta}, mode: "total", invalid: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange.
			c := conversion()
			c.Mode = tc.mode
			if tc.mode == "components" {
				c.TotalRate = 0
				c.InputRate = 1
				c.OutputRate = 1
			}
			budget, err := evaly.NewMemoryBudget(10)
			check(t, err)
			reservation, err := budget.Reserve(context.Background(), "call", 10)
			check(t, err)
			check(t, budget.Claim(context.Background(), reservation))
			// Act.
			usage, err := c.Convert("token", tc.usage)
			check(t, budget.Reconcile(context.Background(), reservation, usage))
			// Assert.
			if (err != nil) != tc.invalid || usage.Known != tc.known || usage.Units != tc.units {
				t.Fatalf("receipt %+v error %v", usage, err)
			}
			expected := 10.0
			if tc.known {
				expected = tc.units
			}
			if budget.Used() != expected {
				t.Fatalf("liability %v", budget.Used())
			}
		})
	}
	// Arrange/Act/Assert: incompatible units and invalid rates never yield known usage.
	for _, rate := range []float64{-1, math.NaN(), math.Inf(1)} {
		c := conversion()
		c.TotalRate = rate
		u, err := c.Convert("token", receipt(1))
		if err == nil || u.Known {
			t.Fatal("invalid conversion accepted")
		}
	}
	u, err := conversion().Convert("money", receipt(1))
	if err == nil || u.Known {
		t.Fatal("unit mismatch accepted")
	}
}

func TestConversionExactUpwardRounding(t *testing.T) {
	// Arrange: integers above float precision cannot be silently rounded down.
	c := conversion()
	c.Bound = math.MaxFloat64
	n := int(int64(1)<<53) + 1
	// Act.
	usage, err := c.Convert("token", receipt(n))
	check(t, err)
	// Assert.
	if usage.Units <= float64(n) {
		t.Fatal("receipt undercounted exact source integer")
	}
	c.TotalRate = 0.1
	usage, err = c.Convert("token", receipt(10))
	check(t, err)
	if usage.Units != 2 {
		t.Fatalf("binary policy ceil not honored: %v", usage.Units)
	}
}

func TestRegistryRejectsInvalidSidecarBeforeDedup(t *testing.T) {
	// Arrange.
	ctx := context.Background()
	provider, mem := metrytest.NewTestProvider(t)
	tracker, err := genai.NewTrackerFromProvider(provider)
	check(t, err)
	registry := recipes.LocalRegistry{Recorder: tracker.EvaluationRecorder()}
	calls, released := 0, false
	e, err := evaly.Run(ctx, config(t, "success", &calls, &released))
	check(t, err)
	records, err := recipes.ProjectEvaluations(e, "egress")
	check(t, err)
	metric := records[1]
	metric.Metric.Value = math.NaN()
	// Act.
	first := registry.Deliver(ctx, metric)
	metric.Evaluation.Outcome = "foreign"
	second := registry.Deliver(ctx, metric)
	check(t, provider.ForceFlush(ctx))
	// Assert.
	if !errors.Is(first, evaly.ErrInvalid) || !errors.Is(second, evaly.ErrInvalid) || len(mem.GetSpans()) != 0 {
		t.Fatal("invalid sidecar entered registry")
	}
}
