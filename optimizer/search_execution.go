package optimizer

import (
	"context"
	"errors"
	"math"
	"sort"
	"strconv"

	"github.com/skosovsky/evaly"
)

type searchExecution[T, I, R any] struct {
	config        Config[T, I, R]
	ctx           context.Context
	result        Result
	codecIdentity evaly.CodecIdentity
	attempts      int
	seen          map[string]bool
	candidates    map[string]Candidate[T]
	feedback      []Feedback
	exhausted     bool
}

func initialResult[T, I, R any](c Config[T, I, R]) Result {
	var zeroObjectiveIdentity evaly.ObjectiveIdentity
	var zeroUsage evaly.Usage
	r := Result{
		Version: resultWireRevision,
		Provenance: Provenance{
			Training:            c.Split.Training.Revision(),
			Calibration:         c.Split.Calibration.Revision(),
			Holdout:             c.Split.Holdout.Revision(),
			CalibrationBaseline: c.CalibrationBaseline.Revision(),
			HoldoutBaseline:     c.HoldoutBaseline.Revision(),
		},
		ID:                      c.ID,
		Algorithm:               c.Algorithm,
		Split:                   c.Split.Revision,
		StopRevision:            c.StopRevision,
		Seed:                    c.Seed,
		State:                   proposingState,
		States:                  []string{proposingState},
		History:                 []Evaluation{},
		Ranking:                 []string{},
		RoundHistory:            []Round{},
		TieRevision:             TieRevision,
		MaximumRounds:           c.MaximumRounds,
		MaximumCandidates:       c.MaximumCandidates,
		TimeoutNanoseconds:      int64(c.Timeout),
		ProposalUnits:           c.ProposalUnits,
		EvaluationUnits:         c.EvaluationUnits,
		BestMeasured:            "",
		ConstraintsRevision:     "",
		SplitValidationRevision: "",
		FeedbackRevision:        "",
		IndependenceValidated:   false,
		Objective:               zeroObjectiveIdentity,
		Revision:                "",
		Reason:                  "",
		Winner:                  "",
		Holdout:                 nil,
		HoldoutComparison:       nil,
		Contaminated:            false,
		ProposalRevision:        "",
		ProposalUsage:           zeroUsage,
	}
	return r
}
func (s *searchExecution[T, I, R]) preflight() error {
	if s.config.ID == "" || s.config.Algorithm == "" || s.config.StopRevision == "" ||
		s.config.MaximumCandidates <= 0 ||
		s.config.MaximumCandidates > 10000 ||
		s.config.MaximumRounds <= 0 ||
		s.config.MaximumRounds > 10000 ||
		s.config.Timeout <= 0 ||
		s.config.Evaluate == nil ||
		math.IsNaN(s.config.ProposalUnits) ||
		math.IsInf(s.config.ProposalUnits, 0) ||
		s.config.ProposalUnits < 0 ||
		math.IsNaN(s.config.EvaluationUnits) ||
		math.IsInf(s.config.EvaluationUnits, 0) ||
		s.config.EvaluationUnits < 0 {
		return evaly.ErrInvalid
	}
	for _, p := range []any{s.config.Budget, s.config.Ledger, s.config.Proposal, s.config.Codec, s.config.Objective, s.config.Constraints} {
		if e := evaly.ValidatePort(p); e != nil {
			return e
		}
	}
	if err := s.validateProposalMaximum(); err != nil {
		return err
	}
	if e := validateSplit(s.config.Split); e != nil {
		return e
	}
	s.result.Objective = s.config.Objective.Identity()
	s.result.ProposalRevision = s.config.Proposal.Revision()
	s.result.ConstraintsRevision = s.config.Constraints.Revision()
	if evaly.ValidateObjectiveIdentity(s.result.Objective) != nil ||
		evaly.ValidateCodecIdentity(s.config.Codec.Identity()) != nil ||
		s.result.ProposalRevision == "" ||
		s.result.ConstraintsRevision == "" {
		return evaly.ErrInvalid
	}
	if _, e := evaly.NewEnvelope("search", s.config.ID, struct{}{}); e != nil {
		return e
	}
	if s.config.CalibrationBaseline.Record().Manifest.Dataset != s.config.Split.Calibration.Revision() ||
		s.config.HoldoutBaseline.Record().Manifest.Dataset != s.config.Split.Holdout.Revision() {
		return evaly.ErrConflict
	}
	for _, b := range []evaly.Experiment{s.config.CalibrationBaseline, s.config.HoldoutBaseline} {
		if _, e := evaly.RestoreExperiment(b.Record()); e != nil {
			return e
		}
		if b.Record().Manifest.State != sealedState {
			return evaly.ErrUnsealed
		}
	}
	return nil
}
func (s *searchExecution[T, I, R]) preflightMeasurement() error {
	cmp, e := evaly.Compare(
		s.config.CalibrationBaseline,
		s.config.CalibrationBaseline,
		s.config.Objective,
		s.config.Gate,
	)
	if e != nil {
		return e
	}
	if cmp.Verdict == evaly.GateInvalid {
		return evaly.ErrInvalid
	}
	if s.config.SplitValidator != nil {
		if e := evaly.ValidatePort(s.config.SplitValidator); e != nil {
			return e
		}
		s.result.SplitValidationRevision = s.config.SplitValidator.Revision()
		if s.result.SplitValidationRevision == "" {
			return evaly.ErrInvalid
		}
		if e := s.config.SplitValidator.ValidateSplit(s.ctx, s.config.Split); e != nil {
			return e
		}
		if s.config.SplitValidator.Revision() != s.result.SplitValidationRevision {
			return evaly.ErrConflict
		}
		s.result.IndependenceValidated = true
	}
	if s.config.FeedbackProjector != nil {
		if e := evaly.ValidatePort(s.config.FeedbackProjector); e != nil {
			return e
		}
		s.result.FeedbackRevision = s.config.FeedbackProjector.Revision()
		if s.result.FeedbackRevision == "" {
			return evaly.ErrInvalid
		}
	}
	return nil
}
func (s *searchExecution[T, I, R]) authorize(id string, units float64) (evaly.Reservation, error) {
	if e := s.ctx.Err(); e != nil {
		return evaly.Reservation{}, e
	}
	res, e := s.config.Budget.Reserve(s.ctx, id, units)
	if e != nil {
		return res, e
	}
	if e = s.config.Budget.Claim(s.ctx, res); e != nil {
		return res, e
	}
	if e := s.ctx.Err(); e != nil {
		return res, e
	}
	return res, nil
}
func (s *searchExecution[T, I, R]) reconcile(res evaly.Reservation, u evaly.Usage) error {
	rc, done := context.WithTimeout(context.WithoutCancel(s.ctx), s.config.Timeout)
	defer done()
	return s.config.Budget.Reconcile(rc, res, u)
}
func (s *searchExecution[T, I, R]) runRounds() {
	for round := 0; round < s.config.MaximumRounds && s.result.State != stoppedState; round++ {
		if s.ctx.Err() != nil {
			s.stop(deadlineReason)
			break
		}
		if s.attempts >= s.config.MaximumCandidates {
			s.stop("candidate_limit")
			break
		}
		rr, proposal, proceed := s.proposeRound(round)
		if !proceed {
			break
		}
		s.evaluateRound(round, proposal, &rr)

		rr.State = completedState
		if s.result.State == stoppedState {
			rr.State = stoppedState
			rr.Reason = s.result.Reason
		}
		s.result.RoundHistory = append(s.result.RoundHistory, rr)
		if proposal.Exhausted && s.result.State != stoppedState {
			s.exhausted = true
			break
		}
	}
	if !s.exhausted && s.result.State != stoppedState {
		if s.attempts >= s.config.MaximumCandidates {
			s.stop("candidate_limit")
		} else {
			s.stop("round_limit")
		}
	}
}
func (s *searchExecution[T, I, R]) proposeRound(round int) (Round, ProposalResult[T], bool) {
	var proposal ProposalResult[T]
	dispatch := s.config.ID + "/proposal/" + strconv.Itoa(round)
	rr := Round{
		Index:         round,
		ReceivedCount: 0, ReceivedTruncated: false,
		DispatchID:       dispatch,
		ProposalRevision: s.result.ProposalRevision,
		State:            proposingState,
		Candidates:       []string{},
		Received:         []CandidateLineage{}, Reason: "", Usage: evaly.Usage{Known: false, Units: 0},
	}
	res, err := s.authorize(dispatch, s.config.ProposalUnits)
	if err != nil {
		rr.State = stoppedState
		rr.Reason = "proposal_budget"
		s.result.RoundHistory = append(s.result.RoundHistory, rr)
		s.stop(rr.Reason)
		return rr, proposal, false
	}

	fb, cloneErr := cloneServiceValue(s.feedback)
	if cloneErr != nil {
		rr.State = stoppedState
		rr.Reason = "feedback_encoding_failure"
		s.result.RoundHistory = append(s.result.RoundHistory, rr)
		s.stop(rr.Reason)
		return rr, proposal, false
	}
	proposal, err = s.config.Proposal.Propose(
		s.ctx,
		ProposalRequest[I, R]{
			Training:    s.config.Split.Training,
			Calibration: s.config.Split.Calibration,
			Maximum:     s.config.MaximumCandidates - s.attempts,
			Seed:        s.config.Seed,
			Round:       round,
			DispatchID:  dispatch,
			Feedback:    fb,
		},
	)
	if math.IsNaN(proposal.Usage.Units) || math.IsInf(proposal.Usage.Units, 0) || proposal.Usage.Units < 0 {
		proposal.Usage = evaly.Usage{Known: false, Units: 0}
		err = evaly.ErrInvalid
	}
	s.recordReceived(&rr, proposal)
	rr.Usage = proposal.Usage
	if s.config.Proposal.Revision() != s.result.ProposalRevision || s.config.Codec.Identity() != s.codecIdentity {
		err = evaly.ErrConflict
	}
	usageErr := s.reconcile(res, proposal.Usage)
	if err != nil || usageErr != nil || s.ctx.Err() != nil {
		rr.State = stoppedState
		rr.Reason = "proposal_failure"
		if s.ctx.Err() != nil {
			rr.Reason = deadlineReason
		}
		if usageErr != nil {
			rr.Reason = "proposal_usage_failure"
		}
		s.result.RoundHistory = append(s.result.RoundHistory, rr)
		s.stop(rr.Reason)
		return rr, proposal, false
	}
	if len(proposal.Candidates) > s.config.MaximumCandidates-s.attempts {
		rr.State = stoppedState
		rr.Reason = "proposal_limit"
		s.result.RoundHistory = append(s.result.RoundHistory, rr)
		s.stop(rr.Reason)
		return rr, proposal, false
	}
	if len(proposal.Candidates) == 0 {
		rr.State = completedState
		s.result.RoundHistory = append(s.result.RoundHistory, rr)
		s.exhausted = true
		return rr, proposal, false
	}

	return rr, proposal, true
}
func (s *searchExecution[T, I, R]) evaluateCandidate(round, index int, p Proposal[T], rr *Round) bool {
	var zeroAggregate evaly.Aggregate

	s.attempts++
	if s.ctx.Err() != nil {
		s.stop(deadlineReason)
		return false
	}
	entry := Evaluation{
		Round:             round,
		DispatchID:        s.config.ID + "/evaluation/" + strconv.Itoa(round) + "/" + strconv.Itoa(index),
		References:        []string{},
		Feasible:          false,
		FeasibilityReason: "",
		Candidate: CandidateRecord{
			ID:             "",
			Revision:       "",
			ParentRevision: "",
			Algorithm:      "",
			Description:    nil,
			Codec:          evaly.CodecIdentity{ID: "", Version: ""},
		},
		State:      "",
		Reason:     "",
		Experiment: nil,
		Comparison: nil,
		Quality:    nil,
	}
	candidate, err := Seal(p.ID, p.ParentRevision, s.config.Algorithm, p.Value, s.config.Codec)
	entry.Candidate = candidate.Record()
	rr.Received[index].Revision = entry.Candidate.Revision
	switch {
	case err != nil:
		entry.Candidate = CandidateRecord{
			ID:             p.ID,
			ParentRevision: p.ParentRevision,
			Algorithm:      s.config.Algorithm,
			Codec:          s.config.Codec.Identity(), Revision: "", Description: nil,
		}
		entry.State = invalidState
		entry.Reason = candidateEncoding
	case s.seen[p.ID] || (p.ParentRevision != "" && s.candidates[p.ParentRevision].Record().Revision == ""):
		entry.State = invalidState
		entry.Reason = identityConflict
	default:
		s.seen[p.ID] = true
		s.candidates[entry.Candidate.Revision] = candidate
		rr.Candidates = append(rr.Candidates, entry.Candidate.Revision)
		s.dispatchCandidate(&entry, candidate)
	}
	if s.ctx.Err() != nil {
		s.stop(deadlineReason)
	}
	s.projectFeedback(&entry)
	s.result.History = append(s.result.History, entry)
	f := Feedback{
		Candidate: CandidateLineage{
			ID:             entry.Candidate.ID,
			Revision:       entry.Candidate.Revision,
			ParentRevision: entry.Candidate.ParentRevision,
			Algorithm:      entry.Candidate.Algorithm,
			Codec:          entry.Candidate.Codec,
		},
		Round:      round,
		State:      entry.State,
		Reason:     entry.Reason,
		Quality:    entry.Quality,
		References: append([]string(nil), entry.References...), Measurement: zeroAggregate,
	}
	if entry.Comparison != nil {
		f.Measurement = entry.Comparison.CandidateAggregate
	}
	s.feedback = append(s.feedback, f)
	s.checkEvaluationBudget(entry)
	return s.result.State != stoppedState
}
func (s *searchExecution[T, I, R]) dispatchCandidate(entry *Evaluation, candidate Candidate[T]) {
	reservation, err := s.authorize(entry.DispatchID, s.config.EvaluationUnits)
	if err != nil {
		entry.State = failedState
		entry.Reason = evaluationBudget
		s.stop(entry.Reason)
		return
	}

	experiment, err := s.config.Evaluate(
		s.ctx,
		EvaluationRequest[T, I, R]{
			ExperimentID: evaluationExperimentID(s.config.ID, "calibration", entry.Candidate.Revision),
			SearchID:     s.config.ID,
			Phase:        "calibration",
			Candidate:    candidate,
			Dataset:      s.config.Split.Calibration,
			Budget:       s.config.Budget,
			DispatchID:   entry.DispatchID,
			Round:        entry.Round,
		},
	)

	settleErr := s.reconcile(reservation, evaly.Usage{Known: s.config.EvaluationUnits == 0, Units: 0})
	if experiment.Revision() != "" {
		if experiment.ID() != evaluationExperimentID(s.config.ID, "calibration", entry.Candidate.Revision) ||
			experiment.Record().Manifest.Dataset != s.config.Split.Calibration.Revision() {
			err = evaly.ErrConflict
		} else if _, bindingErr := evaly.RestoreExperiment(experiment.Record()); bindingErr != nil {
			err = evaly.ErrConflict
		} else {
			record := experiment.Record()
			entry.Experiment = &record
		}
	} else if err == nil {
		err = evaly.ErrUnsealed
	}
	if err != nil {
		entry.State = failedState
		entry.Reason = "evaluation_failure"
		if errors.Is(err, evaly.ErrConflict) || errors.Is(err, evaly.ErrInvalid) ||
			errors.Is(err, evaly.ErrUnsealed) {
			s.stop("evaluation_protocol_failure")
		}
		if errors.Is(err, evaly.ErrBudget) {
			s.stop(evaluationBudget)
		}
	} else {
		s.measureCandidate(entry, candidate, experiment)
	}

	if settleErr != nil {
		s.stop("evaluation_usage_failure")
	}
	if s.ctx.Err() != nil {
		s.stop(deadlineReason)
	}
}

