package optimizer_test

import (
	"context"
	"errors"
	"testing"

	"github.com/skosovsky/evaly"
	"github.com/skosovsky/evaly/internal/fixtures"
	"github.com/skosovsky/evaly/optimizer"
)

func TestPaidProposalCancellationRetainsUsageAndReceivedLineage(t *testing.T) {
	// Arrange: cancellation occurs inside the paid callback, after it produces a candidate.
	c := searchConfig(t, 30)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c.ProposalUnits = 3
	c.Proposal = optimizer.ProposalFunc[recipe, fixtures.Calculation, int]{
		Identity: "paid-cancel-v1",
		Generate: func(context.Context, optimizer.ProposalRequest[fixtures.Calculation, int]) (optimizer.ProposalResult[recipe], error) {
			cancel()
			return optimizer.ProposalResult[recipe]{
				Candidates: []optimizer.Proposal[recipe]{{ID: "received", Value: recipe{}}},
				Usage:      evaly.Usage{Known: true, Units: 2},
			}, nil
		},
	}
	evaluations := 0
	c.Evaluate = func(context.Context, optimizer.EvaluationRequest[recipe, fixtures.Calculation, int]) (evaly.Experiment, error) {
		evaluations++
		return evaly.Experiment{}, errors.New("must not dispatch")
	}
	// Act.
	r, err := optimizer.Search(ctx, c)
	// Assert: settlement survives cancellation; retained metadata never authorizes evaluation.
	if err != nil || r.State != "stopped" || r.Reason != "deadline" || evaluations != 0 || r.Holdout != nil ||
		len(r.History) != 0 ||
		r.ProposalUsage != (evaly.Usage{Known: true, Units: 2}) ||
		len(r.RoundHistory) != 1 {
		t.Fatal(err, evaluations, r)
	}
	round := r.RoundHistory[0]
	if round.Usage != r.ProposalUsage || len(round.Received) != 1 || round.Received[0].ID != "received" ||
		round.Received[0].Algorithm != c.Algorithm ||
		round.Received[0].Codec != c.Codec.Identity() ||
		c.Budget.(*evaly.MemoryBudget).Used() != 2 ||
		optimizer.ValidateResult(r) != nil {
		t.Fatal(round, r)
	}
}

func TestRepeatedSearchIDCannotRedispatchZeroUnitProposal(t *testing.T) {
	// Arrange: even a free reservation carries a single dispatch identity.
	c := searchConfig(t, 30)
	c.ProposalUnits = 0
	proposals := 0
	c.Proposal = optimizer.ProposalFunc[recipe, fixtures.Calculation, int]{
		Identity: "single-dispatch-v1",
		Generate: func(context.Context, optimizer.ProposalRequest[fixtures.Calculation, int]) (optimizer.ProposalResult[recipe], error) {
			proposals++
			return optimizer.ProposalResult[recipe]{Exhausted: true, Usage: evaly.Usage{Known: true}}, nil
		},
	}
	// Act.
	first, firstErr := optimizer.Search(context.Background(), c)
	second, secondErr := optimizer.Search(context.Background(), c)
	// Assert.
	if firstErr != nil || secondErr != nil || proposals != 1 || first.State != "completed" ||
		second.State != "stopped" ||
		second.Reason != "proposal_budget" ||
		second.Holdout != nil ||
		optimizer.ValidateResult(second) != nil {
		t.Fatal(firstErr, secondErr, proposals, first, second)
	}
}

func TestLowerObjectiveSeparatesMeasuredBestFromFeasibleWinner(t *testing.T) {
	for _, feasible := range []bool{true, false} {
		t.Run(map[bool]string{true: "feasible_runner_up", false: "all_infeasible"}[feasible], func(t *testing.T) {
			checkLowerObjectiveSeparatesMeasuredBestFromFeasibleWinner(
				// Arrange: a cheaper measured candidate violates the host constraint.
				t, &feasible)
		},

		// Act.

		// Assert: ranking includes rejected measurements without promoting them.

		)
	}
}

