package optimizer

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
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
	if id == "" || algorithm == "" || c == nil || c.Identity().ID == "" || c.Identity().Version == "" {
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
	if c.codec == nil {
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
	Seed                  int64
}
type ProposalResult[T any] struct {
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
	Candidates                           []Candidate[T]
	Proposal                             Proposer[T, I, R]
	Codec                                evaly.Codec[T]
	Validate                             func(T) error
	Evaluate                             Evaluate[T, I, R]
	Budget                               evaly.Budget
	ProposalUnits                        float64
	Ledger                               HoldoutLedger
	CalibrationBaseline, HoldoutBaseline evaly.Experiment
	Gate                                 evaly.GatePolicy
}
type Evaluation struct {
	Candidate     CandidateRecord
	State, Reason string
	Experiment    *evaly.ExperimentRecord
	Comparison    *evaly.Comparison
	Quality       float64
}
type Result struct {
	Version                                                     int
	ID, Revision, Algorithm, Split, StopRevision, State, Reason string
	Seed                                                        int64
	History                                                     []Evaluation
	Ranking                                                     []string
	Winner                                                      string
	Holdout                                                     *evaly.ExperimentRecord
	HoldoutComparison                                           *evaly.Comparison
	Contaminated                                                bool
	ProposalRevision                                            string
	States                                                      []string
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
		Version:      1,
		ID:           c.ID,
		Algorithm:    c.Algorithm,
		Split:        c.Split.Revision,
		StopRevision: c.StopRevision,
		Seed:         c.Seed,
		State:        "proposing",
		States:       []string{"proposing"},
		History:      []Evaluation{},
		Ranking:      []string{},
	}
	if c.ID == "" || c.Algorithm == "" || c.StopRevision == "" || c.MaximumCandidates <= 0 ||
		c.MaximumCandidates > 10000 ||
		len(c.Candidates) > c.MaximumCandidates ||
		c.Timeout <= 0 ||
		c.Evaluate == nil ||
		c.Validate == nil ||
		c.Budget == nil ||
		c.Ledger == nil {
		return r, evaly.ErrInvalid
	}
	if e := validateSplit(c.Split); e != nil {
		return r, e
	}
	if c.CalibrationBaseline.Record().Manifest.Dataset != c.Split.Calibration.Revision() ||
		c.HoldoutBaseline.Record().Manifest.Dataset != c.Split.Holdout.Revision() {
		return r, evaly.ErrConflict
	}
	ctx, cancel := context.WithTimeout(ctx, c.Timeout)
	defer cancel()
	candidates := append([]Candidate[T](nil), c.Candidates...)
	stop := func(reason string) { r.State = "stopped"; r.Reason = reason }
	if c.Proposal != nil {
		if c.Codec == nil || c.Proposal.Revision() == "" {
			return r, evaly.ErrInvalid
		}
		r.ProposalRevision = c.Proposal.Revision()
		reservation, e := c.Budget.Reserve(ctx, c.ID+"/proposal", c.ProposalUnits)
		if e != nil {
			stop("proposal_budget")
		} else {
			claimErr := c.Budget.Claim(ctx, reservation)
			var proposal ProposalResult[T]
			var e error
			if claimErr != nil {
				e = claimErr
			} else {
				proposal, e = c.Proposal.Propose(
					ctx,
					ProposalRequest[I, R]{
						Training:    c.Split.Training,
						Calibration: c.Split.Calibration,
						Maximum:     c.MaximumCandidates - len(candidates),
						Seed:        c.Seed,
					},
				)
			}
			reconcileCtx, done := context.WithTimeout(context.WithoutCancel(ctx), c.Timeout)
			reconcileErr := c.Budget.Reconcile(reconcileCtx, reservation, proposal.Usage)
			done()
			if e != nil {
				stop("proposal_failure")
			} else if reconcileErr != nil {
				stop("proposal_usage_failure")
			} else {
				if len(proposal.Candidates) > c.MaximumCandidates-len(candidates) {
					stop("proposal_limit")
				}
				for _, p := range proposal.Candidates {
					if r.State == "stopped" {
						break
					}
					sealed, e := Seal(p.ID, p.Parent, c.Algorithm, p.Value, c.Codec)
					if e != nil {
						r.History = append(
							r.History,
							Evaluation{
								Candidate: CandidateRecord{ID: p.ID, Parent: p.Parent},
								State:     "invalid",
								Reason:    "candidate_encoding",
							},
						)
						continue
					}
					candidates = append(candidates, sealed)
				}
			}
		}
	}
	if r.State != "stopped" {
		r.State = "evaluating"
		r.States = append(r.States, "evaluating")
	}
	seen := map[string]bool{}
	eligible := []Evaluation{}
	for _, candidate := range candidates {
		if r.State == "stopped" {
			break
		}
		if ctx.Err() != nil {
			stop("deadline")
			break
		}
		if len(seen) >= c.MaximumCandidates {
			stop("candidate_limit")
			break
		}
		rec := candidate.Record()
		entry := Evaluation{Candidate: rec}
		if seen[rec.ID] || rec.ID == "" || rec.Revision == "" {
			entry.State = "invalid"
			entry.Reason = "identity_conflict"
			r.History = append(r.History, entry)
			continue
		}
		seen[rec.ID] = true
		value, e := candidate.Value()
		if e == nil {
			e = c.Validate(value)
		}
		if e != nil {
			entry.State = "invalid"
			entry.Reason = "host_validation"
			r.History = append(r.History, entry)
			continue
		}
		experiment, e := c.Evaluate(
			ctx,
			EvaluationRequest[T, I, R]{
				ExperimentID: c.ID + "-cal-" + rec.Revision[:16],
				SearchID:     c.ID,
				Phase:        "calibration",
				Candidate:    candidate,
				Dataset:      c.Split.Calibration,
				Budget:       c.Budget,
			},
		)
		if e == nil && experiment.ID() != c.ID+"-cal-"+rec.Revision[:16] {
			e = evaly.ErrConflict
		}
		if experiment.Revision() != "" {
			record := experiment.Record()
			entry.Experiment = &record
		}
		if e != nil {
			entry.State = "failed"
			entry.Reason = "evaluation_failure"
			r.History = append(r.History, entry)
			if errors.Is(e, evaly.ErrBudget) {
				stop("evaluation_budget")
			}
			continue
		}
		comparison, e := evaly.Compare(c.CalibrationBaseline, experiment, c.Gate)
		if e != nil {
			entry.State = "failed"
			entry.Reason = "comparison_failure"
		} else {
			entry.Comparison = &comparison
			entry.Quality = comparison.CandidateAggregate.MeanPassRate
			if experiment.Record().Manifest.State != "sealed" || comparison.CandidateAggregate.Coverage < 1 ||
				comparison.Verdict == evaly.GateInvalid ||
				comparison.Verdict == evaly.GateInconclusive {
				entry.State = "incomplete"
				entry.Reason = "ineligible_experiment"
			} else {
				entry.State = "evaluated"
				eligible = append(eligible, entry)
			}
		}
		r.History = append(r.History, entry)
		if entry.Experiment != nil {
			for _, trial := range entry.Experiment.Trials {
				if trial.Status == evaly.BudgetExhausted {
					stop("evaluation_budget")
				}
				for _, grade := range trial.Grades {
					for _, reason := range grade.Reasons {
						if reason == "grader_budget_exhausted" {
							stop("evaluation_budget")
						}
					}
				}
			}
		}
	}
	r.States = append(r.States, "selecting")
	sort.SliceStable(eligible, func(i, j int) bool { return eligible[i].Quality > eligible[j].Quality })
	for _, e := range eligible {
		r.Ranking = append(r.Ranking, e.Candidate.Revision)
	}
	if len(eligible) > 0 && r.State != "stopped" {
		r.Winner = eligible[0].Candidate.Revision
		var winner Candidate[T]
		for _, candidate := range candidates {
			if candidate.Record().Revision == r.Winner {
				winner = candidate
				break
			}
		}
		contaminated, e := c.Ledger.Claim(ctx, c.Split.Holdout.Revision(), c.ID)
		if e != nil {
			stop("holdout_ledger_failure")
		} else {
			r.Contaminated = contaminated
			held, e := c.Evaluate(
				ctx,
				EvaluationRequest[T, I, R]{
					ExperimentID: c.ID + "-hold-" + winner.Record().Revision[:16],
					SearchID:     c.ID,
					Phase:        "holdout",
					Candidate:    winner,
					Dataset:      c.Split.Holdout,
					Budget:       c.Budget,
				},
			)
			if e == nil && held.ID() != c.ID+"-hold-"+winner.Record().Revision[:16] {
				e = evaly.ErrConflict
			}
			if held.Revision() != "" {
				record := held.Record()
				r.Holdout = &record
			}
			if e != nil {
				stop("holdout_evaluation_failure")
			} else {
				comparison, e := evaly.Compare(c.HoldoutBaseline, held, c.Gate)
				if e != nil {
					stop("holdout_comparison_failure")
				} else {
					r.HoldoutComparison = &comparison
				}
			}
		}
	}
	if r.State != "stopped" {
		r.State = "completed"
		if r.Winner == "" {
			r.Reason = "no_complete_candidate"
		}
	}
	r.States = append(r.States, r.State)
	b, e := json.Marshal(r)
	if e != nil {
		return r, e
	}
	canonical, e := evaly.CanonicalJSON(b)
	if e != nil {
		return r, e
	}
	hash := sha256.Sum256(canonical)
	r.Revision = hex.EncodeToString(hash[:])
	return r, nil
}