func (s *searchExecution[T, I, R]) measureCandidate(
	entry *Evaluation,
	candidate Candidate[T],
	experiment evaly.Experiment,
) {
	comparison, err := evaly.Compare(s.config.CalibrationBaseline, experiment, s.config.Objective, s.config.Gate)
	if err != nil {
		entry.State = failedState
		entry.Reason = comparisonFailure
		s.stop(entry.Reason)
	} else {
		entry.Comparison = &comparison
		if comparison.MatchedMeansAvailable {
			q := comparison.MatchedCandidateMean
			entry.Quality = &q
		}
		entry.State = evaluatedState
		if experiment.Record().Manifest.State != sealedState ||
			comparison.CandidateAggregate.Coverage < 1 ||
			comparison.Verdict == evaly.GateInvalid ||
			comparison.Verdict == evaly.GateInconclusive ||
			entry.Quality == nil {
			entry.State = incompleteState
			entry.Reason = "ineligible_experiment"
		}
		s.checkConstraints(entry, candidate, comparison)
	}
}

func (s *searchExecution[T, I, R]) checkConstraints(
	entry *Evaluation,
	candidate Candidate[T],
	comparison evaly.Comparison,
) {
	value, ve := candidate.Value()
	if ve == nil {
		feas, ce := s.config.Constraints.Check(
			s.ctx,
			value,
			EvaluationSummary{
				State:       entry.State,
				Reason:      entry.Reason,
				Quality:     cloneScore(entry.Quality),
				Measurement: cloneAggregate(comparison.CandidateAggregate),
				Verdict:     comparison.Verdict,
			},
		)
		if s.config.Constraints.Revision() != s.result.ConstraintsRevision {
			ce = evaly.ErrConflict
		}
		entry.Feasible = feas.Feasible
		entry.FeasibilityReason = feas.Reason
		if ce != nil {
			entry.Feasible = false
			entry.FeasibilityReason = "constraint_failure"
			s.stop("constraint_failure")
		}
	} else {
		entry.State = failedState
		entry.Reason = candidateDecoding
		s.stop(candidateDecoding)
	}
}
func (s *searchExecution[T, I, R]) projectFeedback(entry *Evaluation) {
	if s.config.FeedbackProjector == nil || s.result.State == stoppedState {
		return
	}
	{
		cloned, err := cloneServiceValue(*entry)
		if err != nil {
			s.stop("feedback_encoding_failure")
			return
		}
		refs, err := s.config.FeedbackProjector.Project(s.ctx, cloned)
		if s.config.FeedbackProjector.Revision() != s.result.FeedbackRevision {
			err = evaly.ErrConflict
		}
		if err == nil {
			err = evaly.ValidateGrade(
				evaly.Grade{
					Revision: evaly.GraderRevision{
						ID:             "projection",
						Implementation: s.result.FeedbackRevision,
						Rubric:         "references-v1", Model: "", Prompt: "", Configuration: "",
					},
					Status:       evaly.NotApplicable,
					EvidenceRefs: refs,
					Dispatched:   false,
					Metrics:      nil,
					Assertions:   nil,
					Reasons:      nil,
					Usage:        evaly.Usage{Known: false, Units: 0},
				},
			)
		}
		if err != nil {
			s.stop("feedback_projection_failure")
		} else {
			entry.References = append([]string(nil), refs...)
		}
	}
}
func (s *searchExecution[T, I, R]) checkEvaluationBudget(entry Evaluation) {
	if entry.Experiment == nil {
		return
	}
	for _, trial := range entry.Experiment.Trials {
		s.checkTrialBudget(trial)
	}
}
func (s *searchExecution[T, I, R]) checkTrialBudget(trial evaly.TrialRecord) {
	if trial.Status == evaly.BudgetExhausted {
		s.stop(evaluationBudget)
	}
	for _, grade := range trial.Grades {
		for _, reason := range grade.Reasons {
			if reason == "grader_budget_exhausted" {
				s.stop(evaluationBudget)
			}
		}
	}
}

