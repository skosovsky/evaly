package optimizer

import (
	"reflect"
	"strconv"

	"github.com/skosovsky/evaly"
)

// validateResultSemantics binds audit fields to their authoritative attempts.
func validateResultSemantics(r Result) error {
	if r.Provenance.Training == "" || r.Provenance.Calibration == "" || r.Provenance.Holdout == "" ||
		r.Provenance.CalibrationBaseline == "" || r.Provenance.HoldoutBaseline == "" {
		return evaly.ErrInvalid
	}
	usage, err := proposalUsage(r.RoundHistory)
	if err != nil || usage != r.ProposalUsage {
		return evaly.ErrInvalid
	}
	if err := validateStatePath(r); err != nil {
		return err
	}
	audit := attemptAudit{result: r, position: 0, acceptedIDs: map[string]bool{}, acceptedRevisions: map[string]bool{}}
	for index, round := range r.RoundHistory {
		if err := audit.validateRound(index, round); err != nil {
			return err
		}
	}
	if audit.position != len(r.History) {
		return evaly.ErrInvalid
	}
	return nil
}

func validateStatePath(r Result) error {
	selected := len(r.States) == selectedPathLength
	want := []string{proposingState, r.State}
	if selected {
		want = []string{proposingState, selectingState, r.State}
	}
	if !reflect.DeepEqual(r.States, want) || (r.State == completedState && !selected) {
		return evaly.ErrInvalid
	}
	if !selected && (r.Winner != "" || r.Holdout != nil || r.HoldoutComparison != nil || r.Contaminated) {
		return evaly.ErrInvalid
	}
	if r.Winner == "" && r.Contaminated {
		return evaly.ErrInvalid
	}
	if r.State == stoppedState {
		if !validStopReason(r.Reason) {
			return evaly.ErrInvalid
		}
		if err := validateStopPhase(r, selected); err != nil {
			return err
		}
	}
	if r.State == completedState {
		return validateCompletedArtifacts(r)
	}
	return nil
}

func validateCompletedArtifacts(r Result) error {
	if r.Winner != "" {
		if r.Holdout == nil || r.HoldoutComparison == nil || r.Reason != "" {
			return evaly.ErrInvalid
		}
	} else if r.Reason != "no_selectable_candidate" || r.Holdout != nil || r.HoldoutComparison != nil {
		return evaly.ErrInvalid
	}
	return nil
}

type attemptAudit struct {
	result            Result
	position          int
	acceptedIDs       map[string]bool
	acceptedRevisions map[string]bool
}

func validateRoundMetadata(r Result, round Round) error {
	if round.ReceivedCount < len(round.Received) ||
		round.ReceivedTruncated != (round.ReceivedCount > len(round.Received)) ||
		len(round.Received) > r.MaximumCandidates {
		return evaly.ErrInvalid
	}
	if round.ReceivedTruncated &&
		(round.State != stoppedState || len(round.Candidates) != 0 || len(round.Received) != r.MaximumCandidates) {
		return evaly.ErrInvalid
	}
	if (round.State == completedState && round.Reason != "") ||
		(round.State == stoppedState && round.Reason != r.Reason) {
		return evaly.ErrInvalid
	}
	return nil
}

func (a *attemptAudit) validateRound(index int, round Round) error {
	if err := validateRoundMetadata(a.result, round); err != nil {
		return err
	}
	accepted := []string{}
	attemptIndex := 0
	for a.position < len(a.result.History) && a.result.History[a.position].Round == index {
		entry := a.result.History[a.position]
		if attemptIndex >= len(round.Received) {
			return evaly.ErrInvalid
		}
		if err := a.validateAttempt(entry, round.Received[attemptIndex], index, attemptIndex); err != nil {
			return err
		}
		if entry.State != invalidState {
			accepted = append(accepted, entry.Candidate.Revision)
		}
		attemptIndex++
		a.position++
	}
	if !equalRevisions(accepted, round.Candidates) ||
		(round.State == completedState && attemptIndex != len(round.Received)) {
		return evaly.ErrInvalid
	}
	return validateUnattempted(a.result, round.Received[attemptIndex:])
}

func validateUnattempted(r Result, received []CandidateLineage) error {
	for _, entry := range received {
		if entry.Revision != "" || entry.Algorithm != r.Algorithm || evaly.ValidateCodecIdentity(entry.Codec) != nil {
			return evaly.ErrInvalid
		}
	}
	return nil
}

func (a *attemptAudit) validateAttempt(entry Evaluation, received CandidateLineage, round, index int) error {
	if entry.DispatchID != a.result.ID+"/evaluation/"+strconv.Itoa(round)+"/"+strconv.Itoa(index) {
		return evaly.ErrInvalid
	}
	candidate := entry.Candidate
	if received.ID != candidate.ID || received.ParentRevision != candidate.ParentRevision ||
		received.Revision != candidate.Revision ||
		received.Algorithm != candidate.Algorithm ||
		received.Codec != candidate.Codec ||
		received.Algorithm != a.result.Algorithm {
		return evaly.ErrConflict
	}
	if err := validateAttemptState(entry); err != nil {
		return err
	}
	if entry.State == invalidState {
		return a.validateRejected(entry)
	}
	if a.acceptedIDs[candidate.ID] || a.acceptedRevisions[candidate.Revision] ||
		(candidate.ParentRevision != "" && !a.acceptedRevisions[candidate.ParentRevision]) {
		return evaly.ErrConflict
	}
	a.acceptedIDs[candidate.ID] = true
	a.acceptedRevisions[candidate.Revision] = true
	return nil
}

