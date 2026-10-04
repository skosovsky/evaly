package evaly

import (
	"context"
	"math"
)

type GradeStatus string

const (
	Scored               GradeStatus = "scored"
	NotApplicable        GradeStatus = "not_applicable"
	InsufficientEvidence GradeStatus = "insufficient_evidence"
	GraderError          GradeStatus = "grader_error"
)

type GraderRevision struct {
	ID             string `json:"id"`
	Implementation string `json:"implementation"`
	Rubric         string `json:"rubric"`
	Model          string `json:"model"`
	Prompt         string `json:"prompt"`
	Configuration  string `json:"configuration"`
}
type Metric struct {
	Name          string  `json:"name"`
	Unit          string  `json:"unit"`
	ScaleRevision string  `json:"scale_revision"`
	Value         float64 `json:"value"`
	Minimum       float64 `json:"minimum"`
	Maximum       float64 `json:"maximum"`
	Direction     string  `json:"direction"` // higher or lower
}
type Assertion struct {
	Name   string `json:"name"`
	Pass   bool   `json:"pass"`
	Reason string `json:"reason"`
}
type Grade struct {
	Dispatched   bool           `json:"dispatched"`
	Revision     GraderRevision `json:"revision"`
	Status       GradeStatus    `json:"status"`
	Metrics      []Metric       `json:"metrics,omitempty"`
	Assertions   []Assertion    `json:"assertions,omitempty"`
	Reasons      []string       `json:"reasons,omitempty"`
	EvidenceRefs []string       `json:"evidence_refs,omitempty"`
	Usage        Usage          `json:"usage"`
}

// View is already projected by the host; graders must treat its content as data.
type View[I, O, R any] struct {
	Case     Case[I, R]
	Output   O
	Evidence EvidenceRecord
}
type Grader[I, O, R any] interface {
	Revision() GraderRevision
	Grade(context.Context, View[I, O, R]) (Grade, error)
}
type GraderFunc[I, O, R any] struct {
	Identity GraderRevision
	Evaluate func(context.Context, View[I, O, R]) (Grade, error)
}

func (g GraderFunc[I, O, R]) Validate() error {
	if g.Evaluate == nil {
		return ErrInvalid
	}
	return ValidateGraderRevisions([]GraderRevision{g.Identity})
}
func (g GraderFunc[I, O, R]) Revision() GraderRevision { return g.Identity }
func (g GraderFunc[I, O, R]) Grade(ctx context.Context, v View[I, O, R]) (Grade, error) {
	if g.Evaluate == nil {
		return Grade{}, ErrInvalid
	}
	return g.Evaluate(ctx, v)
}
func ValidateGrade(g Grade) error {
	if g.Revision.ID == "" || g.Revision.Implementation == "" || g.Revision.Rubric == "" {
		return ErrInvalid
	}
	if !finiteNonnegative(g.Usage.Units) || len(g.Reasons) > 32 || len(g.EvidenceRefs) > 32 {
		return ErrInvalid
	}
	for _, ref := range g.EvidenceRefs {
		if !safeReference(ref) {
			return ErrInvalid
		}
	}
	switch g.Status {
	case Scored:
		if len(g.Metrics) == 0 && len(g.Assertions) == 0 {
			return ErrInvalid
		}
	case NotApplicable, InsufficientEvidence, GraderError:
		if len(g.Metrics) > 0 || len(g.Assertions) > 0 {
			return ErrInvalid
		}
	default:
		return ErrUnsupported
	}
	names := map[string]bool{}
	for _, m := range g.Metrics {
		if m.Name == "" || m.Unit == "" || m.ScaleRevision == "" || names[m.Name] || math.IsNaN(m.Value) ||
			math.IsInf(m.Value, 0) ||
			math.IsNaN(m.Minimum) ||
			math.IsInf(m.Minimum, 0) ||
			math.IsNaN(m.Maximum) ||
			math.IsInf(m.Maximum, 0) ||
			m.Minimum >= m.Maximum ||
			m.Value < m.Minimum ||
			m.Value > m.Maximum ||
			(m.Direction != "higher" && m.Direction != "lower") {
			return ErrInvalid
		}
		names[m.Name] = true
	}
	names = map[string]bool{}
	for _, a := range g.Assertions {
		if a.Name == "" || names[a.Name] {
			return ErrInvalid
		}
		names[a.Name] = true
	}
	return nil
}

