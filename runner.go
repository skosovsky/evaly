package evaly

import (
	"context"
	"errors"
	"sync"
	"time"
)

type Isolation string

const (
	Isolated     Isolation = "isolated"
	SerialShared Isolation = "serial/shared"
)

type LifecycleIdentity struct {
	Fixture   string    `json:"fixture"`
	Reset     string    `json:"reset"`
	Isolation Isolation `json:"isolation"`
}
type Lifecycle[E any] interface {
	Identity() LifecycleIdentity
	Prepare(context.Context, string) (E, error)
	Reset(context.Context, E) error
	Cleanup(context.Context, E) error
}

// LifecycleFuncs is a runnable reference adapter for host-owned fixtures.
type LifecycleFuncs[E any] struct {
	IdentityValue LifecycleIdentity                        `json:"IdentityValue"`
	PrepareFunc   func(context.Context, string) (E, error) `json:"PrepareFunc"`
	ResetFunc     func(context.Context, E) error           `json:"ResetFunc"`
	CleanupFunc   func(context.Context, E) error           `json:"CleanupFunc"`
}

// Validate checks callback presence without preparing an environment.
func (l LifecycleFuncs[E]) Validate() error {
	if l.PrepareFunc == nil || l.ResetFunc == nil || l.CleanupFunc == nil {
		return ErrInvalid
	}
	return nil
}

func (l LifecycleFuncs[E]) Identity() LifecycleIdentity { return l.IdentityValue }
func (l LifecycleFuncs[E]) Prepare(ctx context.Context, id string) (E, error) {
	if l.PrepareFunc == nil {
		var e E
		return e, ErrInvalid
	}
	return l.PrepareFunc(ctx, id)
}
func (l LifecycleFuncs[E]) Reset(ctx context.Context, e E) error {
	if l.ResetFunc == nil {
		return ErrInvalid
	}
	return l.ResetFunc(ctx, e)
}
func (l LifecycleFuncs[E]) Cleanup(ctx context.Context, e E) error {
	if l.CleanupFunc == nil {
		return ErrInvalid
	}
	return l.CleanupFunc(ctx, e)
}

type TrialContext[E any] struct {
	ID           string       `json:"ID"`
	CaseRevision string       `json:"CaseRevision"`
	Repeat       int          `json:"Repeat"`
	Attempt      int          `json:"Attempt"`
	Seed         int64        `json:"Seed"`
	Environment  E            `json:"Environment"`
	Evidence     EvidenceSink `json:"Evidence"`
	Budget       Budget       `json:"Budget"`
}
type TargetResult[O any] struct {
	Output O     `json:"Output"`
	Usage  Usage `json:"Usage"`
}
type Target[I, O, E any] interface {
	Run(context.Context, I, TrialContext[E]) (TargetResult[O], error)
}
type TargetFunc[I, O, E any] func(context.Context, I, TrialContext[E]) (TargetResult[O], error)

// Validate rejects a missing callback without executing the target.
func (f TargetFunc[I, O, E]) Validate() error {
	if f == nil {
		return ErrInvalid
	}
	return nil
}

func (f TargetFunc[I, O, E]) Run(ctx context.Context, i I, t TrialContext[E]) (TargetResult[O], error) {
	return f(ctx, i, t)
}

type RunPlan struct {
	Repeats              int           `json:"repeats"`
	Concurrency          int           `json:"concurrency"`
	Timeout              time.Duration `json:"timeout"`
	CleanupTimeout       time.Duration `json:"cleanup_timeout"`
	MaxAttempts          int           `json:"max_attempts"`
	Seed                 int64         `json:"seed"`
	StopOnInfrastructure bool          `json:"stop_on_infrastructure"`
	DispatchUnits        float64       `json:"dispatch_units"`
	GraderUnits          float64       `json:"grader_units"`
	AssertionPolicy      string        `json:"assertion_policy"`
}
type Provenance struct {
	Target   string            `json:"target"`
	Model    string            `json:"model"`
	Prompt   string            `json:"prompt"`
	Tools    string            `json:"tools"`
	Policy   string            `json:"policy"`
	Provider map[string]string `json:"provider,omitempty"`
	Unknown  []string          `json:"unknown,omitempty"`
}
type TrialStatus string

