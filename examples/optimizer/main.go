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
type ResourcePlan struct {
	Workers int `json:"workers"`
	Reserve int `json:"reserve"`
}

func main() {
	run("recipe", Recipe{Offset: 1}, Recipe{}, func(r Recipe) int { return r.Offset })
	run(
		"resource-plan",
		ResourcePlan{Workers: 1, Reserve: 1},
		ResourcePlan{Workers: 2},
		func(r ResourcePlan) int { return 2 - r.Workers },
	)
}

// Host code owns both candidate domains and the repair algorithm. Each candidate
// still runs through an ordinary experiment; the search sees only measurements.
func run[T any](name string, initial, repaired T, offset func(T) int) {
	ctx := context.Background()
	datasets := []evaly.Dataset[fixtures.Calculation, int]{}
	for n, id := range []string{"training", "calibration", "holdout"} {
		left := 1 + n*2
		reference := left + 2
		d, err := (evaly.DatasetDraft[fixtures.Calculation, int]{Selection: "all", Cases: []evaly.Case[fixtures.Calculation, int]{{ID: id, Input: fixtures.Calculation{Left: left, Right: 2}, Reference: &reference}}}).Seal(
			fixtures.InputCodec(),
			fixtures.ReferenceCodec(),
		)
		if err != nil {
			log.Fatal(err)
		}
		datasets = append(datasets, d)
	}
	codec := evaly.JSONCodec[T]{ID: name, Version: "1"}
	evaluate := func(ctx context.Context, req optimizer.EvaluationRequest[T, fixtures.Calculation, int]) (evaly.Experiment, error) {
		value, err := req.Candidate.Value()
		if err != nil {
			return evaly.Experiment{}, err
		}
		config, err := fixtures.CalculationConfig(req.ExperimentID, "good", "")
		if err != nil {
			return evaly.Experiment{}, err
		}
		config.Dataset = req.Dataset
		config.Budget = req.Budget
		original := config.Target
		config.Target = evaly.TargetFunc[fixtures.Calculation, fixtures.CalculationOutput, *fixtures.Environment](
			func(ctx context.Context, input fixtures.Calculation, trial evaly.TrialContext[*fixtures.Environment]) (evaly.TargetResult[fixtures.CalculationOutput], error) {
				result, err := original.Run(ctx, input, trial)
				result.Output.Sum += offset(value)
				return result, err
			},
		)
		return evaly.Run(ctx, config)
	}
	baseline, err := optimizer.Seal("baseline", "", "host-repair-v1", repaired, codec)
	if err != nil {
		log.Fatal(err)
	}
	cal, err := evaluate(
		ctx,
		optimizer.EvaluationRequest[T, fixtures.Calculation, int]{
			ExperimentID: name + "-base-cal",
			Candidate:    baseline,
			Dataset:      datasets[1],
		},
	)
	if err != nil {
		log.Fatal(err)
	}
	hold, err := evaluate(
		ctx,
		optimizer.EvaluationRequest[T, fixtures.Calculation, int]{
			ExperimentID: name + "-base-hold",
			Candidate:    baseline,
			Dataset:      datasets[2],
		},
	)
	if err != nil {
		log.Fatal(err)
	}
	proposer := optimizer.ProposalFunc[T, fixtures.Calculation, int]{
		Identity: "host-feedback-repair-v1",
		Generate: func(_ context.Context, request optimizer.ProposalRequest[fixtures.Calculation, int]) (optimizer.ProposalResult[T], error) {
			proposal := optimizer.Proposal[T]{ID: "initial", Value: initial}
			if request.Round == 1 {
				if len(request.Feedback) != 1 || request.Feedback[0].Quality == nil ||
					*request.Feedback[0].Quality != 0 {
					return optimizer.ProposalResult[T]{}, evaly.ErrInvalid
				}
				proposal = optimizer.Proposal[T]{
					ID:     "repaired",
					Parent: request.Feedback[0].Candidate.Revision,
					Value:  repaired,
				}
			}
			return optimizer.ProposalResult[T]{
				Candidates: []optimizer.Proposal[T]{proposal},
				Usage:      evaly.Usage{Known: true},
				Exhausted:  request.Round == 1,
			}, nil
		},
	}
	budget, err := evaly.NewMemoryBudget(20)
	if err != nil {
		log.Fatal(err)
	}
	result, err := optimizer.Search(ctx, optimizer.Config[T, fixtures.Calculation, int]{
		ID: name + "-search", Algorithm: "host-repair-v1", StopRevision: "two-rounds-v1", Seed: 7,
		MaximumCandidates: 2, MaximumRounds: 2, Timeout: time.Second,
		Split: optimizer.Split[fixtures.Calculation, int]{
			Revision: "distinct-content-v1", Training: datasets[0], Calibration: datasets[1], Holdout: datasets[2],
		},
		SplitValidator: optimizer.KeySplitValidator[fixtures.Calculation, int]{
			Identity: "calculation-content-v1",
			ContentKey: func(c evaly.Case[fixtures.Calculation, int]) (string, error) {
				return fmt.Sprintf("%d+%d", c.Input.Left, c.Input.Right), nil
			},
		},
		Proposal: proposer, Codec: codec,
		Constraints: optimizer.ConstraintsFunc[T]{
			Identity: "host-accept-v1",
			Assess: func(context.Context, T, optimizer.EvaluationSummary) (optimizer.Feasibility, error) {
				return optimizer.Feasibility{Feasible: true}, nil
			},
		},
		Evaluate: evaluate, Budget: budget, Ledger: &optimizer.MemoryLedger{},
		CalibrationBaseline: cal, HoldoutBaseline: hold, Gate: fixtures.Gate(), Objective: fixtures.Objective(),
	})
	if err != nil {
		log.Fatal(err)
	}
	if len(result.History) != 2 || result.Winner != result.History[1].Candidate.Revision ||
		result.History[1].Candidate.Parent != result.History[0].Candidate.Revision {
		log.Fatal("feedback improvement contract", result)
	}
	fmt.Println(name, "rounds:", len(result.RoundHistory), "state:", result.State, "winner:", result.Winner)
}
