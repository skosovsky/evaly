package optimizer_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/skosovsky/evaly"
	"github.com/skosovsky/evaly/internal/fixtures"
	"github.com/skosovsky/evaly/optimizer"
)

type recipe struct {
	Offset int `json:"offset"`
}

func splitDataset(t *testing.T, id string) evaly.Dataset[fixtures.Calculation, int] {
	t.Helper()
	left := map[string]int{"train": 1, "calibration": 3, "holdout": 5}[id]
	ref := left + 2
	d, e := (evaly.DatasetDraft[fixtures.Calculation, int]{Selection: "all", Cases: []evaly.Case[fixtures.Calculation, int]{{ID: id, Input: fixtures.Calculation{Left: left, Right: 2}, Reference: &ref}}}).Seal(
		fixtures.InputCodec(),
		fixtures.ReferenceCodec(),
	)
	if e != nil {
		t.Fatal(e)
	}
	return d
}
func searchConfig(t *testing.T, budget float64) optimizer.Config[recipe, fixtures.Calculation, int] {
	t.Helper()
	training, calibration, holdout := splitDataset(
		t,
		"train",
	), splitDataset(
		t,
		"calibration",
	), splitDataset(
		t,
		"holdout",
	)
	base, e := fixtures.CalculationConfig("calibration-baseline", "good", "")
	if e != nil {
		t.Fatal(e)
	}
	base.Dataset = calibration
	calBase, e := evaly.Run(context.Background(), base)
	if e != nil {
		t.Fatal(e)
	}
	base.ID = "holdout-baseline"
	base.Dataset = holdout
	holdBase, e := evaly.Run(context.Background(), base)
	if e != nil {
		t.Fatal(e)
	}
	codec := evaly.JSONCodec[recipe]{ID: "recipe", Version: "1"}
	first, e := optimizer.Seal("first", "", "enumeration-v1", recipe{}, codec)
	if e != nil {
		t.Fatal(e)
	}
	second, e := optimizer.Seal("second", "", "enumeration-v1", recipe{Offset: 1}, codec)
	if e != nil {
		t.Fatal(e)
	}
	b, e := evaly.NewMemoryBudget(budget)
	if e != nil {
		t.Fatal(e)
	}
	return optimizer.Config[recipe, fixtures.Calculation, int]{
		ID:                "search",
		Algorithm:         "enumeration-v1",
		StopRevision:      "bounded-v1",
		MaximumCandidates: 2,
		Timeout:           time.Second,
		Split: optimizer.Split[fixtures.Calculation, int]{
			Revision:    "split-v1",
			Training:    training,
			Calibration: calibration,
			Holdout:     holdout,
		},
		MaximumRounds: 1,
		Proposal: optimizer.NewStaticProposer[recipe, fixtures.Calculation, int](
			"static-v1",
			[]optimizer.Candidate[recipe]{first, second},
		),
		Constraints: optimizer.ConstraintsFunc[recipe]{
			Identity: "nonnegative-v1",
			Assess: func(_ context.Context, value recipe, _ optimizer.EvaluationSummary) (optimizer.Feasibility, error) {
				return optimizer.Feasibility{Feasible: value.Offset >= 0, Reason: "nonnegative"}, nil
			},
		},
		Evaluate: func(ctx context.Context, req optimizer.EvaluationRequest[recipe, fixtures.Calculation, int]) (evaly.Experiment, error) {
			candidate, dataset, budget := req.Candidate, req.Dataset, req.Budget
			r, e := candidate.Value()
			if e != nil {
				return evaly.Experiment{}, e
			}
			c, e := fixtures.CalculationConfig(req.ExperimentID, "good", "")
			if e != nil {
				return evaly.Experiment{}, e
			}
			c.Dataset = dataset
			c.Budget = budget
			original := c.Target
			c.Target = evaly.TargetFunc[fixtures.Calculation, fixtures.CalculationOutput, *fixtures.Environment](
				func(ctx context.Context, i fixtures.Calculation, tc evaly.TrialContext[*fixtures.Environment]) (evaly.TargetResult[fixtures.CalculationOutput], error) {
					o, e := original.Run(ctx, i, tc)
					o.Output.Sum += r.Offset
					return o, e
				},
			)
			return evaly.Run(ctx, c)
		},
		Budget:              b,
		Ledger:              &optimizer.MemoryLedger{},
		CalibrationBaseline: calBase,
		HoldoutBaseline:     holdBase,
		Gate:                fixtures.Gate(),
		Objective:           fixtures.Objective(),
		Codec:               codec,
	}
}
func TestBoundedSearchIncompleteCannotWin(t *testing.T) {
	// Arrange.
	c := searchConfig(t, 1)
	// Act.
	result, e := optimizer.Search(context.Background(), c)
	// Assert.
	if e != nil {
		t.Fatal(e)
	}
	if len(result.History) != 2 || result.History[0].State != "evaluated" || result.History[1].State != "incomplete" ||
		result.Winner == result.History[1].Candidate.Revision ||
		result.History[1].Experiment == nil ||
		len(result.History[1].Experiment.Trials) != 1 {
		t.Fatal(result)
	}
}
func TestProposalIsolationInvalidCandidatesAndContamination(t *testing.T) {
	// Arrange.
	c := searchConfig(t, 20)
	c.Proposal = optimizer.ProposalFunc[recipe, fixtures.Calculation, int]{
		Identity: "scripted-proposal-v1",
		Generate: func(ctx context.Context, r optimizer.ProposalRequest[fixtures.Calculation, int]) (optimizer.ProposalResult[recipe], error) {
			training, e := r.Training.Cases()
			if e != nil {
				t.Fatal(e)
			}
			calibration, e := r.Calibration.Cases()
			if e != nil {
				t.Fatal(e)
			}
			if training[0].ID == "holdout" || calibration[0].ID == "holdout" {
				t.Fatal("holdout leaked")
			}
			return optimizer.ProposalResult[recipe]{
				Candidates: []optimizer.Proposal[recipe]{
					{ID: "valid", Value: recipe{}},
					{ID: "invalid", Value: recipe{Offset: -1}},
				},
				Usage: evaly.Usage{Known: true}, Exhausted: true,
			}, nil
		},
	}
	// Act.
	first, e := optimizer.Search(context.Background(), c)
	if e != nil {
		t.Fatal(e)
	}
	c.ID = "search-two"
	second, e := optimizer.Search(context.Background(), c)
	// Assert.
	if e != nil || first.Contaminated || !second.Contaminated || first.History[1].Feasible ||
		first.Winner == "" ||
		first.HoldoutComparison == nil {
		t.Fatal(e, first, second)
	}
	c.Split.Holdout = c.Split.Calibration
	if _, e = optimizer.Search(context.Background(), c); !errors.Is(e, evaly.ErrConflict) {
		t.Fatal("overlapping split accepted", e)
	}
}