const (
	Completed          TrialStatus = completedState
	TargetError        TrialStatus = "target_error"
	SetupError         TrialStatus = "setup_error"
	Cancelled          TrialStatus = cancelledState
	BudgetExhausted    TrialStatus = "budget_exhausted"
	InfrastructureStop TrialStatus = "infrastructure_stop"
)

type CleanupStatus struct {
	State  string `json:"state"`
	Reason string `json:"reason,omitempty"`
}
type TrialRecord struct {
	ID                string          `json:"id"`
	CaseID            string          `json:"case_id"`
	CaseRevision      string          `json:"case_revision"`
	Repeat            int             `json:"repeat"`
	Attempt           int             `json:"attempt"`
	Seed              int64           `json:"seed"`
	States            []string        `json:"states"`
	Status            TrialStatus     `json:"status"`
	Reason            string          `json:"reason,omitempty"`
	Cleanup           CleanupStatus   `json:"cleanup"`
	Grades            []Grade         `json:"grades,omitempty"`
	Evidence          EvidenceRecord  `json:"evidence"`
	TargetUsage       Usage           `json:"target_usage"`
	UsageError        string          `json:"usage_error,omitempty"`
	GradingState      string          `json:"grading_state"`
	GradingStopReason string          `json:"grading_stop_reason,omitempty"`
	SkippedGraders    []SkippedGrader `json:"skipped_graders,omitempty"`
}
type CaseIdentity struct {
	ID       string `json:"id"`
	Revision string `json:"revision"`
}
type ExperimentManifest struct {
	States              []string          `json:"states"`
	OutputCodec         CodecIdentity     `json:"output_codec"`
	CaptureRequirements []string          `json:"capture_requirements"`
	CaptureMaxEvents    int               `json:"capture_max_events"`
	CaptureMaxBytes     int               `json:"capture_max_bytes"`
	CriticalEvidence    bool              `json:"critical_evidence"`
	Version             int               `json:"version"`
	ID                  string            `json:"id"`
	Revision            string            `json:"revision"`
	State               string            `json:"state"`
	Mode                string            `json:"mode"`
	Dataset             string            `json:"dataset"`
	Selection           string            `json:"selection"`
	Cases               []CaseIdentity    `json:"cases"`
	Provenance          Provenance        `json:"provenance"`
	Lifecycle           LifecycleIdentity `json:"lifecycle"`
	Plan                RunPlan           `json:"plan"`
	Graders             []GraderRevision  `json:"graders"`
	CapturePolicy       string            `json:"capture_policy"`
	CaptureKinds        []string          `json:"capture_kinds"`
	Projection          string            `json:"projection"`
	Started             time.Time         `json:"started"`
	Finished            time.Time         `json:"finished"`
	PairID              string            `json:"pair_id,omitempty"`
	PairSchedule        *PairSchedule     `json:"pair_schedule,omitempty"`
	ParentRevision      string            `json:"parent_revision,omitempty"`
}
type ExperimentRecord struct {
	Manifest ExperimentManifest `json:"manifest"`
	Trials   []TrialRecord      `json:"trials"`
}

// Experiment retains private frozen records, including unsuccessful attempts.
type Experiment struct{ record ExperimentRecord }

