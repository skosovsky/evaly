//go:build e2e

package recipes_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/skosovsky/metry/genai"
	"github.com/skosovsky/metry/metrytest"

	"github.com/skosovsky/evaly"
	"github.com/skosovsky/evaly/integrations/recipes"
)

func TestE2ERoundtripAndRealEvaluationDelivery(t *testing.T) {
	// Arrange.
	ctx := context.Background()
	calls, released := 0, false
	cfg := config(t, "success", &calls, &released)
	provider, mem := metrytest.NewTestProvider(t)
	tracker, err := genai.NewTrackerFromProvider(provider)
	check(t, err)
	registry := recipes.LocalRegistry{Recorder: tracker.EvaluationRecorder()}
	accepted := 0
	sink := recipes.EvaluationSink{Policy: "egress", Deduplicates: true,
		Project: func(_ context.Context, r recipes.EvaluationRecord) (recipes.EvaluationRecord, error) {
			r.Evaluation.Reasoning = ""
			return r, nil
		},
		DeliverOne: func(ctx context.Context, r recipes.EvaluationRecord) error {
			if deliveryErr := registry.Deliver(ctx, r); deliveryErr != nil {
				return deliveryErr
			}
			accepted++
			if accepted == 1 {
				return errors.New("ambiguous ack after accepting")
			}
			return nil
		}}
	// Act.
	experiment, err := evaly.Run(ctx, cfg)
	check(t, err)
	directory := t.TempDir()
	store, err := evaly.OpenFileStore(directory)
	check(t, err)
	check(t, evaly.SaveExperiment(ctx, store, experiment))
	reopened, err := evaly.OpenFileStore(directory)
	check(t, err)
	restored, err := evaly.LoadExperiment(ctx, reopened, cfg.ID)
	check(t, err)
	envelope, err := reopened.Get(ctx, cfg.ID)
	check(t, err)
	before := experiment.Record()
	first := evaly.Export(ctx, sink, evaly.DeliveryRecord{ObservationID: "delivery", Artifact: envelope})
	second := evaly.Export(ctx, sink, evaly.DeliveryRecord{ObservationID: "delivery", Artifact: envelope})
	check(t, provider.ForceFlush(ctx))
	// Assert.
	if !reflect.DeepEqual(before, restored.Record()) {
		t.Fatal("roundtrip changed record")
	}
	if first.State != "failed" || second.State != "delivered" {
		t.Fatalf("delivery: %+v %+v", first, second)
	}
	spans := mem.GetSpans()
	if len(spans) != 2 {
		t.Fatalf("expected two evaluation spans, got %d", len(spans))
	}
	for _, span := range spans {
		if span.Name != "evaluation" || span.EndTime.IsZero() || len(span.Events) != 1 ||
			span.Events[0].Name != "gen_ai.evaluation.result" {
			t.Fatalf("not actual evaluation: %+v", span)
		}
	}
	projected, err := recipes.ProjectEvaluations(restored, "egress")
	check(t, err)
	if projected[1].Metric == nil || projected[1].Metric.Unit != "ratio" {
		t.Fatal("lost metric metadata")
	}
	changed := projected[0]
	changed.Evaluation.Outcome = "fail"
	if !errors.Is(registry.Deliver(ctx, changed), evaly.ErrConflict) {
		t.Fatal("changed content accepted")
	}
	if !reflect.DeepEqual(before, experiment.Record()) {
		t.Fatal("export changed artifact")
	}
	files, err := os.ReadDir(directory)
	check(t, err)
	for _, file := range files {
		data, readErr := os.ReadFile(filepath.Join(directory, file.Name()))
		check(t, readErr)
		if strings.Contains(string(data), secret) {
			t.Fatal("artifact leak")
		}
	}
	data, _ := json.Marshal(spans)
	if strings.Contains(string(data), secret) {
		t.Fatal("sink leak")
	}
}
