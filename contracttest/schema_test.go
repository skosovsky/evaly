package contracttest

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/santhosh-tekuri/jsonschema/v6"

	"github.com/skosovsky/evaly"
	"github.com/skosovsky/evaly/adapters/httpjson"
	"github.com/skosovsky/evaly/internal/fixtures"
	"github.com/skosovsky/evaly/observation"
	"github.com/skosovsky/evaly/optimizer"
)

func compile(t *testing.T, name string) *jsonschema.Schema {
	t.Helper()
	compiler := jsonschema.NewCompiler()
	s, e := compiler.Compile(filepath.Join("..", "schemas", name+"-v1.json"))
	if e != nil {
		t.Fatal(e)
	}
	return s
}
func instance(t *testing.T, v any) any {
	t.Helper()
	b, e := json.Marshal(v)
	if e != nil {
		t.Fatal(e)
	}
	var out any
	if e = json.Unmarshal(b, &out); e != nil {
		t.Fatal(e)
	}
	return out
}

type steps struct{}

func (steps) Revision() string { return "steps-v1" }
func (steps) Step(ctx context.Context, s int) (int, int, bool, error) {
	return s + 1, s + 1, false, ctx.Err()
}
func TestGeneratedArtifactsAgainstWireSchemas(t *testing.T) {
	// Arrange: execute real core paths; validate with an independent JSON Schema engine.
	ctx := context.Background()
	c, e := fixtures.CalculationConfig("contract-baseline", "good", "")
	if e != nil {
		t.Fatal(e)
	}
	experiment, e := evaly.Run(ctx, c)
	if e != nil {
		t.Fatal(e)
	}
	comparison, e := evaly.Compare(experiment, experiment, fixtures.Gate())
	if e != nil {
		t.Fatal(e)
	}
	cases, e := c.Dataset.Cases()
	if e != nil {
		t.Fatal(e)
	}
	evidence := experiment.Record().Trials[0].Evidence
	saved, e := evaly.SaveView(
		evaly.View[fixtures.Calculation, fixtures.CalculationOutput, int]{
			Case:     cases[0],
			Output:   fixtures.CalculationOutput{Sum: 3},
			Evidence: evidence,
		},
		c.ProjectionRevision,
		fixtures.InputCodec(),
		fixtures.OutputCodec(),
		fixtures.ReferenceCodec(),
	)
	if e != nil {
		t.Fatal(e)
	}
	assessment, e := evaly.Rescore(ctx, saved, c.Graders, "observation-1", "", "observation")
	if e != nil {
		t.Fatal(e)
	}
	obs, e := observation.New(
		"observation-1",
		"",
		observation.Sampling{Rule: "all", Reason: "fixture", Population: "staging", Window: "1"},
		saved,
	)
	if e != nil {
		t.Fatal(e)
	}
	worker, e := observation.Start(
		ctx,
		observation.Config[fixtures.Calculation, fixtures.CalculationOutput, int]{
			Capacity:    1,
			Concurrency: 1,
			Deadline:    time.Second,
			Clock:       observation.RealClock{},
			Graders:     c.Graders,
		},
	)
	if e != nil {
		t.Fatal(e)
	}
	future, status := worker.Enqueue(obs)
	if status != "accepted" {
		t.Fatal(status)
	}
	result := <-future.Result
	if e = worker.Cancel(ctx); e != nil {
		t.Fatal(e)
	}
	codec := evaly.JSONCodec[int]{ID: "integer", Version: "1"}
	scenario, e := evaly.RunScenario(
		ctx,
		steps{},
		0,
		evaly.ScenarioPlan{Mode: "search", MaxSteps: 3, Timeout: time.Second},
		codec,
		codec,
	)
	if e != nil {
		t.Fatal(e)
	}
	candidate, e := optimizer.Seal("candidate", "baseline", "enumeration-v1", 1, codec)
	if e != nil {
		t.Fatal(e)
	}
	envelope, e := evaly.NewEnvelope("experiment", experiment.ID(), experiment.Record())
	if e != nil {
		t.Fatal(e)
	}
	values := map[string]any{
		"envelope":           envelope,
		"dataset":            c.Dataset.Record(),
		"experiment":         experiment.Record(),
		"evidence":           evidence,
		"comparison":         comparison,
		"view":               saved.Record(),
		"assessment":         assessment,
		"scenario":           scenario,
		"candidate":          candidate.Record(),
		"observation":        obs.Record(),
		"observation-result": result,
		"search":             searchResult(t),
		"http-request": httpjson.Request{
			Version:     1,
			InputCodec:  fixtures.InputCodec().Identity(),
			OutputCodec: fixtures.OutputCodec().Identity(),
			Trial: httpjson.Trial{
				ID:           "namespace",
				CaseRevision: cases[0].Revision,
				Fixture:      "calculation-v1",
				Reset:        "empty-v1",
			},
			Input: json.RawMessage(`{"left":1,"right":2}`),
		},
		"http-response": httpjson.Response{
			Version: 1,
			Status:  "completed",
			Output:  json.RawMessage(`{"sum":3}`),
			Usage:   evaly.Usage{Known: true, Units: 1},
			Events:  []evaly.Event{},
			Capabilities: evaly.InteropCapabilities{
				Version:       1,
				Outcome:       true,
				ResetIdentity: true,
				Evidence:      true,
				RichStatus:    true,
				MetricScales:  true,
			},
		},
	}
	// Act / Assert.
	for name, value := range values {
		t.Run(name, func(t *testing.T) {
			s := compile(t, name)
			if e = s.Validate(instance(t, value)); e != nil {
				t.Fatal(e)
			}
		})
	}
	// Nested invalid data, not merely malformed JSON or top-level versions.
	schema := compile(t, "experiment")
	for _, variant := range []string{"negative_usage", "unknown_cleanup", "unknown_grade", "invalid_plan", "unknown_version"} {
		t.Run(variant, func(t *testing.T) {
			doc := instance(t, experiment.Record()).(map[string]any)
			manifest := doc["manifest"].(map[string]any)
			trial := doc["trials"].([]any)[0].(map[string]any)
			switch variant {
			case "negative_usage":
				trial["target_usage"].(map[string]any)["units"] = -1
			case "unknown_cleanup":
				trial["cleanup"].(map[string]any)["state"] = "alien"
			case "unknown_grade":
				trial["grades"].([]any)[0].(map[string]any)["status"] = "success-ish"
			case "invalid_plan":
				manifest["plan"].(map[string]any)["repeats"] = 0
			case "unknown_version":
				manifest["version"] = 2
			}
			if schema.Validate(doc) == nil {
				t.Fatal("schema accepted", variant)
			}
		})
	}
	// Every declared schema compiles, including search output schema.
	paths, e := filepath.Glob("../schemas/*-v1.json")
	if e != nil {
		t.Fatal(e)
	}
	for _, path := range paths {
		compiler := jsonschema.NewCompiler()
		if _, e = compiler.Compile(path); e != nil {
			t.Fatal(path, e)
		}
	}
	fixture, e := os.ReadFile("../testdata/contract-fixtures.json")
	if e != nil {
		t.Fatal(e)
	}
	var declared struct {
		Version   int
		Scenarios []struct {
			ID       string
			Expected string
		}
	}
	if e = json.Unmarshal(fixture, &declared); e != nil || declared.Version != 1 || len(declared.Scenarios) != 10 {
		t.Fatal("synthetic fixture catalog", e)
	}
}