func (e Experiment) Record() ExperimentRecord { return mustCloneJSON(e.record) }
func (e Experiment) Revision() string         { return e.record.Manifest.Revision }
func (e Experiment) ID() string               { return e.record.Manifest.ID }
func RestoreExperiment(r ExperimentRecord) (Experiment, error) {
	if r.Manifest.Version != experimentWireRevision {
		return Experiment{}, ErrUnsupported
	}
	if r.Manifest.State != sealedState && r.Manifest.State != incompleteState {
		return Experiment{}, ErrUnsealed
	}
	rev := r.Manifest.Revision
	r.Manifest.Revision = ""
	b, err := canonical(r)
	if err != nil {
		return Experiment{}, ErrCorrupt
	}
	if digest(b) != rev {
		return Experiment{}, ErrCorrupt
	}
	r.Manifest.Revision = rev
	if err = validateExperimentRecord(r); err != nil {
		return Experiment{}, err
	}
	r, err = cloneJSON(r)
	return Experiment{r}, err
}
func freezeExperiment(r ExperimentRecord) (Experiment, error) {
	r.Manifest.Revision = ""
	b, e := canonical(r)
	if e != nil {
		return Experiment{}, e
	}
	r.Manifest.Revision = digest(b)
	return RestoreExperiment(r)
}
func validatePlan(p RunPlan) error {
	if p.Repeats <= 0 || p.Repeats > 10000 || p.Concurrency <= 0 || p.Concurrency > 1024 || p.MaxAttempts <= 0 ||
		p.MaxAttempts > 100 ||
		p.Timeout <= 0 ||
		p.CleanupTimeout <= 0 ||
		!finiteNonnegative(p.DispatchUnits) ||
		!finiteNonnegative(p.GraderUnits) ||
		(p.AssertionPolicy != assertionAll && p.AssertionPolicy != assertionAny) {
		return ErrInvalid
	}
	return nil
}
func validateExperimentRecord(r ExperimentRecord) error {
	if err := validateExperimentManifest(r.Manifest); err != nil {
		return err
	}
	cases, err := experimentCases(r.Manifest)
	if err != nil {
		return err
	}
	if err = validateExperimentTrials(r, cases); err != nil {
		return err
	}
	graders, err := experimentGraders(r.Manifest)
	if err != nil {
		return err
	}
	if err = validateTrialGrading(r, graders); err != nil {
		return err
	}
	return validateExperimentAttempts(r, graders)
}

type RunConfig[I, O, R, E any] struct {
	ID          string            `json:"ID"`
	Dataset     Dataset[I, R]     `json:"Dataset"`
	Target      Target[I, O, E]   `json:"Target"`
	OutputCodec Codec[O]          `json:"OutputCodec"`
	Lifecycle   Lifecycle[E]      `json:"Lifecycle"`
	Plan        RunPlan           `json:"Plan"`
	Provenance  Provenance        `json:"Provenance"`
	Capture     CaptureConfig     `json:"Capture"`
	Graders     []Grader[I, O, R] `json:"Graders"`
	// Project must return a permitted grading view and is part of identity.
	Project            func(context.Context, Case[I, R], O, EvidenceRecord) (View[I, O, R], error) `json:"Project"`
	ProjectionRevision string                                                                      `json:"ProjectionRevision"`
	Budget             Budget                                                                      `json:"Budget"`
	CriticalEvidence   bool                                                                        `json:"CriticalEvidence"`
}

func Run[I, O, R, E any](ctx context.Context, c RunConfig[I, O, R, E]) (Experiment, error) {
	if err := ValidateRunConfig(c); err != nil {
		return Experiment{}, err
	}
	cases := c.Dataset.record.Cases
	life := c.Lifecycle.Identity()
	m := experimentManifest(c)
	count := len(cases) * c.Plan.Repeats
	results := make([][]TrialRecord, count)
	type job struct {
		index     int
		caseIndex int
		repeat    int
	}
	jobs := make(chan job)
	runctx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)
	workers := c.Plan.Concurrency
	if life.Isolation == SerialShared {
		workers = 1
	}
	if workers > count {
		workers = count
	}
	var wg sync.WaitGroup
	for range workers {
		wg.Go(func() {
			for j := range jobs {
				results[j.index] = executeSlot(runctx, c, j.caseIndex, j.repeat)
				last := results[j.index][len(results[j.index])-1]
				if c.Plan.StopOnInfrastructure && trialInfrastructureFailure(last) {
					cancel(errInfrastructureStopped)
				}
			}
		})
	}
	for n := range count {
		jobs <- job{n, n / c.Plan.Repeats, n % c.Plan.Repeats}
	}
	close(jobs)
	wg.Wait()
	r := ExperimentRecord{Manifest: m, Trials: nil}
	r.Manifest.State = sealedState
	for _, group := range results {
		r.Trials = append(r.Trials, group...)
		for _, t := range group {
			if t.Status == Cancelled || t.Status == BudgetExhausted || t.Status == InfrastructureStop {
				r.Manifest.State = incompleteState
			}
		}
	}
	r.Manifest.States = append(r.Manifest.States, r.Manifest.State)
	r.Manifest.Finished = time.Now().UTC()
	return freezeExperiment(r)
}

