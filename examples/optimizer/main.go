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
	if err := run("recipe", Recipe{Offset: 1}, Recipe{Offset: 0}, func(r Recipe) int { return r.Offset }); err != nil {
		log.Fatal(err)
	}
	if err := run(
		"resource-plan",
		ResourcePlan{Workers: 1, Reserve: 1},
		ResourcePlan{Workers: 2, Reserve: 0},
		func(r ResourcePlan) int { return 2 - r.Workers },
	); err != nil {
		log.Fatal(err)
	}
}

// Host code owns both candidate domains and the repair algorithm. Each candidate
// still runs through an ordinary experiment; the search sees only measurements.
func run[T any](name string, initial, repaired T, offset func(T) int) error {
	ctx := context.Background()
	datasets, err := searchDatasets()
	if err != nil {
		return err
	}
	codec := evaly.JSONCodec[T]{ID: name, Version: "1"}
	evaluate := func(ctx context.Context, req optimizer.EvaluationRequest[T, fixtures.Calculation, int]) (evaly.Experiment, error) {
		return evaluateCandidate(ctx, req, &offset)
	}

	baseline, err := optimizer.Seal("baseline", "", "host-repair-v1", repaired, codec)
	if err != nil {
		return err
	}
	cal, err := evaluate(
		ctx,
		optimizer.EvaluationRequest[T, fixtures.Calculation, int]{
			ExperimentID: name + "-base-cal",
			Candidate:    baseline,
			Dataset:      datasets[1], SearchID: "", Phase: "", DispatchID: "", Round: 0, Budget: nil,
		},
	)
	if err != nil {
		return err
	}
	hold, err := evaluate(
		ctx,
		optimizer.EvaluationRequest[T, fixtures.Calculation, int]{
			ExperimentID: name + "-base-hold",
			Candidate:    baseline,
			Dataset:      datasets[2], SearchID: "", Phase: "", DispatchID: "", Round: 0, Budget: nil,
		},
	)
	if err != nil {
		return err
	}
	proposer := optimizer.ProposalFunc[T, fixtures.Calculation, int]{
		Identity: "host-feedback-repair-v1",
		Generate: func(proposalCtx context.Context, request optimizer.ProposalRequest[fixtures.Calculation, int]) (optimizer.ProposalResult[T], error) {
			return proposeRepair(proposalCtx, request, &initial, &repaired)
		},
	}
	budget, err := evaly.NewMemoryBudget(searchBudgetUnits)
	if err != nil {
		return err
	}
	result, err := optimizer.Search(ctx, optimizer.Config[T, fixtures.Calculation, int]{
		ID:                name + "-search",
		Algorithm:         "host-repair-v1",
		StopRevision:      "two-rounds-v1",
		Seed:              searchSeed,
		MaximumCandidates: 2,
		MaximumRounds:     2,
		Timeout:           time.Second,
		Split: optimizer.Split[fixtures.Calculation, int]{
			Revision: "distinct-content-v1", Training: datasets[0], Calibration: datasets[1], Holdout: datasets[2],
		},
		SplitValidator: optimizer.KeySplitValidator[fixtures.Calculation, int]{
			Identity: "calculation-content-v1",
			ContentKey: func(c evaly.Case[fixtures.Calculation, int]) (string, error) {
				return fmt.Sprintf("%d+%d", c.Input.Left, c.Input.Right), nil
			}, GroupKey: nil,
		},
		Proposal: proposer,
		Codec:    codec,
		Constraints: optimizer.ConstraintsFunc[T]{
			Identity: "host-accept-v1",
			Assess: func(context.Context, T, optimizer.EvaluationSummary) (optimizer.Feasibility, error) {
				return optimizer.Feasibility{Feasible: true, Reason: ""}, nil
			},
		},
		Evaluate:            evaluate,
		Budget:              budget,
		Ledger:              &optimizer.MemoryLedger{},
		CalibrationBaseline: cal,
		HoldoutBaseline:     hold,
		Gate:                fixtures.Gate(),
		Objective:           fixtures.Objective(),
		EvaluationUnits:     0,
		FeedbackProjector:   nil,
		ProposalUnits:       0,
	})
	if err != nil {
		return err
	}
	if len(result.History) != 2 || result.Winner != result.History[1].Candidate.Revision ||
		result.History[1].Candidate.Parent != result.History[0].Candidate.Revision {
		return fmt.Errorf("feedback improvement contract: %+v", result)
	}
	fmt.Println(name, "rounds:", len(result.RoundHistory), "state:", result.State, "winner:", result.Winner)

	return nil
}

func evaluateCandidate[T any](
	ctx context.Context,
	req optimizer.EvaluationRequest[T, fixtures.Calculation, int],
	offset *func(
		T) int,
) (evaly.Experiment, error) {
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
			result.Output.Sum += (*offset)(value)
			return result, err
		},
	)
	return evaly.Run(ctx, config)
}

func proposeRepair[T any](
	_ context.Context,
	request optimizer.ProposalRequest[fixtures.Calculation, int],
	initial *T,
	repaired *T,
) (optimizer.ProposalResult[T], error) {
	proposal := optimizer.Proposal[T]{ID: "initial", Value: (*initial), Parent: ""}
	if request.Round == 1 {
		if len(request.Feedback) != 1 || request.Feedback[0].Quality == nil ||
			*request.Feedback[0].Quality != 0 {
			return optimizer.ProposalResult[T]{}, evaly.ErrInvalid
		}
		proposal = optimizer.Proposal[T]{
			ID:     "repaired",
			Parent: request.Feedback[0].Candidate.Revision,
			Value:  (*repaired),
		}
	}
	return optimizer.ProposalResult[T]{
		Candidates: []optimizer.Proposal[T]{proposal},
		Usage:      evaly.Usage{Known: true, Units: 0},
		Exhausted:  request.Round == 1,
	}, nil
}

func searchDatasets() ([]evaly.Dataset[fixtures.Calculation, int], error) {
	datasets := []evaly.Dataset[fixtures.Calculation, int]{}
	for n, id := range []string{"training", "calibration", "holdout"} {
		left := 1 + n*2
		reference := left + 2
		d, err := (evaly.DatasetDraft[fixtures.Calculation, int]{Selection: "all", Cases: []evaly.Case[fixtures.Calculation, int]{{ID: id, Input: fixtures.Calculation{Left: left, Right: 2}, Reference: &reference, Revision: "", Metadata: nil, RequiredEvidence: nil, Generation: nil}}, ParentRevision: ""}).Seal(
			fixtures.InputCodec(),
			fixtures.ReferenceCodec(),
		)
		if err != nil {
			return nil, err
		}
		datasets = append(datasets, d)
	}
	return datasets, nil
}