func TestSearchUsesDeclaredDirectionAndAbsentMeasurements(t *testing.T) {
	// Arrange: the host deliberately selects lower-is-better values, independent of assertion quality.
	c := searchConfig(t, 20)
	identity := fixtures.Objective().Identity()
	identity.ID = "host-error-cost"
	identity.Unit = "cost"
	identity.Direction = "lower"
	c.Objective = evaly.ObjectiveFuncs{Descriptor: identity,
		Select: func(evaly.CaseIdentity) (evaly.Eligibility, error) { return evaly.Eligibility{Eligible: true}, nil },
		Evaluate: func(trial evaly.TrialRecord) (evaly.Measurement, error) {
			pass, available := evaly.AssertionOutcome(trial.Grades, "all")
			if !available {
				return evaly.Measurement{Reason: "missing_cost"}, nil
			}
			value := 0.0
			if pass {
				value = 1
			}
			return evaly.Measurement{Present: true, Value: value}, nil
		}}
	c.Gate.MinimumQuality = 1
	c.Gate.MaximumRegression = 1
	// Act.
	result, err := optimizer.Search(context.Background(), c)
	// Assert.
	if err != nil || result.Winner != result.History[1].Candidate.Revision || len(result.Ranking) != 2 ||
		result.History[1].Quality == nil ||
		*result.History[1].Quality != 0 ||
		result.Objective != identity {
		t.Fatal(result, err)
	}

	// Arrange: all measurements are unavailable, never a zero-ranked candidate.
	c.ID = "missing-search"
	missing := c.Objective.(evaly.ObjectiveFuncs)
	missing.Evaluate = func(evaly.TrialRecord) (evaly.Measurement, error) {
		return evaly.Measurement{Reason: "missing_cost"}, nil
	}
	c.Objective = missing
	// Act.
	result, err = optimizer.Search(context.Background(), c)
	// Assert.
	if err != nil || result.Winner != "" || len(result.Ranking) != 0 {
		t.Fatal(result, err)
	}
	for _, entry := range result.History {
		if entry.Quality != nil || entry.State != "incomplete" {
			t.Fatal(entry)
		}
	}
}

func TestInvalidObjectivePreventsPaidDispatch(t *testing.T) {
	// Arrange.
	for _, objective := range []evaly.Objective{nil, evaly.AssertionObjective{ID: "broken", Revision: "1", Policy: "unknown"}} {
		c := searchConfig(t, 20)
		c.Objective = objective
		calls := 0
		c.Proposal = optimizer.ProposalFunc[recipe, fixtures.Calculation, int]{
			Identity: "paid",
			Generate: func(context.Context, optimizer.ProposalRequest[fixtures.Calculation, int]) (optimizer.ProposalResult[recipe], error) {
				calls++
				return optimizer.ProposalResult[recipe]{}, nil
			},
		}
		c.Evaluate = func(context.Context, optimizer.EvaluationRequest[recipe, fixtures.Calculation, int]) (evaly.Experiment, error) {
			calls++
			return evaly.Experiment{}, nil
		}
		// Act.
		_, err := optimizer.Search(context.Background(), c)
		// Assert.
		if !errors.Is(err, evaly.ErrInvalid) || calls != 0 {
			t.Fatal(err, calls)
		}
	}
}
