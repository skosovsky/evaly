package optimizer

import (
	"context"

	"github.com/skosovsky/evaly"
)

const TieRevision = "candidate-revision-lexical-v1"

type Feasibility struct {
	Feasible bool
	Reason   string
}
type EvaluationSummary struct {
	Verdict       evaly.GateVerdict
	State, Reason string
	Quality       *float64
	Measurement   evaly.Aggregate
}
type Constraints[T any] interface {
	Revision() string
	Check(context.Context, T, EvaluationSummary) (Feasibility, error)
}
type ConstraintsFunc[T any] struct {
	Identity string
	Assess   func(context.Context, T, EvaluationSummary) (Feasibility, error)
}

func (c ConstraintsFunc[T]) Revision() string { return c.Identity }
func (c ConstraintsFunc[T]) Validate() error {
	if c.Identity == "" || c.Assess == nil {
		return evaly.ErrInvalid
	}
	return nil
}
func (c ConstraintsFunc[T]) Check(ctx context.Context, v T, e EvaluationSummary) (Feasibility, error) {
	if c.Validate() != nil {
		return Feasibility{}, evaly.ErrInvalid
	}
	if err := ctx.Err(); err != nil {
		return Feasibility{}, err
	}
	return c.Assess(ctx, v, e)
}

type CandidateLineage struct {
	ID, Revision, Parent, Algorithm string
	Codec                           evaly.CodecIdentity
}
type Feedback struct {
	Candidate     CandidateLineage
	Round         int
	State, Reason string
	Quality       *float64
	Measurement   evaly.Aggregate
	References    []string
}
type FeedbackProjector interface {
	Revision() string
	Project(context.Context, Evaluation) ([]string, error)
}
type FeedbackProjectionFunc struct {
	Identity        string
	ProjectFeedback func(context.Context, Evaluation) ([]string, error)
}

func (p FeedbackProjectionFunc) Revision() string { return p.Identity }
func (p FeedbackProjectionFunc) Validate() error {
	if p.Identity == "" || p.ProjectFeedback == nil {
		return evaly.ErrInvalid
	}
	return nil
}
func (p FeedbackProjectionFunc) Project(ctx context.Context, e Evaluation) ([]string, error) {
	if p.Validate() != nil {
		return nil, evaly.ErrInvalid
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return p.ProjectFeedback(ctx, e)
}

type StaticProposer[T, I, R any] struct {
	identity   string
	candidates []Candidate[T]
}

func NewStaticProposer[T, I, R any](identity string, candidates []Candidate[T]) StaticProposer[T, I, R] {
	return StaticProposer[T, I, R]{identity, append([]Candidate[T](nil), candidates...)}
}
func (p StaticProposer[T, I, R]) Revision() string { return p.identity }
func (p StaticProposer[T, I, R]) Validate() error {
	if p.identity == "" {
		return evaly.ErrInvalid
	}
	return nil
}
func (p StaticProposer[T, I, R]) Propose(ctx context.Context, r ProposalRequest[I, R]) (ProposalResult[T], error) {
	if err := ctx.Err(); err != nil {
		return ProposalResult[T]{}, err
	}
	if r.Round != 0 {
		return ProposalResult[T]{Candidates: []Proposal[T]{}, Usage: evaly.Usage{Known: true}}, nil
	}
	out := ProposalResult[T]{Exhausted: true, Candidates: []Proposal[T]{}, Usage: evaly.Usage{Known: true}}
	for _, candidate := range p.candidates {
		value, err := candidate.Value()
		if err != nil {
			return out, err
		}
		rec := candidate.Record()
		out.Candidates = append(out.Candidates, Proposal[T]{ID: rec.ID, Parent: rec.Parent, Value: value})
	}
	return out, nil
}

type Round struct {
	Index            int
	DispatchID       string
	ProposalRevision string
	State, Reason    string
	Usage            evaly.Usage
	Candidates       []string
	Received         []CandidateLineage
}