func runTrial[I, O, R, E any](
	ctx context.Context,
	c RunConfig[I, O, R, E],
	cs Case[I, R],
	caseIndex int,
	repeat, attempt int,
) TrialRecord {
	var zeroEvidenceRecord EvidenceRecord
	var zeroUsage Usage
	id := c.ID + "/" + cs.ID + "/" + identityPart(repeat) + "/" + identityPart(attempt)
	r := TrialRecord{
		ID:           id,
		CaseID:       cs.ID,
		CaseRevision: cs.Revision,
		Repeat:       repeat,
		Attempt:      attempt,
		Seed:         c.Plan.Seed + int64(repeat),
		States:       []string{"queued"},
		Cleanup: CleanupStatus{
			State:  cleanupNotNeeded,
			Reason: "",
		},
		Status:            "",
		Reason:            "",
		Grades:            nil,
		Evidence:          zeroEvidenceRecord,
		TargetUsage:       zeroUsage,
		UsageError:        "",
		GradingState:      "",
		GradingStopReason: "",
		SkippedGraders:    nil,
	}
	cfg := c.Capture
	cfg.RequiredKinds = append(append([]string(nil), cfg.RequiredKinds...), cs.RequiredEvidence...)
	capture, err := NewCapture(cfg)
	if err != nil {
		capture, _ = NewCapture(c.Capture)
		r.Status = SetupError
		r.Reason = "unsupported_case_evidence"
		finalizeTrialGrading(&r, c.Graders)
		r.Evidence = capture.Seal()
		r.States = append(r.States, "terminal")
		return r
	}
	finish := func() {
		finalizeTrialGrading(&r, c.Graders)
		r.Evidence = capture.Seal()
		r.States = append(r.States, "terminal")
	}
	if ctx.Err() != nil {
		r.Status = Cancelled
		r.Reason = "cancelled_before_prepare"
		if errors.Is(context.Cause(ctx), errInfrastructureStopped) {
			r.Status = InfrastructureStop
			r.Reason = "infrastructure_stop"
		}
		finish()
		return r
	}
	trialctx, cancel := context.WithTimeout(ctx, c.Plan.Timeout)
	defer cancel()
	env, prepared := prepareTrial(trialctx, c, id, &r)
	if prepared {
		reservation, authorized := authorizeTrial(trialctx, ctx, c, id, &r)
		if authorized {
			dispatchTrial(trialctx, ctx, c, cs, caseIndex, id, env, capture, reservation, &r)
		}
	}
	cleanupTrial(ctx, c, env, &r)
	finish()
	return r
}