// Assess preserves errors as distinct grading results. It never calls a target.
// The factory must return a fresh, permitted snapshot for each grader; evaly
// supplies no target capability and cannot sandbox arbitrary caller closures.
func Assess[I, O, R any](
	ctx context.Context,
	graders []Grader[I, O, R],
	factory func() (View[I, O, R], error),
) []Grade {
	out := make([]Grade, 0, len(graders))
	revisions := make([]GraderRevision, len(graders))
	valid := factory != nil
	for i, g := range graders {
		if ValidatePort(g) != nil {
			valid = false
			revisions[i] = GraderRevision{ID: "invalid", Implementation: "invalid", Rubric: "invalid"}
		} else {
			revisions[i] = g.Revision()
		}
	}
	if ValidateGraderRevisions(revisions) != nil {
		valid = false
	}
	if !valid {
		for _, rev := range revisions {
			out = append(out, Grade{Revision: rev, Status: GraderError, Reasons: []string{"invalid_grading_plan"}})
		}
		return out
	}
	for _, g := range graders {
		rev := g.Revision()
		var result Grade
		dispatched := false
		var e error
		if ctx.Err() != nil {
			e = ctx.Err()
		} else {
			var view View[I, O, R]
			view, e = factory()
			if e == nil {
				e = ctx.Err()
			}
			if e == nil {
				dispatched = true
				result, e = g.Grade(ctx, view)
			}
		}
		if e == nil {
			e = ctx.Err()
		}
		result.Revision = rev
		result.Dispatched = dispatched
		if e == nil {
			e = ValidateGrade(result)
		}
		if e != nil {
			result = Grade{
				Dispatched: dispatched,
				Revision:   rev,
				Status:     GraderError,
				Reasons:    []string{"grader_failure"},
				Usage:      result.Usage,
			}
			if !finiteNonnegative(result.Usage.Units) {
				result.Usage = Usage{}
			}
		}
		result, _ = cloneJSON(result)
		out = append(out, result)
	}
	return out
}

// AssertionOutcome applies an explicit conflict policy; errors/missing evidence
// from any required grader prevent success even with an "any" assertion policy.
func AssertionOutcome(grades []Grade, policy string) (bool, bool) {
	if len(grades) == 0 || (policy != "all" && policy != "any") {
		return false, false
	}
	count, passed := 0, 0
	for _, g := range grades {
		if g.Status != Scored {
			return false, false
		}
		for _, a := range g.Assertions {
			count++
			if a.Pass {
				passed++
			}
		}
	}
	if count == 0 {
		return false, false
	}
	if policy == "all" {
		return passed == count, true
	}
	return passed > 0, true
}

// AbsenceGrader requires complete coverage of the relevant event kind.
func AbsenceGrader[I, O, R any](rev GraderRevision, kind string, forbidden func(Event) bool) GraderFunc[I, O, R] {
	return GraderFunc[I, O, R]{Identity: rev, Evaluate: func(ctx context.Context, v View[I, O, R]) (Grade, error) {
		if e := ctx.Err(); e != nil {
			return Grade{}, e
		}
		if !CompleteFor(v.Evidence, kind) {
			return Grade{Status: InsufficientEvidence, Reasons: []string{"capture_incomplete"}}, nil
		}
		pass := true
		for _, event := range v.Evidence.Events {
			if event.Kind == kind && forbidden(event) {
				pass = false
			}
		}
		return Grade{
			Status:     Scored,
			Assertions: []Assertion{{Name: "absence:" + kind, Pass: pass, Reason: "trajectory"}},
		}, nil
	}}
}

// JudgeRequest separates trusted rubric from untrusted projected content.
type JudgeRequest[I, O, R any] struct {
	Instructions string
	Data         View[I, O, R]
}
type Judge[I, O, R any] interface {
	Judge(context.Context, JudgeRequest[I, O, R]) (Grade, error)
}
type LLMGrader[I, O, R any] struct {
	Identity     GraderRevision
	Instructions string
	Port         Judge[I, O, R]
}