func TestFeedbackRoundRepairsFailureWithoutHoldoutLeakageOrReselection(t *testing.T) {
	// Arrange: the first candidate fails evaluation; the proposer uses its sealed lineage to repair it.
	c := searchConfig(t, 30)
	c.MaximumRounds = 2
	proposals, calibrations, holdouts := 0, 0, 0
	c.Proposal = optimizer.ProposalFunc[recipe, fixtures.Calculation, int]{
		Identity: "repair-v1",
		Generate: func(_ context.Context, request optimizer.ProposalRequest[fixtures.Calculation, int]) (optimizer.ProposalResult[recipe], error) {
			return repairFailureProposal(t, request, c, &proposals)
		},
	}
	evaluate := c.Evaluate
	c.Evaluate = func(ctx context.Context, request optimizer.EvaluationRequest[recipe, fixtures.Calculation, int]) (evaly.Experiment, error) {
		if request.Phase == "calibration" {
			calibrations++
			if request.Candidate.Record().ID == "broken" {
				return evaly.Experiment{}, errors.New("host evaluation failed")
			}
		} else {
			holdouts++
			// Holdout rejects the repaired winner; this result must never feed another proposal.
			bad, err := optimizer.Seal(
				"repaired",
				request.Candidate.Record().Parent,
				c.Algorithm,
				recipe{Offset: 1},
				c.Codec,
			)
			if err != nil {
				t.Fatal(err)
			}
			request.Candidate = bad
		}
		return evaluate(ctx, request)
	}
	// Act.
	r, err := optimizer.Search(context.Background(), c)
	// Assert: an ordinary sealed experiment measures the repair, with one final holdout only.
	if err != nil || proposals != 2 || calibrations != 2 || holdouts != 1 || len(r.History) != 2 ||
		len(r.RoundHistory) != 2 {
		t.Fatal(err, proposals, calibrations, holdouts, r)
	}
	if r.State != "completed" ||
		r.History[1].Candidate.Parent != r.History[0].Candidate.Revision ||
		r.History[1].Experiment == nil ||
		r.History[1].Experiment.Manifest.State != "sealed" ||
		r.History[1].Quality == nil ||
		*r.History[1].Quality != 1 ||
		r.Winner != r.History[1].Candidate.Revision ||
		r.HoldoutComparison == nil ||
		r.HoldoutComparison.Verdict != evaly.GateFail ||
		optimizer.ValidateResult(r) != nil {
		t.Fatal(err, proposals, calibrations, holdouts, r)
	}
}