func experimentManifest[I, O, R, E any](c RunConfig[I, O, R, E]) ExperimentManifest {
	var zeroTime time.Time
	cases := c.Dataset.record.Cases
	life := c.Lifecycle.Identity()
	m := ExperimentManifest{
		States:              []string{"planned", "running"},
		OutputCodec:         c.OutputCodec.Identity(),
		CaptureRequirements: append([]string{}, c.Capture.RequiredKinds...),
		CaptureMaxEvents:    c.Capture.MaxEvents,
		CaptureMaxBytes:     c.Capture.MaxBytes,
		CriticalEvidence:    c.CriticalEvidence,
		Version:             experimentWireRevision,
		ID:                  c.ID,
		State:               "running",
		Mode:                controlledMode,
		Dataset:             c.Dataset.Revision(),
		Selection:           c.Dataset.record.Selection,
		Provenance:          c.Provenance,
		Lifecycle:           life,
		Plan:                c.Plan,
		CapturePolicy:       c.Capture.Policy.Revision(),
		CaptureKinds:        append([]string(nil), c.Capture.KnownKinds...),
		Projection:          c.ProjectionRevision,
		Started: time.Now().
			UTC(),
		Revision:       "",
		Cases:          nil,
		Graders:        nil,
		Finished:       zeroTime,
		PairID:         "",
		PairSchedule:   nil,
		ParentRevision: "",
	}
	for _, cs := range cases {
		m.Cases = append(m.Cases, CaseIdentity{cs.ID, cs.Revision})
	}
	for _, g := range c.Graders {
		m.Graders = append(m.Graders, g.Revision())
	}
	return m
}

func validateProvenance(p Provenance) error {
	if p.Target == "" {
		return ErrInvalid
	}
	unknown := map[string]bool{}
	for _, field := range p.Unknown {
		if field == "" || unknown[field] {
			return ErrInvalid
		}
		unknown[field] = true
	}
	for field, value := range map[string]string{"model": p.Model, "prompt": p.Prompt, "tools": p.Tools, "policy": p.Policy} {
		if value == "" && !unknown[field] {
			return ErrInvalid
		}
	}
	return nil
}

func codecFailure[I, O, R, E any](c RunConfig[I, O, R, E], cs Case[I, R], repeat, attempt int) TrialRecord {
	var zeroUsage Usage
	capture, _ := NewCapture(c.Capture)
	r := TrialRecord{
		ID:                c.ID + "/" + cs.ID + "/" + identityPart(repeat) + "/" + identityPart(attempt),
		CaseID:            cs.ID,
		CaseRevision:      cs.Revision,
		Repeat:            repeat,
		Attempt:           attempt,
		Seed:              c.Plan.Seed + int64(repeat),
		States:            []string{"queued", "terminal"},
		Status:            SetupError,
		Reason:            "codec_failure",
		Cleanup:           CleanupStatus{State: cleanupNotNeeded, Reason: ""},
		Evidence:          capture.Seal(),
		Grades:            nil,
		TargetUsage:       zeroUsage,
		UsageError:        "",
		GradingState:      "",
		GradingStopReason: "",
		SkippedGraders:    nil,
	}
	finalizeTrialGrading(&r, c.Graders)
	return r
}

func trialInfrastructureFailure(r TrialRecord) bool {
	if r.Status == SetupError || r.Status == BudgetExhausted || r.Cleanup.State == failedState || r.UsageError != "" {
		return true
	}
	for _, g := range r.Grades {
		if g.Status == GraderError {
			return true
		}
	}
	return false
}

func finalizeTrialGrading[I, O, R any](r *TrialRecord, graders []Grader[I, O, R]) {
	r.GradingState = gradingComplete
	reason := r.UsageError
	if reason == "" && r.Status != Completed {
		reason = r.Reason
		if reason == "" {
			reason = string(r.Status)
		}
	}
	obtained := map[string]bool{}
	for _, g := range r.Grades {
		obtained[g.Revision.ID] = true
	}
	for _, g := range graders {
		if !obtained[g.Revision().ID] {
			if reason == "" {
				reason = "infrastructure_stop"
			}
			r.SkippedGraders = append(r.SkippedGraders, SkippedGrader{Revision: g.Revision(), Reason: reason})
		}
	}
	if len(r.SkippedGraders) > 0 || r.UsageError != "" {
		r.GradingState = gradingPartial
		r.GradingStopReason = reason
	}
}

var errInfrastructureStopped = errors.New("evaly: infrastructure stop")