func searchResult(t *testing.T) optimizer.Result {
	t.Helper()
	ctx := context.Background()
	codec := evaly.JSONCodec[int]{ID: "integer", Version: "1"}
	candidate, e := optimizer.Seal("candidate", "baseline", "enumeration-v1", 1, codec)
	if e != nil {
		t.Fatal(e)
	}
	datasets := []evaly.Dataset[fixtures.Calculation, int]{}
	for _, id := range []string{"train", "calibration", "holdout"} {
		reference := 3
		d, e := (evaly.DatasetDraft[fixtures.Calculation, int]{Selection: "all", Cases: []evaly.Case[fixtures.Calculation, int]{{ID: id, Input: fixtures.Calculation{Left: 1, Right: 2}, Reference: &reference}}}).Seal(
			fixtures.InputCodec(),
			fixtures.ReferenceCodec(),
		)
		if e != nil {
			t.Fatal(e)
		}
		datasets = append(datasets, d)
	}
	evaluate := func(ctx context.Context, req optimizer.EvaluationRequest[int, fixtures.Calculation, int]) (evaly.Experiment, error) {
		c, e := fixtures.CalculationConfig(req.ExperimentID, "good", "")
		if e != nil {
			return evaly.Experiment{}, e
		}
		c.Dataset = req.Dataset
		c.Budget = req.Budget
		return evaly.Run(ctx, c)
	}
	baseline, e := evaluate(
		ctx,
		optimizer.EvaluationRequest[int, fixtures.Calculation, int]{
			ExperimentID: "calibration-baseline",
			Dataset:      datasets[1],
		},
	)
	if e != nil {
		t.Fatal(e)
	}
	holdout, e := evaluate(
		ctx,
		optimizer.EvaluationRequest[int, fixtures.Calculation, int]{
			ExperimentID: "holdout-baseline",
			Dataset:      datasets[2],
		},
	)
	if e != nil {
		t.Fatal(e)
	}
	budget, e := evaly.NewMemoryBudget(10)
	if e != nil {
		t.Fatal(e)
	}
	result, e := optimizer.Search(
		ctx,
		optimizer.Config[int, fixtures.Calculation, int]{
			ID:                "schema-search",
			Algorithm:         "enumeration-v1",
			StopRevision:      "bounded-v1",
			MaximumCandidates: 1,
			Timeout:           time.Second,
			Split: optimizer.Split[fixtures.Calculation, int]{
				Revision:    "split-v1",
				Training:    datasets[0],
				Calibration: datasets[1],
				Holdout:     datasets[2],
			},
			Candidates:          []optimizer.Candidate[int]{candidate},
			Validate:            func(int) error { return nil },
			Evaluate:            evaluate,
			Budget:              budget,
			Ledger:              &optimizer.MemoryLedger{},
			CalibrationBaseline: baseline,
			HoldoutBaseline:     holdout,
			Gate:                fixtures.Gate(),
		},
	)
	if e != nil {
		t.Fatal(e)
	}
	return result
}