func (a *attemptAudit) validateRejected(entry Evaluation) error {
	candidate := entry.Candidate
	switch entry.Reason {
	case candidateEncoding:
		if candidate.Revision != "" || candidate.Description != nil {
			return evaly.ErrInvalid
		}
	case identityConflict:
		conflict := a.acceptedIDs[candidate.ID] || a.acceptedRevisions[candidate.Revision] ||
			(candidate.ParentRevision != "" && !a.acceptedRevisions[candidate.ParentRevision])
		if candidate.Revision == "" || !conflict {
			return evaly.ErrInvalid
		}
	}
	return nil
}

func equalRevisions(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func validateAttemptState(entry Evaluation) error {
	switch entry.State {
	case invalidState:
		if (entry.Reason != candidateEncoding && entry.Reason != identityConflict) || entry.Experiment != nil ||
			entry.Comparison != nil || entry.Quality != nil || entry.Feasible {
			return evaly.ErrInvalid
		}
	case failedState:
		return validateFailedAttempt(entry)
	case incompleteState, evaluatedState:
		return validateMeasuredAttempt(entry)
	default:
		return evaly.ErrInvalid
	}
	return nil
}

func validateFailedAttempt(entry Evaluation) error {
	if entry.Candidate.Revision == "" || entry.Feasible {
		return evaly.ErrInvalid
	}
	switch entry.Reason {
	case "evaluation_failure", evaluationBudget, comparisonFailure:
		if entry.Comparison != nil || entry.Quality != nil {
			return evaly.ErrInvalid
		}
	case candidateDecoding:
		if entry.Experiment == nil || entry.Comparison == nil {
			return evaly.ErrInvalid
		}
	default:
		return evaly.ErrInvalid
	}
	if entry.Reason == evaluationBudget && entry.Experiment != nil {
		return evaly.ErrInvalid
	}
	if entry.Reason == comparisonFailure && entry.Experiment == nil {
		return evaly.ErrInvalid
	}
	return nil
}

func validateMeasuredAttempt(entry Evaluation) error {
	if entry.Candidate.Revision == "" || entry.Experiment == nil || entry.Comparison == nil {
		return evaly.ErrInvalid
	}
	eligible := entry.Experiment.Manifest.State == sealedState && entry.Comparison.CandidateAggregate.Coverage >= 1 &&
		entry.Comparison.MatchedMeansAvailable && entry.Comparison.Verdict != evaly.GateInvalid && entry.Comparison.Verdict != evaly.GateInconclusive
	if entry.State == incompleteState {
		if entry.Reason != "ineligible_experiment" || eligible {
			return evaly.ErrInvalid
		}
	} else if !eligible || entry.Reason != "" {
		return evaly.ErrInvalid
	}
	return nil
}

// Closed categories keep persisted failure diagnostics independent of host errors.
func validStopReason(reason string) bool {
	switch reason {
	case deadlineReason,
		"candidate_limit",
		"round_limit",
		"proposal_budget",
		"proposal_failure",
		"proposal_usage_failure",
		"proposal_limit",
		evaluationBudget,
		"evaluation_protocol_failure",
		"evaluation_usage_failure",
		comparisonFailure,
		"constraint_failure",
		candidateDecoding,
		"feedback_projection_failure",
		"feedback_encoding_failure",
		"holdout_ledger_failure",
		"holdout_budget",
		"holdout_evaluation_failure",
		"holdout_comparison_failure",
		"holdout_usage_failure":
		return true
	default:
		return false
	}
}

func validateStopPhase(r Result, selected bool) error {
	if err := validateSelectedStop(r, selected); err != nil {
		return err
	}
	switch r.Reason {
	case "holdout_ledger_failure":
		if !selected || r.Holdout != nil || r.HoldoutComparison != nil || r.Contaminated {
			return evaly.ErrInvalid
		}
	case "holdout_budget":
		if !selected || r.Holdout != nil || r.HoldoutComparison != nil {
			return evaly.ErrInvalid
		}
	case "holdout_evaluation_failure":
		if !selected || r.HoldoutComparison != nil {
			return evaly.ErrInvalid
		}
	case "holdout_comparison_failure":
		if !selected || r.Holdout == nil || r.HoldoutComparison != nil {
			return evaly.ErrInvalid
		}
	case "holdout_usage_failure":
		if !selected {
			return evaly.ErrInvalid
		}
	case deadlineReason:
		if r.HoldoutComparison != nil {
			return evaly.ErrInvalid
		}
	default:
		if selected {
			return evaly.ErrInvalid
		}
	}
	return validateLimitStop(r)
}

func validateLimitStop(r Result) error {
	switch r.Reason {
	case "candidate_limit":
		if len(r.History) != r.MaximumCandidates {
			return evaly.ErrInvalid
		}
	case "round_limit":
		if len(r.RoundHistory) != r.MaximumRounds || len(r.History) >= r.MaximumCandidates {
			return evaly.ErrInvalid
		}
	}
	return nil
}

func validateSelectedStop(r Result, selected bool) error {
	if selected {
		if r.Winner == "" {
			return evaly.ErrInvalid
		}
		for _, round := range r.RoundHistory {
			if round.State != completedState {
				return evaly.ErrInvalid
			}
		}
	}
	return nil
}
