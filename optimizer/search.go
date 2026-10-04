package optimizer

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/skosovsky/evaly"
)

type CandidateRecord struct {
	ID, Revision, Parent, Algorithm string
	Description                     json.RawMessage
	Codec                           evaly.CodecIdentity
}
type Candidate[T any] struct {
	record CandidateRecord
	codec  evaly.Codec[T]
}

func Seal[T any](id, parent, algorithm string, v T, c evaly.Codec[T]) (Candidate[T], error) {
	if id == "" || algorithm == "" || evaly.ValidatePort(c) != nil || evaly.ValidateCodecIdentity(c.Identity()) != nil {
		return Candidate[T]{}, evaly.ErrInvalid
	}
	b, e := c.Encode(v)
	if e != nil {
		return Candidate[T]{}, e
	}
	b, e = evaly.CanonicalJSON(b)
	if e != nil {
		return Candidate[T]{}, e
	}
	r := CandidateRecord{ID: id, Parent: parent, Algorithm: algorithm, Description: b, Codec: c.Identity()}
	bytes, e := json.Marshal(r)
	if e != nil {
		return Candidate[T]{}, e
	}
	canonical, e := evaly.CanonicalJSON(bytes)
	if e != nil {
		return Candidate[T]{}, e
	}
	h := sha256.Sum256(canonical)
	r.Revision = hex.EncodeToString(h[:])
	return Candidate[T]{r, c}, nil
}
func (c Candidate[T]) Value() (T, error) {
	if evaly.ValidatePort(c.codec) != nil || c.codec.Identity() != c.record.Codec {
		var zero T
		return zero, evaly.ErrUnsealed
	}
	return c.codec.Decode(append([]byte(nil), c.record.Description...))
}
func (c Candidate[T]) Record() CandidateRecord {
	r := c.record
	r.Description = append([]byte(nil), r.Description...)
	return r
}

type Proposal[T any] struct {
	ID, Parent string
	Value      T
}
type ProposalRequest[I, R any] struct {
	Training, Calibration evaly.Dataset[I, R]
	Maximum               int
	Round                 int
	DispatchID            string
	Feedback              []Feedback
	Seed                  int64
}
type ProposalResult[T any] struct {
	Exhausted  bool
	Candidates []Proposal[T]
	Usage      evaly.Usage
}
type Proposer[T, I, R any] interface {
	Propose(context.Context, ProposalRequest[I, R]) (ProposalResult[T], error)
	Revision() string
}
type ProposalFunc[T, I, R any] struct {
	Identity string
	Generate func(context.Context, ProposalRequest[I, R]) (ProposalResult[T], error)
}

func (p ProposalFunc[T, I, R]) Revision() string { return p.Identity }
func (p ProposalFunc[T, I, R]) Validate() error {
	if p.Identity == "" || p.Generate == nil {
		return evaly.ErrInvalid
	}
	return nil
}
func (p ProposalFunc[T, I, R]) Propose(ctx context.Context, r ProposalRequest[I, R]) (ProposalResult[T], error) {
	if p.Generate == nil {
		return ProposalResult[T]{}, evaly.ErrInvalid
	}
	return p.Generate(ctx, r)
}

type Split[I, R any] struct {
	Revision                       string
	Training, Calibration, Holdout evaly.Dataset[I, R]
}
type EvaluationRequest[T, I, R any] struct {
	ExperimentID, SearchID, Phase string
	DispatchID                    string
	Round                         int
	Candidate                     Candidate[T]
	Dataset                       evaly.Dataset[I, R]
	Budget                        evaly.Budget
}
type Evaluate[T, I, R any] func(context.Context, EvaluationRequest[T, I, R]) (evaly.Experiment, error)
type HoldoutLedger interface {
	Claim(context.Context, string, string) (bool, error)
}
type MemoryLedger struct {
	mu   sync.Mutex
	uses map[string]map[string]bool
}

