package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/skosovsky/evaly"
	"github.com/skosovsky/evaly/internal/fixtures"
	"github.com/skosovsky/evaly/optimizer"
)

type Recipe struct {
	Offset int `json:"offset"`
}

func main() {
	ctx := context.Background()
	datasets := []evaly.Dataset[fixtures.Calculation, int]{}
	for _, id := range []string{"training", "calibration", "holdout"} {
		ref := 3
		d, e := (evaly.DatasetDraft[fixtures.Calculation, int]{Selection: "all", Cases: []evaly.Case[fixtures.Calculation, int]{{ID: id, Input: fixtures.Calculation{Left: 1, Right: 2}, Reference: &ref}}}).Seal(
			fixtures.InputCodec(),
			fixtures.ReferenceCodec(),
		)
		if e != nil {
			log.Fatal(e)
		}
		datasets = append(datasets, d)
	}
	evaluate := func(ctx context.Context, req optimizer.EvaluationRequest[Recipe, fixtures.Calculation, int]) (evaly.Experiment, error) {
		candidate, d, budget := req.Candidate, req.Dataset, req.Budget
		c, e := fixtures.CalculationConfig(req.ExperimentID, "good", "")
		if e != nil {
			return evaly.Experiment{}, e
		}
		c.Dataset = d
		c.Budget = budget
		recipe, e := candidate.Value()
		if e != nil {
			return evaly.Experiment{}, e
		}
		original := c.Target
		c.Target = evaly.TargetFunc[fixtures.Calculation, fixtures.CalculationOutput, *fixtures.Environment](
			func(ctx context.Context, i fixtures.Calculation, t evaly.TrialContext[*fixtures.Environment]) (evaly.TargetResult[fixtures.CalculationOutput], error) {
				out, e := original.Run(ctx, i, t)
				out.Output.Sum += recipe.Offset
				return out, e
			},
		)
		return evaly.Run(ctx, c)
	}
	codec := evaly.JSONCodec[Recipe]{ID: "recipe", Version: "1"}
	baseline, e := optimizer.Seal("baseline", "", "enumeration-v1", Recipe{}, codec)
	if e != nil {
		log.Fatal(e)
	}
	other, e := optimizer.Seal("other", baseline.Record().Revision, "enumeration-v1", Recipe{Offset: 1}, codec)
	if e != nil {
		log.Fatal(e)
	}
	calBase, e := evaluate(
		ctx,
		optimizer.EvaluationRequest[Recipe, fixtures.Calculation, int]{
			ExperimentID: "baseline-calibration",
			Candidate:    baseline,
			Dataset:      datasets[1],
		},
	)
	if e != nil {
		log.Fatal(e)
	}
	holdBase, e := evaluate(
		ctx,
		optimizer.EvaluationRequest[Recipe, fixtures.Calculation, int]{
			ExperimentID: "baseline-holdout",
			Candidate:    baseline,
			Dataset:      datasets[2],
		},
	)
	if e != nil {
		log.Fatal(e)
	}
	budget, e := evaly.NewMemoryBudget(10)
	if e != nil {
		log.Fatal(e)
	}
	result, e := optimizer.Search(
		ctx,
		optimizer.Config[Recipe, fixtures.Calculation, int]{
			ID:                "search",
			Algorithm:         "enumeration-v1",
			StopRevision:      "bounded-v1",
			MaximumCandidates: 2,
			Timeout:           time.Second,
			Split: optimizer.Split[fixtures.Calculation, int]{
				Revision:    "split-v1",
				Training:    datasets[0],
				Calibration: datasets[1],
				Holdout:     datasets[2],
			},
			Candidates:          []optimizer.Candidate[Recipe]{baseline, other},
			Validate:            func(Recipe) error { return nil },
			Evaluate:            evaluate,
			Budget:              budget,
			Ledger:              &optimizer.MemoryLedger{},
			CalibrationBaseline: calBase,
			HoldoutBaseline:     holdBase,
			Gate:                fixtures.Gate(),
			Objective:           fixtures.Objective(),
		},
	)
	if e != nil {
		log.Fatal(e)
	}
	fmt.Println("search:", result.State, "winner:", result.Winner)
	if result.HoldoutComparison != nil {
		fmt.Print(evaly.Report(*result.HoldoutComparison))
	}
}
