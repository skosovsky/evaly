package optimizer

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sync"
	"time"

	"github.com/skosovsky/evaly"
)

type CandidateRecord struct {
	ID          string              `json:"ID"`
	Revision    string              `json:"Revision"`
	Parent      string              `json:"Parent"`
	Algorithm   string              `json:"Algorithm"`
	Description json.RawMessage     `json:"Description"`
	Codec       evaly.CodecIdentity `json:"Codec"`
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
	r := CandidateRecord{
		ID:          id,
		Parent:      parent,
		Algorithm:   algorithm,
		Description: b,
		Codec:       c.Identity(),
		Revision:    "",
	}
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
	ID     string `json:"ID"`
	Parent string `json:"Parent"`
	Value  T      `json:"Value"`
}
type ProposalRequest[I, R any] struct {
	Training    evaly.Dataset[I, R] `json:"Training"`
	Calibration evaly.Dataset[I, R] `json:"Calibration"`
	Maximum     int                 `json:"Maximum"`
	Round       int                 `json:"Round"`
	DispatchID  string              `json:"DispatchID"`
	Feedback    []Feedback          `json:"Feedback"`
	Seed        int64               `json:"Seed"`
}
type ProposalResult[T any] struct {
	Exhausted  bool          `json:"Exhausted"`
	Candidates []Proposal[T] `json:"Candidates"`
	Usage      evaly.Usage   `json:"Usage"`
}
type Proposer[T, I, R any] interface {
	Propose(context.Context, ProposalRequest[I, R]) (ProposalResult[T], error)
	Revision() string
}
type ProposalFunc[T, I, R any] struct {
	Identity string                                                                  `json:"Identity"`
	Generate func(context.Context, ProposalRequest[I, R]) (ProposalResult[T], error) `json:"Generate"`
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
	Revision    string              `json:"Revision"`
	Training    evaly.Dataset[I, R] `json:"Training"`
	Calibration evaly.Dataset[I, R] `json:"Calibration"`
	Holdout     evaly.Dataset[I, R] `json:"Holdout"`
}
type EvaluationRequest[T, I, R any] struct {
	ExperimentID string              `json:"ExperimentID"`
	SearchID     string              `json:"SearchID"`
	Phase        string              `json:"Phase"`
	DispatchID   string              `json:"DispatchID"`
	Round        int                 `json:"Round"`
	Candidate    Candidate[T]        `json:"Candidate"`
	Dataset      evaly.Dataset[I, R] `json:"Dataset"`
	Budget       evaly.Budget        `json:"Budget"`
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
	ID                  string               `json:"ID"`
	Algorithm           string               `json:"Algorithm"`
	StopRevision        string               `json:"StopRevision"`
	Seed                int64                `json:"Seed"`
	MaximumCandidates   int                  `json:"MaximumCandidates"`
	Timeout             time.Duration        `json:"Timeout"`
	Split               Split[I, R]          `json:"Split"`
	MaximumRounds       int                  `json:"MaximumRounds"`
	EvaluationUnits     float64              `json:"EvaluationUnits"`
	Constraints         Constraints[T]       `json:"Constraints"`
	SplitValidator      SplitValidator[I, R] `json:"SplitValidator"`
	FeedbackProjector   FeedbackProjector    `json:"FeedbackProjector"`
	Proposal            Proposer[T, I, R]    `json:"Proposal"`
	Codec               evaly.Codec[T]       `json:"Codec"`
	Evaluate            Evaluate[T, I, R]    `json:"Evaluate"`
	Budget              evaly.Budget         `json:"Budget"`
	ProposalUnits       float64              `json:"ProposalUnits"`
	Ledger              HoldoutLedger        `json:"Ledger"`
	CalibrationBaseline evaly.Experiment     `json:"CalibrationBaseline"`
	HoldoutBaseline     evaly.Experiment     `json:"HoldoutBaseline"`
	Gate                evaly.GatePolicy     `json:"Gate"`
	Objective           evaly.Objective      `json:"Objective"`
}
type Evaluation struct {
	Round             int                     `json:"Round"`
	DispatchID        string                  `json:"DispatchID"`
	Feasible          bool                    `json:"Feasible"`
	FeasibilityReason string                  `json:"FeasibilityReason"`
	References        []string                `json:"References"`
	Candidate         CandidateRecord         `json:"Candidate"`
	State             string                  `json:"State"`
	Reason            string                  `json:"Reason"`
	Experiment        *evaly.ExperimentRecord `json:"Experiment"`
	Comparison        *evaly.Comparison       `json:"Comparison"`
	Quality           *float64                `json:"Quality"`
}
type Result struct {
	MaximumRounds           int                     `json:"MaximumRounds"`
	MaximumCandidates       int                     `json:"MaximumCandidates"`
	TimeoutNanoseconds      int64                   `json:"TimeoutNanoseconds"`
	ProposalUnits           float64                 `json:"ProposalUnits"`
	EvaluationUnits         float64                 `json:"EvaluationUnits"`
	Version                 int                     `json:"Version"`
	RoundHistory            []Round                 `json:"RoundHistory"`
	BestMeasured            string                  `json:"BestMeasured"`
	ConstraintsRevision     string                  `json:"ConstraintsRevision"`
	SplitValidationRevision string                  `json:"SplitValidationRevision"`
	FeedbackRevision        string                  `json:"FeedbackRevision"`
	TieRevision             string                  `json:"TieRevision"`
	IndependenceValidated   bool                    `json:"IndependenceValidated"`
	Objective               evaly.ObjectiveIdentity `json:"Objective"`
	ID                      string                  `json:"ID"`
	Revision                string                  `json:"Revision"`
	Algorithm               string                  `json:"Algorithm"`
	Split                   string                  `json:"Split"`
	StopRevision            string                  `json:"StopRevision"`
	State                   string                  `json:"State"`
	Reason                  string                  `json:"Reason"`
	Seed                    int64                   `json:"Seed"`
	History                 []Evaluation            `json:"History"`
	Ranking                 []string                `json:"Ranking"`
	Winner                  string                  `json:"Winner"`
	Holdout                 *evaly.ExperimentRecord `json:"Holdout"`
	HoldoutComparison       *evaly.Comparison       `json:"HoldoutComparison"`
	Contaminated            bool                    `json:"Contaminated"`
	ProposalRevision        string                  `json:"ProposalRevision"`
	ProposalUsage           evaly.Usage             `json:"ProposalUsage"`
	States                  []string                `json:"States"`
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
	var s searchExecution[T, I, R]
	s.config = c
	s.ctx = ctx
	s.result = initialResult(c)
	if err := s.preflight(); err != nil {
		return s.result, err
	}
	executionCtx, cancel := context.WithTimeout(ctx, c.Timeout)
	defer cancel()
	s.ctx = executionCtx
	if err := s.preflightMeasurement(); err != nil {
		return s.result, err
	}
	s.codecIdentity = c.Codec.Identity()
	s.seen = map[string]bool{}
	s.candidates = map[string]Candidate[T]{}
	s.feedback = []Feedback{}
	s.runRounds()
	s.selectWinner()
	s.evaluateHoldout()

	if s.result.State != stoppedState {
		s.result.State = completedState
		if s.result.Winner == "" {
			s.result.Reason = "no_selectable_candidate"
		}
	}
	s.result.States = append(s.result.States, s.result.State)
	if e := sealResult(&s.result); e != nil {
		return s.result, e
	}
	return s.result, nil
}

// All identity parts participate in the digest; generated artifact IDs stay
// within the core identifier limit even when the search ID uses all 128 bytes.
func evaluationExperimentID(search, phase, revision string) string {
	digest := sha256.Sum256([]byte(search + "\x00" + phase + "\x00" + revision))
	return "evaly-" + phase + "-" + hex.EncodeToString(digest[:])
}

func sealResult(r *Result) error {
	cloned := *r
	cloned.Revision = ""
	b, e := json.Marshal(cloned)
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