func TestTwoRoundProposalImprovesMeasuredCandidateFromFeedback(t *testing.T) {
	// Arrange: a deterministic proposer changes the host recipe only after observing failed quality.
	c := searchConfig(t, 30)
	c.MaximumRounds = 2
	proposalCalls := 0
	c.Proposal = optimizer.ProposalFunc[recipe, fixtures.Calculation, int]{
		Identity: "quality-repair-v1",
		Generate: func(_ context.Context, request optimizer.ProposalRequest[fixtures.Calculation, int]) (optimizer.ProposalResult[recipe], error) {
			proposalCalls++
			if request.Round == 0 {
				return optimizer.ProposalResult[recipe]{
					Candidates: []optimizer.Proposal[recipe]{{ID: "poor", Value: recipe{Offset: 1}}},
					Usage:      evaly.Usage{Known: true},
				}, nil
			}
			if request.Round != 1 || len(request.Feedback) != 1 {
				t.Fatal(request)
			}
			previous := request.Feedback[0]
			if previous.Candidate.ID != "poor" || previous.State != "evaluated" || previous.Quality == nil ||
				*previous.Quality != 0 ||
				!previous.Measurement.MeanAvailable ||
				previous.Measurement.Mean != 0 ||
				previous.Candidate.Revision == "" {
				t.Fatal(previous)
			}
			return optimizer.ProposalResult[recipe]{
				Candidates: []optimizer.Proposal[recipe]{
					{ID: "improved", Parent: previous.Candidate.Revision, Value: recipe{}},
				},
				Usage:     evaly.Usage{Known: true},
				Exhausted: true,
			}, nil
		},
	}
	// Act.
	r, err := optimizer.Search(context.Background(), c)
	// Assert: feedback-driven improvement is evaluated through the ordinary experiment callback.
	if err != nil || proposalCalls != 2 || len(r.History) != 2 {
		t.Fatal(err, proposalCalls, r)
	}
	if r.History[0].Quality == nil || *r.History[0].Quality != 0 ||
		r.History[1].Quality == nil ||
		*r.History[1].Quality != 1 ||
		r.History[1].Candidate.Parent != r.History[0].Candidate.Revision ||
		r.History[1].Experiment == nil ||
		r.History[1].Experiment.Manifest.State != "sealed" ||
		r.Winner != r.History[1].Candidate.Revision ||
		r.HoldoutComparison == nil ||
		r.HoldoutComparison.Verdict != evaly.GatePass ||
		optimizer.ValidateResult(r) != nil {
		t.Fatal(err, proposalCalls, r)
	}
}
func checkLowerObjectiveSeparatesMeasuredBestFromFeasibleWinner(t *testing.T, feasible *bool) {
	t.Helper()
	// Arrange.
	c := searchConfig(t, 30)
	id := fixtures.Objective().Identity()
	id.ID, id.Unit, id.Direction = "host-cost", "cost", "lower"
	c.Objective = evaly.ObjectiveFuncs{
		Descriptor: id,
		Select:     func(evaly.CaseIdentity) (evaly.Eligibility, error) { return evaly.Eligibility{Eligible: true}, nil },
		Evaluate: func(trial evaly.TrialRecord) (evaly.Measurement, error) {
			pass, present := evaly.AssertionOutcome(trial.Grades, "all")
			value := 0.0
			if pass {
				value = 1
			}
			return evaly.Measurement{Present: present, Value: value}, nil
		},
	}
	c.Gate.MinimumQuality, c.Gate.MaximumRegression = 1, 1
	c.Constraints = optimizer.ConstraintsFunc[recipe]{
		Identity: "host-feasibility-v1",
		Assess: func(_ context.Context, value recipe, _ optimizer.EvaluationSummary) (optimizer.Feasibility, error) {
			return optimizer.Feasibility{
				Feasible: (*feasible) && value.Offset == 0,
				Reason:   "host_constraint",
			}, nil
		},
	}

	// Act.
	r, err := optimizer.Search(context.Background(), c)

	// Assert.
	if err != nil || len(r.History) != 2 || len(r.Ranking) != 2 ||
		r.BestMeasured != r.History[1].Candidate.Revision ||
		r.History[1].Feasible ||
		*r.History[1].Quality != 0 ||
		optimizer.ValidateResult(r) != nil {
		t.Fatal(err, r)
	}
	if *feasible {
		if r.Winner != r.History[0].Candidate.Revision || r.Winner == r.BestMeasured || r.Holdout == nil {
			t.Fatal(r)
		}
	} else if r.Winner != "" || r.Holdout != nil || r.Reason != "no_selectable_candidate" {
		t.Fatal(r)
	}
}

func repairFailureProposal(
	t *testing.T,
	request optimizer.ProposalRequest[fixtures.Calculation, int],
	c optimizer.Config[recipe, fixtures.Calculation, int],
	proposals *int,
) (optimizer.ProposalResult[recipe], error) {
	(*proposals)++
	checkProposalVisibility(t, request)
	if request.Round == 0 {
		if len(request.Feedback) != 0 {
			t.Fatal(request.Feedback)
		}
		return optimizer.ProposalResult[recipe]{
			Candidates: []optimizer.Proposal[recipe]{{ID: "broken", Value: recipe{Offset: 1}}},
			Usage:      evaly.Usage{Known: true},
		}, nil
	}
	if request.Round != 1 || len(request.Feedback) != 1 {
		t.Fatal(request)
	}
	f := request.Feedback[0]
	if f.Candidate.ID != "broken" || f.Candidate.Revision == "" || f.Round != 0 || f.State != "failed" ||
		f.Reason != "evaluation_failure" ||
		f.Quality != nil ||
		f.Candidate.Algorithm != c.Algorithm ||
		f.Candidate.Codec != c.Codec.Identity() {
		t.Fatal(f)
	}
	return optimizer.ProposalResult[recipe]{
		Candidates: []optimizer.Proposal[recipe]{
			{ID: "repaired", Parent: f.Candidate.Revision, Value: recipe{}},
		},
		Usage:     evaly.Usage{Known: true},
		Exhausted: true,
	}, nil
}

func checkProposalVisibility(t *testing.T, request optimizer.ProposalRequest[fixtures.Calculation, int]) {
	t.Helper()
	for name, dataset := range map[string]evaly.Dataset[fixtures.Calculation, int]{"train": request.Training, "calibration": request.Calibration} {
		cases, err := dataset.Cases()
		if err != nil || len(cases) != 1 || cases[0].ID != name || cases[0].Reference == nil ||
			*cases[0].Reference != cases[0].Input.Left+2 {
			t.Fatal("wrong visible dataset or label", name, err, cases)
		}
	}
}
