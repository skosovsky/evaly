package optimizer

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"math"
	"reflect"
	"sort"
	"strconv"

	"github.com/skosovsky/evaly"
)

func RestoreCandidate[T any](r CandidateRecord, c evaly.Codec[T]) (Candidate[T], error) {
	if err := evaly.ValidatePort(c); err != nil {
		return Candidate[T]{}, err
	}
	if err := evaly.ValidateCodecIdentity(c.Identity()); err != nil {
		return Candidate[T]{}, err
	}
	if r.Codec != c.Identity() {
		return Candidate[T]{}, evaly.ErrUnsupported
	}
	value, e := c.Decode(append([]byte(nil), r.Description...))
	if e != nil {
		return Candidate[T]{}, e
	}
	candidate, e := Seal(r.ID, r.ParentRevision, r.Algorithm, value, c)
	if e != nil {
		return Candidate[T]{}, e
	}
	if !reflect.DeepEqual(candidate.Record(), r) {
		return Candidate[T]{}, evaly.ErrConflict
	}
	return candidate, nil
}

// ValidateResult verifies persisted protocol semantics without dispatching host callbacks.
func ValidateResult(r Result) error {
	if err := validateResultEnvelope(r); err != nil {
		return err
	}
	if err := validateResultSemantics(r); err != nil {
		return err
	}
	dispatch := map[string]bool{}
	candidate := map[string]Evaluation{}
	if err := validateResultRounds(r, dispatch); err != nil {
		return err
	}
	for _, entry := range r.History {
		if err := validateResultEvaluation(r, entry, dispatch, candidate); err != nil {
			return err
		}
	}
	expected, err := validateResultRanking(r, candidate)
	if err != nil {
		return err
	}
	if err = validateResultSelection(r, candidate, expected); err != nil {
		return err
	}
	return validateResultHoldout(r)
}

func RestoreResult[T any](r Result, c evaly.Codec[T]) (Result, error) {
	if e := ValidateResult(r); e != nil {
		return Result{}, e
	}
	if e := evaly.ValidatePort(c); e != nil {
		return Result{}, e
	}
	if e := evaly.ValidateCodecIdentity(c.Identity()); e != nil {
		return Result{}, e
	}
	for _, entry := range r.History {
		if entry.Candidate.Revision != "" {
			if _, e := RestoreCandidate(entry.Candidate, c); e != nil {
				return Result{}, e
			}
		}
	}
	b, e := json.Marshal(r)
	if e != nil {
		return Result{}, e
	}
	var out Result
	if e = json.Unmarshal(b, &out); e != nil {
		return Result{}, e
	}
	return out, nil
}

func validateComparison(c evaly.Comparison) error {
	if c.Version != 2 || evaly.ValidateObjectiveIdentity(c.Objective) != nil {
		return evaly.ErrInvalid
	}
	revision := c.Revision
	c.Revision = ""
	b, e := json.Marshal(c)
	if e != nil {
		return e
	}
	b, e = evaly.CanonicalJSON(b)
	if e != nil {
		return e
	}
	h := sha256.Sum256(b)
	if hex.EncodeToString(h[:]) != revision {
		return evaly.ErrConflict
	}
	return nil
}

func validateCandidateRecord(r CandidateRecord) error {
	if r.ID == "" || r.Algorithm == "" || evaly.ValidateCodecIdentity(r.Codec) != nil {
		return evaly.ErrInvalid
	}
	revision := r.Revision
	r.Revision = ""
	b, e := json.Marshal(r)
	if e != nil {
		return e
	}
	b, e = evaly.CanonicalJSON(b)
	if e != nil {
		return e
	}
	h := sha256.Sum256(b)
	if hex.EncodeToString(h[:]) != revision {
		return evaly.ErrConflict
	}
	return nil
}