func (s *searchExecution[T, I, R]) selectWinner() {
	measured := []Evaluation{}
	for _, e := range s.result.History {
		if e.Quality != nil {
			measured = append(measured, e)
		}
	}
	sort.Slice(measured, func(i, j int) bool {
		a, b := *measured[i].Quality, *measured[j].Quality
		if a == b {
			return measured[i].Candidate.Revision < measured[j].Candidate.Revision
		}
		if s.result.Objective.Direction == "lower" {
			return a < b
		}
		return a > b
	})
	for _, e := range measured {
		s.result.Ranking = append(s.result.Ranking, e.Candidate.Revision)
	}
	if len(measured) > 0 {
		s.result.BestMeasured = measured[0].Candidate.Revision
	}
	if s.result.State != stoppedState {
		s.result.States = append(s.result.States, selectingState)
		for _, e := range measured {
			if e.State == evaluatedState && e.Feasible && e.Comparison != nil &&
				e.Comparison.Verdict == evaly.GatePass {
				s.result.Winner = e.Candidate.Revision
				break
			}
		}
	}
}
func (s *searchExecution[T, I, R]) evaluateHoldout() {
	if s.result.Winner == "" || s.result.State == stoppedState {
		return
	}
	if s.ctx.Err() != nil {
		s.stop(deadlineReason)
		return
	}
	contaminated, err := s.config.Ledger.Claim(s.ctx, s.config.Split.Holdout.Revision(), s.config.ID)
	if err != nil {
		s.stop("holdout_ledger_failure")
		return
	}
	s.result.Contaminated = contaminated
	dispatch := s.config.ID + "/holdout/" + s.result.Winner
	res, err := s.authorize(dispatch, s.config.EvaluationUnits)
	if err != nil {
		s.stop("holdout_budget")
		return
	}
	held, err := s.config.Evaluate(
		s.ctx,
		EvaluationRequest[T, I, R]{
			ExperimentID: evaluationExperimentID(s.config.ID, "holdout", s.result.Winner),
			SearchID:     s.config.ID,
			Phase:        "holdout",
			Candidate:    s.candidates[s.result.Winner],
			Dataset:      s.config.Split.Holdout,
			Budget:       s.config.Budget,
			DispatchID:   dispatch,
			Round:        len(s.result.RoundHistory),
		},
	)
	settleErr := s.reconcile(res, evaly.Usage{Known: s.config.EvaluationUnits == 0, Units: 0})
	if held.Revision() != "" {
		if held.ID() != evaluationExperimentID(s.config.ID, "holdout", s.result.Winner) ||
			held.Record().Manifest.Dataset != s.config.Split.Holdout.Revision() {
			err = evaly.ErrConflict
		} else if _, bindingErr := evaly.RestoreExperiment(held.Record()); bindingErr != nil {
			err = evaly.ErrConflict
		} else {
			record := held.Record()
			s.result.Holdout = &record
		}
	} else if err == nil {
		err = evaly.ErrUnsealed
	}
	switch {
	case err != nil:
		s.stop("holdout_evaluation_failure")
	case s.ctx.Err() != nil:
		s.stop(deadlineReason)
	default:
		comparison, err := evaly.Compare(
			s.config.HoldoutBaseline,
			held,
			s.config.Objective,
			s.config.Gate,
		)
		if err != nil {
			s.stop("holdout_comparison_failure")
		} else {
			s.result.HoldoutComparison = &comparison
		}
	}
	if settleErr != nil {
		s.stop("holdout_usage_failure")
	}
}

