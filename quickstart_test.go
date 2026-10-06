package evaly_test

import (
	"context"
	"fmt"
	"time"

	"github.com/skosovsky/evaly"
)

func ExampleRun() {
	ctx := context.Background()
	codec := evaly.JSONCodec[int]{ID: "integer", Version: "1"}
	reference := 3
	dataset, err := (evaly.DatasetDraft[int, int]{
		Cases: []evaly.Case[int, int]{{ID: "addition", Input: 2, Reference: &reference}}, Selection: "all",
	}).Seal(codec, codec)
	if err != nil {
		panic(err)
	}
	config := evaly.RunConfig[int, int, int, struct{}]{
		ID:          "quickstart",
		Dataset:     dataset,
		OutputCodec: codec,
		Lifecycle: evaly.LifecycleFuncs[struct{}]{
			IdentityValue: evaly.LifecycleIdentity{Fixture: "empty-v1", Reset: "empty-v1", Isolation: evaly.Isolated},
			PrepareFunc:   func(context.Context, string) (struct{}, error) { return struct{}{}, nil },
			ResetFunc:     func(context.Context, struct{}) error { return nil },
			CleanupFunc:   func(context.Context, struct{}) error { return nil },
		},
		Target: evaly.TargetFunc[int, int, struct{}](
			func(ctx context.Context, input int, trial evaly.TrialContext[struct{}]) (evaly.TargetResult[int], error) {
				recordErr := trial.Evidence.Record(ctx, evaly.Event{Version: 1, Sequence: 1, Kind: "operation"})
				return evaly.TargetResult[int]{Output: input + 1, Usage: evaly.Usage{Known: true}}, recordErr
			},
		),
		Capture: evaly.CaptureConfig{
			Policy: evaly.FieldPolicy{
				ID: "no-payload-v1",
			},
			KnownKinds: []string{"operation"},
			MaxEvents:  1,
			MaxBytes:   1024,
		},
		ProjectionRevision: "permitted-v1",
		Project: func(_ context.Context, cs evaly.Case[int, int], output int, evidence evaly.EvidenceRecord) (evaly.View[int, int, int], error) {
			return evaly.View[int, int, int]{Case: cs, Output: output, Evidence: evidence}, nil
		},
		Graders: []evaly.Grader[int, int, int]{evaly.GraderFunc[int, int, int]{
			Identity: evaly.GraderRevision{ID: "exact", Implementation: "1", Rubric: "equal-v1"},
			Evaluate: func(_ context.Context, view evaly.View[int, int, int]) (evaly.Grade, error) {
				return evaly.Grade{
					Status: evaly.Scored,
					Usage:  evaly.Usage{Known: true},
					Assertions: []evaly.Assertion{
						{Name: "equal", Pass: view.Case.Reference != nil && view.Output == *view.Case.Reference},
					},
				}, nil
			},
		}},
		Plan: evaly.RunPlan{
			Repeats: 1, Concurrency: 1, MaxAttempts: 1, Timeout: time.Second,
			CleanupTimeout: time.Second, AssertionPolicy: "all",
		},
		Provenance: evaly.Provenance{Target: "add-one-v1", Unknown: []string{"model", "prompt", "tools", "policy"}},
	}
	experiment, err := evaly.Run(ctx, config)
	if err != nil {
		panic(err)
	}
	trial := experiment.Record().Trials[0]
	quality, scored := evaly.AssertionOutcome(trial.Grades, "all")
	healthy := trial.Status == evaly.Completed && trial.Cleanup.State == "completed" && trial.UsageError == ""
	coverage := evaly.CompleteFor(trial.Evidence, "operation")
	// The host explicitly defines its gate; completed/graded alone does not pass.
	fmt.Println("quality pass:", quality && scored)
	fmt.Println("health pass:", healthy && coverage)
	fmt.Println("gate pass:", quality && scored && healthy && coverage)
	// Output:
	// quality pass: true
	// health pass: true
	// gate pass: true
}
