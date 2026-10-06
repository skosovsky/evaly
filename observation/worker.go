package observation

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/skosovsky/evaly"
)

type Sampling struct {
	Rule         string        `json:"Rule"`
	Reason       string        `json:"Reason"`
	Population   string        `json:"Population"`
	Window       string        `json:"Window"`
	Probability  *float64      `json:"Probability"`
	OutcomeDelay time.Duration `json:"OutcomeDelay"`
}
type Record struct {
	Version  int      `json:"Version"`
	ID       string   `json:"ID"`
	Revision string   `json:"Revision"`
	Parent   string   `json:"Parent"`
	View     string   `json:"View"`
	Sampling Sampling `json:"Sampling"`
}
type Observation[I, O, R any] struct {
	record Record
	saved  evaly.SavedView[I, O, R]
}

func New[I, O, R any](id, parent string, s Sampling, v evaly.SavedView[I, O, R]) (Observation[I, O, R], error) {
	if id == "" || v.Revision() == "" || s.Rule == "" || s.Population == "" || s.Window == "" || s.OutcomeDelay < 0 {
		return Observation[I, O, R]{}, evaly.ErrInvalid
	}
	if s.Probability != nil && (*s.Probability < 0 || *s.Probability > 1) {
		return Observation[I, O, R]{}, evaly.ErrInvalid
	}
	if s.Probability != nil {
		p := *s.Probability
		s.Probability = &p
	}
	r := Record{Version: 1, ID: id, Parent: parent, View: v.Revision(), Sampling: s, Revision: ""}
	b, e := json.Marshal(r)
	if e != nil {
		return Observation[I, O, R]{}, e
	}
	canonical, e := evaly.CanonicalJSON(b)
	if e != nil {
		return Observation[I, O, R]{}, e
	}
	h := sha256.Sum256(canonical)
	r.Revision = hex.EncodeToString(h[:])
	return Observation[I, O, R]{record: r, saved: v}, nil
}
func (o Observation[I, O, R]) Record() Record {
	r := o.record
	if r.Sampling.Probability != nil {
		p := *r.Sampling.Probability
		r.Sampling.Probability = &p
	}
	return r
}

type Clock interface {
	Now() time.Time
	After(time.Duration) <-chan time.Time
}
type RealClock struct{}

func (RealClock) Now() time.Time                         { return time.Now() }
func (RealClock) After(d time.Duration) <-chan time.Time { return time.After(d) }

// Config is executable host configuration, not a portable artifact. Graders,
// Clock and Budget remain shared implementations: use concurrency-safe ports or
// Concurrency=1. Deadline also bounds detached usage reconciliation.
type Config[I, O, R any] struct {
	Capacity    int
	Concurrency int
	Deadline    time.Duration
	Clock       Clock
	Graders     []evaly.Grader[I, O, R]
	Budget      evaly.Budget
	GraderUnits float64
}
type Result struct {
	Version     int              `json:"version"`
	State       State            `json:"State"`
	Observation Record           `json:"Observation"`
	Assessment  evaly.Assessment `json:"Assessment"`
	Reason      Reason           `json:"Reason"`
}
type Future struct {
	Result <-chan Result `json:"Result"`
}
type task[I, O, R any] struct {
	attempt     string
	observation Observation[I, O, R]
	result      chan Result
	expires     time.Time
}
type Worker[I, O, R any] struct {
	identity string
	next     uint64
	mu       sync.Mutex
	config   Config[I, O, R]
	ctx      context.Context
	cancel   context.CancelFunc
	queue    chan task[I, O, R]
	pending  int
	closed   bool
	changed  chan struct{}
	done     chan struct{}
}

func Start[I, O, R any](ctx context.Context, c Config[I, O, R]) (*Worker[I, O, R], error) {
	if err := evaly.ValidatePort(c.Clock); err != nil {
		return nil, err
	}
	if c.Budget != nil {
		if err := evaly.ValidatePort(c.Budget); err != nil {
			return nil, err
		}
	}
	if c.Capacity <= 0 || c.Capacity > 100000 || c.Concurrency <= 0 || c.Concurrency > 1024 || c.Deadline <= 0 ||
		c.Clock == nil ||
		len(c.Graders) == 0 ||
		c.GraderUnits < 0 ||
		math.IsNaN(c.GraderUnits) ||
		math.IsInf(c.GraderUnits, 0) {
		return nil, evaly.ErrInvalid
	}
	revisions := make([]evaly.GraderRevision, len(c.Graders))
	for i, g := range c.Graders {
		if e := evaly.ValidatePort(g); e != nil {
			return nil, e
		}
		revisions[i] = g.Revision()
	}
	if e := evaly.ValidateGraderRevisions(revisions); e != nil {
		return nil, e
	}
	c.Graders = append([]evaly.Grader[I, O, R](nil), c.Graders...)
	runctx, cancel := context.WithCancel(ctx)
	w := &Worker[I, O, R]{
		config:  c,
		ctx:     runctx,
		cancel:  cancel,
		queue:   make(chan task[I, O, R], c.Capacity),
		changed: make(chan struct{}),
		done:    make(chan struct{}), identity: "", next: 0, mu: sync.Mutex{}, pending: 0, closed: false,
	}
	nonce := make([]byte, admissionNonceBytes)
	if _, e := rand.Read(nonce); e != nil {
		cancel()
		return nil, e
	}
	w.identity = hex.EncodeToString(nonce)
	var wg sync.WaitGroup
	for range c.Concurrency {
		wg.Go(func() { ; w.loop() })
	}
	go func() { wg.Wait(); close(w.done) }()
	return w, nil
}

