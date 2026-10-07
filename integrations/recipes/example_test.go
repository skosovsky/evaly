package recipes_test

import (
	"context"
	"fmt"
	"iter"

	"github.com/skosovsky/prompty"

	"github.com/skosovsky/evaly"
	"github.com/skosovsky/evaly/integrations/recipes"
)

func ExampleStreamingTarget() {
	ctx := context.Background()
	conversion := recipes.Conversion{
		Identity:   "token-to-credit",
		SourceUnit: "token",
		Unit:       "credit",
		Mode:       "total",
		TotalRate:  1,
		Bound:      10,
	}
	capture, err := evaly.NewCapture(
		evaly.CaptureConfig{
			Policy:     evaly.FieldPolicy{ID: "drop-payload"},
			KnownKinds: []string{"action"},
			MaxEvents:  8,
			MaxBytes:   4096,
		},
	)
	if err != nil {
		panic(err)
	}
	target := recipes.StreamingTarget[int, string, struct{}]{
		Identity: "completed-output", Conversion: conversion,
		Open: func(ctx context.Context, _ int, _ evaly.TrialContext[struct{}]) (*prompty.Stream, error) {
			stream := prompty.NewStream(
				ctx,
				prompty.StreamNative,
				func(context.Context) iter.Seq2[*prompty.ResponseChunk, error] {
					return func(yield func(*prompty.ResponseChunk, error) bool) {
						frames := []*prompty.ResponseChunk{
							{
								Kind:     prompty.StreamDelta,
								Identity: prompty.StreamIdentity{PartIDs: []string{"text"}},
								Content:  []prompty.ContentPart{prompty.TextPart{Text: "hello"}},
							},
							{Kind: prompty.StreamFinish, Outcome: prompty.OutcomeCompleted, FinishReason: "stop"},
							{
								Kind: prompty.StreamUsage,
								Usage: prompty.Usage{
									TotalTokens: 4,
									Known:       []prompty.UsageCounter{prompty.UsageTotal},
									Mode:        prompty.UsageCumulative,
								},
							},
						}
						for _, f := range frames {
							if !yield(f, nil) {
								return
							}
						}
					}
				},
			)
			return stream, nil
		},
		Observe: func(ctx context.Context, frame *prompty.ResponseChunk, trial evaly.TrialContext[struct{}]) error {
			if frame.Kind != prompty.StreamDelta {
				return nil
			}
			bridge := recipes.CaptureBridge{Sink: trial.Evidence}
			return bridge.Record(
				ctx,
				evaly.Event{Version: 1, Sequence: 1, Kind: "action", CorrelationID: trial.ID},
				true,
			)
		},
		Project: func(_ context.Context, r *prompty.Response, _ evaly.TrialContext[struct{}]) (string, error) {
			return r.StrictText()
		},
		// Full capture is an explicit promise of this local source, not inferred from OTel.
		Complete: func(s prompty.StreamStatus) bool { return s.State == prompty.StreamCompleted },
	}
	out, err := target.Run(ctx, 0, evaly.TrialContext[struct{}]{ID: "trial", Evidence: capture})
	if err != nil {
		panic(err)
	}
	fmt.Println(out.Output, out.Usage.Known, out.Usage.Units)
	fmt.Println("complete capture:", evaly.CompleteFor(capture.Seal(), "action"))
	// Output:
	// hello true 4
	// complete capture: true
}
