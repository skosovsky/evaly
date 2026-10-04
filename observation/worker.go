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
	Rule, Reason, Population, Window string
	Probability                      *float64
	OutcomeDelay                     time.Duration
}
type Record struct {
	Version                    int
	ID, Revision, Parent, View string
	Sampling                   Sampling
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
	r := Record{Version: 1, ID: id, Parent: parent, View: v.Revision(), Sampling: s}
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

type Config[I, O, R any] struct {
	Capacity, Concurrency int
	Deadline              time.Duration
	Clock                 Clock
	Graders               []evaly.Grader[I, O, R]
	Budget                evaly.Budget
	GraderUnits           float64
}
type Result struct {
	State       string
	Observation Record
	Assessment  evaly.Assessment
	Reason      string
}
type Future struct{ Result <-chan Result }
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
	if c.Capacity <= 0 || c.Capacity > 100000 || c.Concurrency <= 0 || c.Concurrency > 1024 || c.Deadline <= 0 ||
		c.Clock == nil ||
		len(c.Graders) == 0 ||
		c.GraderUnits < 0 ||
		math.IsNaN(c.GraderUnits) ||
		math.IsInf(c.GraderUnits, 0) {
		return nil, evaly.ErrInvalid
	}
	for _, g := range c.Graders {
		if g == nil {
			return nil, evaly.ErrInvalid
		}
	}
	c.Graders = append([]evaly.Grader[I, O, R](nil), c.Graders...)
	runctx, cancel := context.WithCancel(ctx)
	w := &Worker[I, O, R]{
		config:  c,
		ctx:     runctx,
		cancel:  cancel,
		queue:   make(chan task[I, O, R], c.Capacity),
		changed: make(chan struct{}),
		done:    make(chan struct{}),
	}
	nonce := make([]byte, 16)
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
						Result{State: "cancelled", Observation: t.observation.Record(), Reason: "worker_cancelled"},
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
func (w *Worker[I, O, R]) evaluate(t task[I, O, R]) Result {
	r := Result{Observation: t.observation.Record()}
	if w.ctx.Err() != nil {
		r.State = "cancelled"
		r.Reason = "worker_cancelled"
		return r
	}
	remaining := t.expires.Sub(w.config.Clock.Now())
	if remaining <= 0 {
		r.State = "expired"
		r.Reason = "observation_deadline"
		return r
	}
	ctx, cancel := context.WithCancel(w.ctx)
	watchDone := make(chan struct{})
	timer := w.config.Clock.After(remaining)
	if !w.config.Clock.Now().Before(t.expires) {
		cancel()
		r.State = "expired"
		r.Reason = "observation_deadline"
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
	grades := []evaly.Grade{}
	for _, g := range w.config.Graders {
		if !w.config.Clock.Now().Before(t.expires) || ctx.Err() != nil {
			r.State = "expired"
			r.Reason = "observation_deadline"
			return r
		}
		var reservation evaly.Reservation
		var e error
		if w.config.Budget != nil {
			reservation, e = w.config.Budget.Reserve(
				ctx,
				t.observation.record.Revision+"/"+t.attempt+"/"+g.Revision().ID,
				w.config.GraderUnits,
			)
			if e != nil {
				r.State = "budget_exhausted"
				r.Reason = "grader_reservation"
				return r
			}
		}
		if w.config.Budget != nil {
			if e = w.config.Budget.Claim(ctx, reservation); e != nil {
				r.State = "budget_exhausted"
				r.Reason = "grader_dispatch_claim"
				return r
			}
		}
		if ctx.Err() != nil || !w.config.Clock.Now().Before(t.expires) {
			r.State = "expired"
			r.Reason = "observation_deadline"
			return r
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
			r.State = "failed"
			r.Reason = "invalid_observation"
			return r
		}
		grades = append(grades, a.Grades...)
		if w.config.Budget != nil {
			reconcileCtx, done := context.WithTimeout(context.WithoutCancel(w.ctx), w.config.Deadline)
			e = w.config.Budget.Reconcile(reconcileCtx, reservation, a.Grades[0].Usage)
			done()
			if e != nil {
				r.State = "failed"
				r.Reason = "usage_reconciliation"
				return r
			}
		}
	}
	r.Assessment = evaly.Assessment{
		Version: 1,
		Source:  t.observation.record.ID,
		Parent:  t.observation.record.Parent,
		View:    t.observation.saved.Revision(),
		Mode:    "observation",
		Grades:  grades,
	}
	b, _ := json.Marshal(r.Assessment)
	canonical, e := evaly.CanonicalJSON(b)
	if e != nil {
		r.State = "failed"
		r.Reason = "assessment_encoding"
		return r
	}
	hash := sha256.Sum256(canonical)
	r.Assessment.Revision = hex.EncodeToString(hash[:])
	r.State = "graded"
	if ctx.Err() != nil || !w.config.Clock.Now().Before(t.expires) {
		r.State = "expired"
		r.Reason = "observation_deadline"
	}
	return r
}
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