// Enqueue never waits for queue capacity or grader work.
func (w *Worker[I, O, R]) Enqueue(o Observation[I, O, R]) (Future, string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed || w.ctx.Err() != nil {
		return Future{}, "rejected_closed"
	}
	if o.record.Revision == "" {
		return Future{}, "rejected_invalid"
	}
	r := make(chan Result, 1)
	w.next++
	t := task[I, O, R]{
		attempt:     fmt.Sprintf("%s/%d", w.identity, w.next),
		observation: o,
		result:      r,
		expires:     w.config.Clock.Now().Add(w.config.Deadline),
	}
	select {
	case w.queue <- t:
		w.pending++
		return Future{r}, "accepted"
	default:
		return Future{}, "rejected_overload"
	}
}
func (w *Worker[I, O, R]) signal() { close(w.changed); w.changed = make(chan struct{}) }
func (w *Worker[I, O, R]) finish(t task[I, O, R], r Result) {
	t.result <- r
	close(t.result)
	w.mu.Lock()
	w.pending--
	w.signal()
	w.mu.Unlock()
}
func (w *Worker[I, O, R]) loop() {
	for {
		select {
		case <-w.ctx.Done():
			// Admission checks the same context; drain every accepted queued observation.
			w.mu.Lock()
			w.closed = true
			w.mu.Unlock()
			for {
				select {
				case t := <-w.queue:
					w.finish(
						t,
						w.cancelledResult(t),
					)
				default:
					return
				}
			}
		case t := <-w.queue:
			w.finish(t, w.evaluate(t))
		}
	}
}
func (w *Worker[I, O, R]) cancelledResult(t task[I, O, R]) Result {
	var zeroAssessment evaly.Assessment
	return w.sealResult(
		t,
		Result{
			Version:     resultWireRevision,
			State:       Cancelled,
			Observation: t.observation.Record(),
			Reason:      WorkerCancelled,
			Assessment:  zeroAssessment,
		},
	)
}
func (w *Worker[I, O, R]) sealResult(t task[I, O, R], r Result) Result {
	a := evaly.Assessment{
		Version: resultWireRevision,
		Source:  t.observation.record.ID,
		Parent:  t.observation.record.Parent,
		View:    t.observation.saved.Revision(),
		Mode:    "observation",
		Grades:  r.Assessment.Grades,
		State:   "complete", Revision: "", Planned: nil, StopReason: "", Skipped: nil,
	}
	seen := map[string]bool{}
	for _, g := range a.Grades {
		seen[g.Revision.ID] = true
	}
	for _, g := range w.config.Graders {
		rev := g.Revision()
		a.Planned = append(a.Planned, rev)
		if !seen[rev.ID] {
			a.Skipped = append(a.Skipped, evaly.SkippedGrader{Revision: rev, Reason: string(r.Reason)})
		}
	}
	if r.State != Graded {
		a.State = "partial"
		a.StopReason = string(r.Reason)
	}
	sealed, e := evaly.SealAssessment(a)
	if e != nil {
		r.State = Failed
		r.Reason = AssessmentEncoding
		return r
	}
	r.Assessment = sealed
	return r
}
func (w *Worker[I, O, R]) evaluate(t task[I, O, R]) Result {
	return w.sealResult(t, w.evaluateUnsealed(t))
}