func validateResultEnvelope(r Result) error {
	if r.Version != resultWireRevision {
		return evaly.ErrUnsupported
	}

	wire, wireErr := json.Marshal(r)
	if wireErr != nil {
		return evaly.ErrInvalid
	}
	if _, err := evaly.DecodeWire[Result](wire); err != nil {
		return err
	}
	if _, err := evaly.NewEnvelope("search", r.ID, struct{}{}); err != nil {
		return err
	}
	if r.ID == "" || r.Revision == "" || r.Algorithm == "" || r.Split == "" || r.StopRevision == "" ||
		r.ProposalRevision == "" ||
		r.ConstraintsRevision == "" ||
		r.TieRevision != TieRevision ||
		r.MaximumRounds <= 0 ||
		r.MaximumRounds > 10000 ||
		r.MaximumCandidates <= 0 ||
		r.MaximumCandidates > 10000 ||
		r.TimeoutNanoseconds <= 0 ||
		evaly.ValidateObjectiveIdentity(r.Objective) != nil {
		return evaly.ErrInvalid
	}
	if r.State != completedState && r.State != stoppedState {
		return evaly.ErrInvalid
	}
	if r.State == stoppedState && r.Reason == "" || r.IndependenceValidated != (r.SplitValidationRevision != "") {
		return evaly.ErrInvalid
	}
	cloned := r
	if e := sealResult(&cloned); e != nil {
		return e
	}
	if cloned.Revision != r.Revision {
		return evaly.ErrConflict
	}
	if len(r.RoundHistory) > r.MaximumRounds || len(r.History) > r.MaximumCandidates {
		return evaly.ErrInvalid
	}
	return nil
}

func validateResultRounds(r Result, dispatch map[string]bool) error {
	for index, round := range r.RoundHistory {
		if round.Index != index || round.DispatchID != r.ID+"/proposal/"+strconv.Itoa(index) ||
			round.ProposalRevision != r.ProposalRevision ||
			(round.State != completedState && round.State != stoppedState) ||
			dispatch[round.DispatchID] {
			return evaly.ErrInvalid
		}
		dispatch[round.DispatchID] = true
		if round.State == stoppedState &&
			(r.State != stoppedState || index != len(r.RoundHistory)-1 || round.Reason == "") {
			return evaly.ErrInvalid
		}
	}
	return nil
}

func validateEvaluationArtifacts(r Result, entry Evaluation) error {
	if entry.Experiment != nil {
		if entry.Experiment.Manifest.ID != evaluationExperimentID(r.ID, "calibration", entry.Candidate.Revision) ||
			entry.Experiment.Manifest.Dataset != r.Provenance.Calibration {
			return evaly.ErrConflict
		}
		if _, e := evaly.RestoreExperiment(*entry.Experiment); e != nil {
			return e
		}
	}
	if entry.Comparison == nil {
		return nil
	}
	if entry.Experiment == nil || entry.Comparison.Candidate != entry.Experiment.Manifest.Revision ||
		entry.Comparison.Objective != r.Objective || entry.Comparison.Baseline != r.Provenance.CalibrationBaseline {
		return evaly.ErrConflict
	}
	if entry.Comparison.MatchedMeansAvailable {
		if entry.Quality == nil || *entry.Quality != entry.Comparison.MatchedCandidateMean {
			return evaly.ErrInvalid
		}
	} else if entry.Quality != nil {
		return evaly.ErrInvalid
	}

	if e := validateComparison(*entry.Comparison); e != nil {
		return e
	}

	return nil
}

func validateEvaluationReferences(r Result, entry Evaluation) error {
	if len(entry.References) > 0 {
		if r.FeedbackRevision == "" {
			return evaly.ErrInvalid
		}
		if e := evaly.ValidateGrade(
			evaly.Grade{
				Revision: evaly.GraderRevision{
					ID:             "projection",
					Implementation: r.FeedbackRevision,
					Rubric:         "references-v1", Model: "", Prompt: "", Configuration: "",
				},
				Status:       evaly.NotApplicable,
				EvidenceRefs: entry.References,
				Dispatched:   false,
				Metrics:      nil,
				Assertions:   nil,
				Reasons:      nil,
				Usage:        evaly.Usage{Known: false, Units: 0},
			},
		); e != nil {
			return e
		}
	}
	return nil
}