func (l *MemoryLedger) Claim(ctx context.Context, holdout, search string) (bool, error) {
	if e := ctx.Err(); e != nil {
		return false, e
	}
	if holdout == "" || search == "" {
		return false, evaly.ErrInvalid
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.uses == nil {
		l.uses = map[string]map[string]bool{}
	}
	if l.uses[holdout] == nil {
		l.uses[holdout] = map[string]bool{}
	}
	old := l.uses[holdout]
	if old[search] {
		return true, nil
	}
	contaminated := len(old) > 0
	old[search] = true
	return contaminated, nil
}

type Config[T, I, R any] struct {
	ID, Algorithm, StopRevision          string
	Seed                                 int64
	MaximumCandidates                    int
	Timeout                              time.Duration
	Split                                Split[I, R]
	MaximumRounds                        int
	EvaluationUnits                      float64
	Constraints                          Constraints[T]
	SplitValidator                       SplitValidator[I, R]
	FeedbackProjector                    FeedbackProjector
	Proposal                             Proposer[T, I, R]
	Codec                                evaly.Codec[T]
	Evaluate                             Evaluate[T, I, R]
	Budget                               evaly.Budget
	ProposalUnits                        float64
	Ledger                               HoldoutLedger
	CalibrationBaseline, HoldoutBaseline evaly.Experiment
	Gate                                 evaly.GatePolicy
	Objective                            evaly.Objective
}
type Evaluation struct {
	Round             int
	DispatchID        string
	Feasible          bool
	FeasibilityReason string
	References        []string
	Candidate         CandidateRecord
	State, Reason     string
	Experiment        *evaly.ExperimentRecord
	Comparison        *evaly.Comparison
	Quality           *float64
}
type Result struct {
	MaximumRounds, MaximumCandidates                                            int
	TimeoutNanoseconds                                                          int64
	ProposalUnits, EvaluationUnits                                              float64
	Version                                                                     int
	RoundHistory                                                                []Round
	BestMeasured                                                                string
	ConstraintsRevision, SplitValidationRevision, FeedbackRevision, TieRevision string
	IndependenceValidated                                                       bool
	Objective                                                                   evaly.ObjectiveIdentity
	ID, Revision, Algorithm, Split, StopRevision, State, Reason                 string
	Seed                                                                        int64
	History                                                                     []Evaluation
	Ranking                                                                     []string
	Winner                                                                      string
	Holdout                                                                     *evaly.ExperimentRecord
	HoldoutComparison                                                           *evaly.Comparison
	Contaminated                                                                bool
	ProposalRevision                                                            string
	ProposalUsage                                                               evaly.Usage
	States                                                                      []string
}

func validateSplit[I, R any](s Split[I, R]) error {
	if s.Revision == "" || s.Training.Len() == 0 || s.Calibration.Len() == 0 || s.Holdout.Len() == 0 {
		return evaly.ErrInvalid
	}
	seen := map[string]bool{}
	for _, d := range []evaly.Dataset[I, R]{s.Training, s.Calibration, s.Holdout} {
		cs, e := d.Cases()
		if e != nil {
			return e
		}
		for _, c := range cs {
			if seen[c.ID] {
				return evaly.ErrConflict
			}
			seen[c.ID] = true
		}
	}
	return nil
}
func Search[T, I, R any](ctx context.Context, c Config[T, I, R]) (Result, error) {
	r := Result{
		Version:            4,
		ID:                 c.ID,
		Algorithm:          c.Algorithm,
		Split:              c.Split.Revision,
		StopRevision:       c.StopRevision,
		Seed:               c.Seed,
		State:              "proposing",
		States:             []string{"proposing"},
		History:            []Evaluation{},
		Ranking:            []string{},
		RoundHistory:       []Round{},
		TieRevision:        TieRevision,
		MaximumRounds:      c.MaximumRounds,
		MaximumCandidates:  c.MaximumCandidates,
		TimeoutNanoseconds: int64(c.Timeout),
		ProposalUnits:      c.ProposalUnits,
		EvaluationUnits:    c.EvaluationUnits,
	}
	if c.ID == "" || c.Algorithm == "" || c.StopRevision == "" || c.MaximumCandidates <= 0 ||
		c.MaximumCandidates > 10000 ||
		c.MaximumRounds <= 0 ||
		c.MaximumRounds > 10000 ||
		c.Timeout <= 0 ||
		c.Evaluate == nil ||
		math.IsNaN(c.ProposalUnits) ||
		math.IsInf(c.ProposalUnits, 0) ||
		c.ProposalUnits < 0 ||
		math.IsNaN(c.EvaluationUnits) ||
		math.IsInf(c.EvaluationUnits, 0) ||
		c.EvaluationUnits < 0 {
		return r, evaly.ErrInvalid
	}
	for _, p := range []any{c.Budget, c.Ledger, c.Proposal, c.Codec, c.Objective, c.Constraints} {
		if e := evaly.ValidatePort(p); e != nil {
			return r, e
		}
	}
	if e := validateSplit(c.Split); e != nil {
		return r, e
	}
	r.Objective = c.Objective.Identity()
	r.ProposalRevision = c.Proposal.Revision()
	r.ConstraintsRevision = c.Constraints.Revision()
	if evaly.ValidateObjectiveIdentity(r.Objective) != nil || evaly.ValidateCodecIdentity(c.Codec.Identity()) != nil ||
		r.ProposalRevision == "" ||
		r.ConstraintsRevision == "" {
		return r, evaly.ErrInvalid
	}
	if _, e := evaly.NewEnvelope("search", c.ID, struct{}{}); e != nil {
		return r, e
	}
	if c.CalibrationBaseline.Record().Manifest.Dataset != c.Split.Calibration.Revision() ||
		c.HoldoutBaseline.Record().Manifest.Dataset != c.Split.Holdout.Revision() {
		return r, evaly.ErrConflict
	}
	for _, b := range []evaly.Experiment{c.CalibrationBaseline, c.HoldoutBaseline} {
		if _, e := evaly.RestoreExperiment(b.Record()); e != nil {
			return r, e
		}
		if b.Record().Manifest.State != "sealed" {
			return r, evaly.ErrUnsealed
		}
	}
	ctx, cancel := context.WithTimeout(ctx, c.Timeout)
	defer cancel()
	// Calibration alone exercises objective preflight; holdout measurement is deferred.
	cmp, e := evaly.Compare(c.CalibrationBaseline, c.CalibrationBaseline, c.Objective, c.Gate)
	if e != nil {
		return r, e
	}
	if cmp.Verdict == evaly.GateInvalid {
		return r, evaly.ErrInvalid
	}
	if c.SplitValidator != nil {
		if e := evaly.ValidatePort(c.SplitValidator); e != nil {
			return r, e
		}
		r.SplitValidationRevision = c.SplitValidator.Revision()
		if r.SplitValidationRevision == "" {
			return r, evaly.ErrInvalid
		}
		if e := c.SplitValidator.ValidateSplit(ctx, c.Split); e != nil {
			return r, e
		}
		if c.SplitValidator.Revision() != r.SplitValidationRevision {
			return r, evaly.ErrConflict
		}
		r.IndependenceValidated = true
	}
	if c.FeedbackProjector != nil {
		if e := evaly.ValidatePort(c.FeedbackProjector); e != nil {
			return r, e
		}
		r.FeedbackRevision = c.FeedbackProjector.Revision()
		if r.FeedbackRevision == "" {
			return r, evaly.ErrInvalid
		}
	}
	stop := func(reason string) { r.State = "stopped"; r.Reason = reason }
	authorize := func(id string, units float64) (evaly.Reservation, error) {
		if e := ctx.Err(); e != nil {
			return evaly.Reservation{}, e
		}
		res, e := c.Budget.Reserve(ctx, id, units)
		if e != nil {
			return res, e
		}
		if e = c.Budget.Claim(ctx, res); e != nil {
			return res, e
		}
		if e := ctx.Err(); e != nil {
			return res, e
		}
		return res, nil
	}
	reconcile := func(res evaly.Reservation, u evaly.Usage) error {
		rc, done := context.WithTimeout(context.WithoutCancel(ctx), c.Timeout)
		defer done()
		return c.Budget.Reconcile(rc, res, u)
	}
	codecIdentity := c.Codec.Identity()
	attempts := 0
	seen := map[string]bool{}
	candidates := map[string]Candidate[T]{}
	feedback := []Feedback{}
	exhausted := false
	for round := 0; round < c.MaximumRounds && r.State != "stopped"; round++ {
		if ctx.Err() != nil {
			stop("deadline")
			break
		}
		if attempts >= c.MaximumCandidates {
			stop("candidate_limit")
			break
		}
		dispatch := c.ID + "/proposal/" + fmt.Sprint(round)
		rr := Round{
			Index:            round,
			DispatchID:       dispatch,
			ProposalRevision: r.ProposalRevision,
			State:            "proposing",
			Candidates:       []string{},
			Received:         []CandidateLineage{},
		}
		res, err := authorize(dispatch, c.ProposalUnits)
		if err != nil {
			rr.State = "stopped"
			rr.Reason = "proposal_budget"
			r.RoundHistory = append(r.RoundHistory, rr)
			stop(rr.Reason)
			break
		}
		// Clone service feedback so a proposer cannot mutate persisted history.
		fbBytes, _ := json.Marshal(feedback)
		var fb []Feedback
		_ = json.Unmarshal(fbBytes, &fb)
		proposal, err := c.Proposal.Propose(
			ctx,
			ProposalRequest[I, R]{
				Training:    c.Split.Training,
				Calibration: c.Split.Calibration,
				Maximum:     c.MaximumCandidates - attempts,
				Seed:        c.Seed,
				Round:       round,
				DispatchID:  dispatch,
				Feedback:    fb,
			},
		)
		if math.IsNaN(proposal.Usage.Units) || math.IsInf(proposal.Usage.Units, 0) || proposal.Usage.Units < 0 {
			proposal.Usage = evaly.Usage{}
			err = evaly.ErrInvalid
		}
		for _, p := range proposal.Candidates {
			rr.Received = append(
				rr.Received,
				CandidateLineage{ID: p.ID, Parent: p.Parent, Algorithm: c.Algorithm, Codec: c.Codec.Identity()},
			)
		}
		rr.Usage = proposal.Usage
		r.ProposalUsage.Units += proposal.Usage.Units
		if round == 0 {
			r.ProposalUsage.Known = proposal.Usage.Known
		} else {
			r.ProposalUsage.Known = r.ProposalUsage.Known && proposal.Usage.Known
		}
		if c.Proposal.Revision() != r.ProposalRevision || c.Codec.Identity() != codecIdentity {
			err = evaly.ErrConflict
		}
		usageErr := reconcile(res, proposal.Usage)
		if err != nil || usageErr != nil || ctx.Err() != nil {
			rr.State = "stopped"
			rr.Reason = "proposal_failure"
			if ctx.Err() != nil {
				rr.Reason = "deadline"
			}
			if usageErr != nil {
				rr.Reason = "proposal_usage_failure"
			}
			r.RoundHistory = append(r.RoundHistory, rr)
			stop(rr.Reason)
			break
		}
		if len(proposal.Candidates) > c.MaximumCandidates-attempts {
			rr.State = "stopped"
			rr.Reason = "proposal_limit"
			r.RoundHistory = append(r.RoundHistory, rr)
			stop(rr.Reason)
			break
		}
		if len(proposal.Candidates) == 0 {
			rr.State = "completed"
			r.RoundHistory = append(r.RoundHistory, rr)
			exhausted = true
			break
		}
		for index, p := range proposal.Candidates {
			attempts++
			if ctx.Err() != nil {
				stop("deadline")
				break
			}
			entry := Evaluation{
				Round:      round,
				DispatchID: c.ID + "/evaluation/" + fmt.Sprint(round) + "/" + fmt.Sprint(index),
				References: []string{},
			}
			candidate, err := Seal(p.ID, p.Parent, c.Algorithm, p.Value, c.Codec)
			entry.Candidate = candidate.Record()
			if err != nil {
				entry.Candidate = CandidateRecord{
					ID:        p.ID,
					Parent:    p.Parent,
					Algorithm: c.Algorithm,
					Codec:     c.Codec.Identity(),
				}
				entry.State = "invalid"
				entry.Reason = "candidate_encoding"
			} else if seen[p.ID] || (p.Parent != "" && candidates[p.Parent].Record().Revision == "") {
				entry.State = "invalid"
				entry.Reason = "identity_conflict"
			} else {
				seen[p.ID] = true
				candidates[entry.Candidate.Revision] = candidate
				rr.Candidates = append(rr.Candidates, entry.Candidate.Revision)
				reservation, err := authorize(entry.DispatchID, c.EvaluationUnits)
				if err != nil {
					entry.State = "failed"
					entry.Reason = "evaluation_budget"
					stop(entry.Reason)
				} else {
					experiment, err := c.Evaluate(
						ctx,
						EvaluationRequest[T, I, R]{
							ExperimentID: evaluationExperimentID(c.ID, "calibration", entry.Candidate.Revision),
							SearchID:     c.ID,
							Phase:        "calibration",
							Candidate:    candidate,
							Dataset:      c.Split.Calibration,
							Budget:       c.Budget,
							DispatchID:   entry.DispatchID,
							Round:        round,
						},
					)
					// This reservation accounts only for dispatch overhead; actual target/grader costs are accounted by Evaluate.
					settleErr := reconcile(reservation, evaly.Usage{Known: c.EvaluationUnits == 0})
					if experiment.Revision() != "" {
						record := experiment.Record()
						entry.Experiment = &record
					}
					if err == nil &&
						experiment.ID() != evaluationExperimentID(c.ID, "calibration", entry.Candidate.Revision) {
						err = evaly.ErrConflict
					}
					if err != nil {
						entry.State = "failed"
						entry.Reason = "evaluation_failure"
						if errors.Is(err, evaly.ErrConflict) || errors.Is(err, evaly.ErrInvalid) ||
							errors.Is(err, evaly.ErrUnsealed) {
							stop("evaluation_protocol_failure")
						}
						if errors.Is(err, evaly.ErrBudget) {
							stop("evaluation_budget")
						}
					} else {
						comparison, err := evaly.Compare(c.CalibrationBaseline, experiment, c.Objective, c.Gate)
						if err != nil {
							entry.State = "failed"
							entry.Reason = "comparison_failure"
							stop(entry.Reason)
						} else {
							entry.Comparison = &comparison
							if comparison.MatchedMeansAvailable {
								q := comparison.MatchedCandidateMean
								entry.Quality = &q
							}
							entry.State = "evaluated"
							if experiment.Record().Manifest.State != "sealed" ||
								comparison.CandidateAggregate.Coverage < 1 ||
								comparison.Verdict == evaly.GateInvalid ||
								comparison.Verdict == evaly.GateInconclusive ||
								entry.Quality == nil {
								entry.State = "incomplete"
								entry.Reason = "ineligible_experiment"
							}
							value, ve := candidate.Value()
							if ve == nil {
								feas, ce := c.Constraints.Check(
									ctx,
									value,
									EvaluationSummary{
										State:       entry.State,
										Reason:      entry.Reason,
										Quality:     cloneScore(entry.Quality),
										Measurement: cloneAggregate(comparison.CandidateAggregate),
										Verdict:     comparison.Verdict,
									},
								)
								if c.Constraints.Revision() != r.ConstraintsRevision {
									ce = evaly.ErrConflict
								}
								entry.Feasible = feas.Feasible
								entry.FeasibilityReason = feas.Reason
								if ce != nil {
									entry.Feasible = false
									entry.FeasibilityReason = "constraint_failure"
									stop("constraint_failure")
								}
							} else {
								entry.State = "invalid"
								entry.Reason = "candidate_decoding"
							}
						}
					}
					if settleErr != nil {
						stop("evaluation_usage_failure")
					}
					if ctx.Err() != nil {
						stop("deadline")
					}
				}
			}
			if ctx.Err() != nil {
				stop("deadline")
			}
			if c.FeedbackProjector != nil && r.State != "stopped" {
				refs, err := c.FeedbackProjector.Project(ctx, cloneEvaluation(entry))
				if c.FeedbackProjector.Revision() != r.FeedbackRevision {
					err = evaly.ErrConflict
				}
				if err == nil {
					err = evaly.ValidateGrade(
						evaly.Grade{
							Revision: evaly.GraderRevision{
								ID:             "projection",
								Implementation: r.FeedbackRevision,
								Rubric:         "references-v1",
							},
							Status:       evaly.NotApplicable,
							EvidenceRefs: refs,
						},
					)
				}
				if err != nil {
					stop("feedback_projection_failure")
				} else {
					entry.References = append([]string(nil), refs...)
				}
			}
			r.History = append(r.History, entry)
			f := Feedback{
				Candidate: CandidateLineage{
					ID:        entry.Candidate.ID,
					Revision:  entry.Candidate.Revision,
					Parent:    entry.Candidate.Parent,
					Algorithm: entry.Candidate.Algorithm,
					Codec:     entry.Candidate.Codec,
				},
				Round:      round,
				State:      entry.State,
				Reason:     entry.Reason,
				Quality:    entry.Quality,
				References: append([]string(nil), entry.References...),
			}
			if entry.Comparison != nil {
				f.Measurement = entry.Comparison.CandidateAggregate
			}
			feedback = append(feedback, f)
			if entry.Experiment != nil {
				for _, t := range entry.Experiment.Trials {
					if t.Status == evaly.BudgetExhausted {
						stop("evaluation_budget")
					}
					for _, g := range t.Grades {
						for _, reason := range g.Reasons {
							if reason == "grader_budget_exhausted" {
								stop("evaluation_budget")
							}
						}
					}
				}
			}
			if r.State == "stopped" {
				break
			}
		}
		rr.State = "completed"
		if r.State == "stopped" {
			rr.State = "stopped"
			rr.Reason = r.Reason
		}
		r.RoundHistory = append(r.RoundHistory, rr)
		if proposal.Exhausted && r.State != "stopped" {
			exhausted = true
			break
		}
	}
	if !exhausted && r.State != "stopped" {
		if attempts >= c.MaximumCandidates {
			stop("candidate_limit")
		} else {
			stop("round_limit")
		}
	}
	measured := []Evaluation{}
	for _, e := range r.History {
		if e.Quality != nil {
			measured = append(measured, e)
		}
	}
	sort.Slice(measured, func(i, j int) bool {
		a, b := *measured[i].Quality, *measured[j].Quality
		if a == b {
			return measured[i].Candidate.Revision < measured[j].Candidate.Revision
		}
		if r.Objective.Direction == "lower" {
			return a < b
		}
		return a > b
	})
	for _, e := range measured {
		r.Ranking = append(r.Ranking, e.Candidate.Revision)
	}
	if len(measured) > 0 {
		r.BestMeasured = measured[0].Candidate.Revision
	}
	if r.State != "stopped" {
		r.States = append(r.States, "selecting")
		for _, e := range measured {
			if e.State == "evaluated" && e.Feasible && e.Comparison != nil && e.Comparison.Verdict == evaly.GatePass {
				r.Winner = e.Candidate.Revision
				break
			}
		}
	}
	if r.Winner != "" && r.State != "stopped" {
		if ctx.Err() != nil {
			stop("deadline")
		} else {
			contaminated, err := c.Ledger.Claim(ctx, c.Split.Holdout.Revision(), c.ID)
			if err != nil {
				stop("holdout_ledger_failure")
			} else {
				r.Contaminated = contaminated
				dispatch := c.ID + "/holdout/" + r.Winner
				res, err := authorize(dispatch, c.EvaluationUnits)
				if err != nil {
					stop("holdout_budget")
				} else {
					held, err := c.Evaluate(
						ctx,
						EvaluationRequest[T, I, R]{
							ExperimentID: evaluationExperimentID(c.ID, "holdout", r.Winner),
							SearchID:     c.ID,
							Phase:        "holdout",
							Candidate:    candidates[r.Winner],
							Dataset:      c.Split.Holdout,
							Budget:       c.Budget,
							DispatchID:   dispatch,
							Round:        len(r.RoundHistory),
						},
					)
					settleErr := reconcile(res, evaly.Usage{Known: c.EvaluationUnits == 0})
					if held.Revision() != "" {
						record := held.Record()
						r.Holdout = &record
					}
					if err == nil && held.ID() != evaluationExperimentID(c.ID, "holdout", r.Winner) {
						err = evaly.ErrConflict
					}
					if err != nil {
						stop("holdout_evaluation_failure")
					} else if ctx.Err() != nil {
						stop("deadline")
					} else {
						comparison, err := evaly.Compare(c.HoldoutBaseline, held, c.Objective, c.Gate)
						if err != nil {
							stop("holdout_comparison_failure")
						} else {
							r.HoldoutComparison = &comparison
						}
					}
					if settleErr != nil {
						stop("holdout_usage_failure")
					}
				}
			}
		}
	}
	if r.State != "stopped" {
		r.State = "completed"
		if r.Winner == "" {
			r.Reason = "no_selectable_candidate"
		}
	}
	r.States = append(r.States, r.State)
	if e := sealResult(&r); e != nil {
		return r, e
	}
	return r, nil
}

// All identity parts participate in the digest; generated artifact IDs stay
// within the core identifier limit even when the search ID uses all 128 bytes.
func evaluationExperimentID(search, phase, revision string) string {
	digest := sha256.Sum256([]byte(search + "\x00" + phase + "\x00" + revision))
	return "evaly-" + phase + "-" + hex.EncodeToString(digest[:])
}

func sealResult(r *Result) error {
	copy := *r
	copy.Revision = ""
	b, e := json.Marshal(copy)
	if e != nil {
		return e
	}
	b, e = evaly.CanonicalJSON(b)
	if e != nil {
		return e
	}
	h := sha256.Sum256(b)
	r.Revision = hex.EncodeToString(h[:])
	return nil
}

func cloneScore(q *float64) *float64 {
	if q == nil {
		return nil
	}
	v := *q
	return &v
}
func cloneAggregate(a evaly.Aggregate) evaly.Aggregate {
	a.Excluded = append([]evaly.Exclusion(nil), a.Excluded...)
	return a
}
func cloneEvaluation(e Evaluation) Evaluation {
	b, _ := json.Marshal(e)
	var out Evaluation
	_ = json.Unmarshal(b, &out)
	return out
}