func (w *Worker[I, O, R]) evaluateUnsealed(t task[I, O, R]) Result {
	var zeroAssessment evaly.Assessment
	r := Result{
		Version:     resultWireRevision,
		Observation: t.observation.Record(),
		State:       "",
		Assessment:  zeroAssessment,
		Reason:      "",
	}
	if w.ctx.Err() != nil {
		r.State = Cancelled
		r.Reason = WorkerCancelled
		return r
	}
	remaining := t.expires.Sub(w.config.Clock.Now())
	if remaining <= 0 {
		r.State = Expired
		r.Reason = ObservationDeadline
		return r
	}
	ctx, cancel := context.WithCancel(w.ctx)
	watchDone := make(chan struct{})
	timer := w.config.Clock.After(remaining)
	if !w.config.Clock.Now().Before(t.expires) {
		cancel()
		r.State = Expired
		r.Reason = ObservationDeadline
		return r
	}
	go func() {
		select {
		case <-timer:
			cancel()
		case <-watchDone:
		case <-ctx.Done():
		}
	}()
	defer close(watchDone)
	defer cancel()
	for _, g := range w.config.Graders {
		if !w.evaluateGrade(ctx, t, g, &r) {
			return r
		}
	}
	r.State = Graded
	if ctx.Err() != nil || !w.config.Clock.Now().Before(t.expires) {
		r.State = Expired
		r.Reason = ObservationDeadline
		if w.ctx.Err() != nil {
			r.State = Cancelled
			r.Reason = WorkerCancelled
		}
	}
	return r
}

// Flush waits for pending work without closing admission. Stop producers before
// calling Flush for a final drain; cancellation stops waiting, not grading.
func (w *Worker[I, O, R]) Flush(ctx context.Context) error {
	for {
		w.mu.Lock()
		pending := w.pending
		changed := w.changed
		w.mu.Unlock()
		if pending == 0 {
			return nil
		}
		select {
		case <-changed:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
func (w *Worker[I, O, R]) Cancel(ctx context.Context) error {
	w.mu.Lock()
	w.closed = true
	w.mu.Unlock()
	w.cancel()
	select {
	case <-w.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (w *Worker[I, O, R]) evaluateGrade(ctx context.Context, t task[I, O, R], g evaly.Grader[I, O, R], r *Result) bool {
	if !w.config.Clock.Now().Before(t.expires) || ctx.Err() != nil {
		r.State = Expired
		r.Reason = ObservationDeadline
		if w.ctx.Err() != nil {
			r.State = Cancelled
			r.Reason = WorkerCancelled
		}
		return false
	}
	reservation, authorized := w.authorizeGrade(ctx, t, g, r)
	if !authorized {
		return false
	}
	if ctx.Err() != nil || !w.config.Clock.Now().Before(t.expires) {
		r.State = Expired
		r.Reason = ObservationDeadline
		if w.ctx.Err() != nil {
			r.State = Cancelled
			r.Reason = WorkerCancelled
		}
		return false
	}
	a, e := evaly.Rescore(
		ctx,
		t.observation.saved,
		[]evaly.Grader[I, O, R]{g},
		t.observation.record.ID,
		t.observation.record.Parent,
		"observation",
	)
	if e != nil {
		r.State = Failed
		r.Reason = InvalidObservation
		return false
	}
	r.Assessment.Grades = append(r.Assessment.Grades, a.Grades...)
	if len(a.Grades) == 0 {
		return w.missingGrade(a, reservation, r)
	}

	if w.config.Budget != nil {
		reconcileCtx, done := context.WithTimeout(context.WithoutCancel(w.ctx), w.config.Deadline)
		e = w.config.Budget.Reconcile(reconcileCtx, reservation, a.Grades[0].Usage)
		done()
		if e != nil {
			r.State = Failed
			r.Reason = UsageReconciliation
			return false
		}
	}
	return true
}

func (w *Worker[I, O, R]) authorizeGrade(
	ctx context.Context,
	t task[I, O, R],
	g evaly.Grader[I, O, R],
	r *Result,
) (evaly.Reservation, bool) {
	var reservation evaly.Reservation
	var e error
	if w.config.Budget != nil {
		reservation, e = w.config.Budget.Reserve(
			ctx,
			t.observation.record.Revision+"/"+t.attempt+"/"+g.Revision().ID,
			w.config.GraderUnits,
		)
		if e != nil {
			r.State = BudgetExhausted
			r.Reason = GraderReservation
			return reservation, false
		}
	}
	if w.config.Budget != nil {
		if e = w.config.Budget.Claim(ctx, reservation); e != nil {
			r.State = BudgetExhausted
			r.Reason = GraderDispatchClaim
			return reservation, false
		}
	}
	return reservation, true
}
func (w *Worker[I, O, R]) missingGrade(a evaly.Assessment, reservation evaly.Reservation, r *Result) bool {
	r.State = Failed
	r.Reason = GradingProjection
	if a.StopReason == "context_cancelled" {
		r.State = Expired
		r.Reason = ObservationDeadline
	}
	if w.ctx.Err() != nil {
		r.State = Cancelled
		r.Reason = WorkerCancelled
	}
	// A claimed reservation retains unknown liability; never release a claim.
	if w.config.Budget != nil {
		reconcileCtx, done := context.WithTimeout(context.WithoutCancel(w.ctx), w.config.Deadline)
		reconcileErr := w.config.Budget.Reconcile(reconcileCtx, reservation, evaly.Usage{Known: false, Units: 0})
		done()
		if reconcileErr != nil {
			return false
		}
	}
	return false
}