func (s *searchExecution[T, I, R]) stop(reason string) {
	s.result.State = stoppedState
	s.result.Reason = reason
}

func (s *searchExecution[T, I, R]) evaluateRound(round int, proposal ProposalResult[T], rr *Round) {
	for index, p := range proposal.Candidates {
		if !s.evaluateCandidate(round, index, p, rr) {
			break
		}
	}
}

func (s *searchExecution[T, I, R]) validateProposalMaximum() error {
	if bounded, ok := s.config.Proposal.(interface{ ValidateMaximum(int) error }); ok {
		if err := bounded.ValidateMaximum(s.config.MaximumCandidates); err != nil {
			return err
		}
	}
	return nil
}

func (s *searchExecution[T, I, R]) recordReceived(rr *Round, proposal ProposalResult[T]) {
	rr.ReceivedCount = len(proposal.Candidates)
	rr.ReceivedTruncated = rr.ReceivedCount > s.config.MaximumCandidates
	for index, p := range proposal.Candidates {
		if index == s.config.MaximumCandidates {
			break
		}
		rr.Received = append(
			rr.Received,
			CandidateLineage{
				ID:             p.ID,
				ParentRevision: p.ParentRevision,
				Algorithm:      s.config.Algorithm,
				Codec:          s.codecIdentity,
				Revision:       "",
			},
		)
	}
}