func (g LLMGrader[I, O, R]) Validate() error {
	if g.Instructions == "" {
		return ErrInvalid
	}
	if e := ValidateGraderRevisions([]GraderRevision{g.Identity}); e != nil {
		return e
	}
	return ValidatePort(g.Port)
}
func (g LLMGrader[I, O, R]) Revision() GraderRevision { return g.Identity }
func (g LLMGrader[I, O, R]) Grade(ctx context.Context, v View[I, O, R]) (Grade, error) {
	if g.Port == nil || g.Instructions == "" {
		return Grade{}, ErrInvalid
	}
	return g.Port.Judge(ctx, JudgeRequest[I, O, R]{g.Instructions, v})
}

// ScriptedJudge is a deterministic reference adapter, not proof of real judge accuracy.
type ScriptedJudge[I, O, R any] struct {
	Evaluate func(context.Context, JudgeRequest[I, O, R]) (Grade, error)
}

func (s ScriptedJudge[I, O, R]) Validate() error {
	if s.Evaluate == nil {
		return ErrInvalid
	}
	return nil
}
func (s ScriptedJudge[I, O, R]) Judge(ctx context.Context, r JudgeRequest[I, O, R]) (Grade, error) {
	if s.Evaluate == nil {
		return Grade{}, ErrInvalid
	}
	return s.Evaluate(ctx, r)
}

type PairRequest[T any] struct {
	Instructions string
	A, B         T
}
type PairJudgment struct {
	Preferred string `json:"preferred"`
	Reason    string `json:"reason"`
	Usage     Usage  `json:"usage"`
}
type PairJudge[T any] interface {
	JudgePair(context.Context, PairRequest[T]) (PairJudgment, error)
}
type PairCheck struct {
	Forward, Reverse                     PairJudgment
	ForwardDispatched, ReverseDispatched bool
	Disagreement, Abstention             bool
	Errors                               []string
	Reviewed                             int
}

func CheckPair[T any](ctx context.Context, j PairJudge[T], instructions string, a, b Snapshot[T]) PairCheck {
	r := PairCheck{}
	if ValidatePort(j) != nil || instructions == "" || a.Validate() != nil || b.Validate() != nil {
		r.Errors = []string{"invalid_pair_protocol"}
		return r
	}
	run := func(a, b Snapshot[T], dispatched *bool) PairJudgment {
		if ctx.Err() != nil {
			r.Errors = append(r.Errors, "pair_cancelled_before_dispatch")
			return PairJudgment{}
		}
		av, e := a.Value()
		if e != nil {
			r.Errors = append(r.Errors, "pair_snapshot_failure")
			return PairJudgment{}
		}
		bv, e := b.Value()
		if e != nil {
			r.Errors = append(r.Errors, "pair_snapshot_failure")
			return PairJudgment{}
		}
		if ctx.Err() != nil {
			r.Errors = append(r.Errors, "pair_cancelled_before_dispatch")
			return PairJudgment{}
		}
		*dispatched = true
		v, e := j.JudgePair(ctx, PairRequest[T]{instructions, av, bv})
		if e == nil {
			e = ctx.Err()
		}
		if !finiteNonnegative(v.Usage.Units) {
			v.Usage = Usage{}
			e = ErrInvalid
		}
		if e != nil || (v.Preferred != "A" && v.Preferred != "B" && v.Preferred != "abstain") {
			r.Errors = append(r.Errors, "pair_judge_failure")
			v.Preferred = ""
			return v
		}
		r.Reviewed++
		return v
	}
	r.Forward = run(a, b, &r.ForwardDispatched)
	r.Reverse = run(b, a, &r.ReverseDispatched)
	r.Abstention = r.Forward.Preferred == "abstain" || r.Reverse.Preferred == "abstain"
	r.Disagreement = r.Reviewed == 2 && !r.Abstention && r.Forward.Preferred == r.Reverse.Preferred
	return r
}
