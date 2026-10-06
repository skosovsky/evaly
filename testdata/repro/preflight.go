// Baseline behavioral repro; these assertions intentionally describe bad behavior.
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/skosovsky/evaly"
)

type preflightDriver struct{}

func (preflightDriver) Revision() string { return "v1" }
func (preflightDriver) Step(_ context.Context, s int, _ evaly.ScenarioContext) (int, int, bool, error) {
	return s + 1, s + 1, true, nil
}

func main() {
	ctx := context.Background()
	codec := evaly.JSONCodec[int]{ID: "int", Version: "1"}
	dataset, err := (evaly.DatasetDraft[int, int]{Selection: "all", Cases: []evaly.Case[int, int]{{ID: "case", Input: 1}}}).Seal(codec, codec)
	preflightMust(err)
	scenario, err := evaly.RunScenario(ctx, preflightDriver{}, 0, evaly.ScenarioPlan{Mode: "replay", MaxSteps: 1, Timeout: time.Second}, codec, codec)
	preflightMust(err)
	var missing *evaly.JSONCodec[int]
	if !preflightPanics(func() { _, err = evaly.RestoreDataset(dataset.Record(), missing, codec) }) {
		panic("F06 dataset typed-nil panic not reproduced")
	}
	if !preflightPanics(func() { _, err = evaly.RestoreScenario(scenario, codec, missing) }) {
		panic("F06 scenario typed-nil panic not reproduced")
	}
	grader := evaly.AbsenceGrader[int, int, int](evaly.GraderRevision{ID: "absence", Implementation: "v1", Rubric: "v1"}, "tool", nil)
	preflightMust(evaly.ValidatePort(grader))
	empty := preflightEvidence(ctx, false)
	grades := evaly.Assess(ctx, []evaly.Grader[int, int, int]{grader}, func() (evaly.View[int, int, int], error) { return evaly.View[int, int, int]{Evidence: empty}, nil })
	if len(grades) != 1 || grades[0].Status != evaly.Scored || !grades[0].Assertions[0].Pass {
		panic("F07 nil predicate false pass not reproduced")
	}
	event := preflightEvidence(ctx, true)
	if !preflightPanics(func() {
		evaly.Assess(ctx, []evaly.Grader[int, int, int]{grader}, func() (evaly.View[int, int, int], error) { return evaly.View[int, int, int]{Evidence: event}, nil })
	}) {
		panic("F07 nil predicate panic not reproduced")
	}
	fmt.Println("F06/F07 reproduced: typed-nil restore panics; nil predicate passes or panics")
}
func preflightEvidence(ctx context.Context, event bool) evaly.EvidenceRecord {
	capture, err := evaly.NewCapture(evaly.CaptureConfig{Policy: evaly.FieldPolicy{ID: "none"}, KnownKinds: []string{"tool"}, MaxEvents: 1, MaxBytes: 1024})
	preflightMust(err)
	if event {
		preflightMust(capture.Record(ctx, evaly.Event{Version: 1, Sequence: 1, Kind: "tool"}))
	}
	return capture.Seal()
}
func preflightPanics(fn func()) (did bool) {
	defer func() {
		if recover() != nil {
			did = true
		}
	}()
	fn()
	return false
}
func preflightMust(err error) {
	if err != nil {
		panic(err)
	}
}