func validateResultEvaluation(
	r Result,
	entry Evaluation,
	dispatch map[string]bool,
	candidate map[string]Evaluation,
) error {
	if entry.Round < 0 || entry.Round >= len(r.RoundHistory) || entry.DispatchID == "" ||
		dispatch[entry.DispatchID] {
		return evaly.ErrInvalid
	}
	dispatch[entry.DispatchID] = true
	if entry.Quality != nil && entry.Comparison == nil {
		return evaly.ErrInvalid
	}
	switch entry.State {
	case invalidState, failedState, incompleteState, evaluatedState:
	default:
		return evaly.ErrInvalid
	}
	if entry.Candidate.Revision != "" {
		if e := validateCandidateRecord(entry.Candidate); e != nil {
			return e
		}

		if entry.State != invalidState {
			if _, ok := candidate[entry.Candidate.Revision]; ok {
				return evaly.ErrConflict
			}
			candidate[entry.Candidate.Revision] = entry
		}
	}
	if err := validateEvaluationArtifacts(r, entry); err != nil {
		return err
	}
	return validateEvaluationReferences(r, entry)
}

func validateResultRanking(r Result, candidate map[string]Evaluation) ([]Evaluation, error) {
	expected := []Evaluation{}
	for _, entry := range r.History {
		if entry.Quality != nil {
			if invalidQuality(*entry.Quality) {
				return nil, evaly.ErrInvalid
			}
			expected = append(expected, entry)
		}
	}
	sort.Slice(expected, func(i, j int) bool {
		a, b := *expected[i].Quality, *expected[j].Quality
		if a == b {
			return expected[i].Candidate.Revision < expected[j].Candidate.Revision
		}
		if r.Objective.Direction == "lower" {
			return a < b
		}
		return a > b
	})
	if len(expected) != len(r.Ranking) {
		return nil, evaly.ErrInvalid
	}
	for i, entry := range expected {
		if r.Ranking[i] != entry.Candidate.Revision {
			return nil, evaly.ErrInvalid
		}
	}
	seen := map[string]bool{}
	for _, revision := range r.Ranking {
		entry, ok := candidate[revision]
		if !ok || entry.Quality == nil || seen[revision] {
			return nil, evaly.ErrInvalid
		}
		seen[revision] = true
	}
	wantBest := ""
	if len(r.Ranking) > 0 {
		wantBest = r.Ranking[0]
	}
	if r.BestMeasured != wantBest {
		return nil, evaly.ErrInvalid
	}
	return expected, nil
}

func validateResultSelection(r Result, candidate map[string]Evaluation, expected []Evaluation) error {
	if len(r.States) == selectedPathLength {
		want := ""
		for _, entry := range expected {
			if entry.State == evaluatedState && entry.Feasible && entry.Comparison != nil &&
				entry.Comparison.Verdict == evaly.GatePass {
				want = entry.Candidate.Revision
				break
			}
		}
		if want != r.Winner {
			return evaly.ErrInvalid
		}
	}
	if r.Winner != "" {
		entry, ok := candidate[r.Winner]
		if !ok || !entry.Feasible || entry.State != evaluatedState || entry.Comparison == nil ||
			entry.Comparison.Verdict != evaly.GatePass {
			return evaly.ErrInvalid
		}
	}
	return nil
}

func validateResultHoldout(r Result) error {
	if r.Holdout != nil {
		if r.Winner == "" {
			return evaly.ErrInvalid
		}
		if r.Holdout.Manifest.ID != evaluationExperimentID(r.ID, "holdout", r.Winner) ||
			r.Holdout.Manifest.Dataset != r.Provenance.Holdout {
			return evaly.ErrConflict
		}
		if _, e := evaly.RestoreExperiment(*r.Holdout); e != nil {
			return e
		}
	}
	if r.HoldoutComparison != nil {
		if r.Holdout == nil {
			return evaly.ErrInvalid
		}
		if r.HoldoutComparison.Candidate != r.Holdout.Manifest.Revision ||
			r.HoldoutComparison.Objective != r.Objective || r.HoldoutComparison.Baseline != r.Provenance.HoldoutBaseline {
			return evaly.ErrConflict
		}
		if e := validateComparison(*r.HoldoutComparison); e != nil {
			return e
		}
	}
	return nil
}

func invalidQuality(value float64) bool { return math.IsNaN(value) || math.IsInf(value, 0) }
