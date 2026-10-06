package optimizer

import (
	"context"

	"github.com/skosovsky/evaly"
)

const TieRevision = "candidate-revision-lexical-v1"

type Feasibility struct {
	Feasible bool   `json:"Feasible"`
	Reason   string `json:"Reason"`
}
type EvaluationSummary struct {
	Verdict     evaly.GateVerdict `json:"Verdict"`
	State       string            `json:"State"`
	Reason      string            `json:"Reason"`
	Quality     *float64          `json:"Quality"`
	Measurement evaly.Aggregate   `json:"Measurement"`
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
	ID             string              `json:"ID"`
	Revision       string              `json:"Revision"`
	ParentRevision string              `json:"ParentRevision"`
	Algorithm      string              `json:"Algorithm"`
	Codec          evaly.CodecIdentity `json:"Codec"`
}
type Feedback struct {
	Candidate   CandidateLineage `json:"Candidate"`
	Round       int              `json:"Round"`
	State       string           `json:"State"`
	Reason      string           `json:"Reason"`
	Quality     *float64         `json:"Quality"`
	Measurement evaly.Aggregate  `json:"Measurement"`
	References  []string         `json:"References"`
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

// ValidateMaximum rejects an oversized static batch before search budget dispatch.
func (p StaticProposer[T, I, R]) ValidateMaximum(maximum int) error {
	if maximum <= 0 || len(p.candidates) > maximum {
		return evaly.ErrInvalid
	}
	return p.Validate()
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
	if p.Validate() != nil || r.Maximum <= 0 {
		return ProposalResult[T]{}, evaly.ErrInvalid
	}
	if r.Round == 0 && len(p.candidates) > r.Maximum {
		return ProposalResult[T]{}, evaly.ErrInvalid
	}
	if r.Round != 0 {
		return ProposalResult[T]{
			Candidates: []Proposal[T]{},
			Usage:      evaly.Usage{Known: true, Units: 0},
			Exhausted:  false,
		}, nil
	}
	out := ProposalResult[T]{Exhausted: true, Candidates: []Proposal[T]{}, Usage: evaly.Usage{Known: true, Units: 0}}
	for _, candidate := range p.candidates {
		value, err := candidate.Value()
		if err != nil {
			return out, err
		}
		rec := candidate.Record()
		out.Candidates = append(
			out.Candidates,
			Proposal[T]{ID: rec.ID, ParentRevision: rec.ParentRevision, Value: value},
		)
	}
	return out, nil
}

type Round struct {
	ReceivedCount     int                `json:"ReceivedCount"`
	ReceivedTruncated bool               `json:"ReceivedTruncated"`
	Index             int                `json:"Index"`
	DispatchID        string             `json:"DispatchID"`
	ProposalRevision  string             `json:"ProposalRevision"`
	State             string             `json:"State"`
	Reason            string             `json:"Reason"`
	Usage             evaly.Usage        `json:"Usage"`
	Candidates        []string           `json:"Candidates"`
	Received          []CandidateLineage `json:"Received"`
}
