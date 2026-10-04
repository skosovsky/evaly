package evaly

import (
	"context"
	"strings"
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
	IdentityValue          LifecycleIdentity
	PrepareFunc            func(context.Context, string) (E, error)
	ResetFunc, CleanupFunc func(context.Context, E) error
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
	ID, CaseRevision string
	Repeat, Attempt  int
	Seed             int64
	Environment      E
	Evidence         EvidenceSink
	Budget           Budget
}
type TargetResult[O any] struct {
	Output      O
	Usage       Usage
	OutcomeRefs []string
}
type Target[I, O, E any] interface {
	Run(context.Context, I, TrialContext[E]) (TargetResult[O], error)
}
type TargetFunc[I, O, E any] func(context.Context, I, TrialContext[E]) (TargetResult[O], error)

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
	Completed       TrialStatus = "completed"
	TargetError     TrialStatus = "target_error"
	SetupError      TrialStatus = "setup_error"
	Cancelled       TrialStatus = "cancelled"
	BudgetExhausted TrialStatus = "budget_exhausted"
)

type CleanupStatus struct {
	State  string `json:"state"`
	Reason string `json:"reason,omitempty"`
}
type TrialRecord struct {
	ID           string         `json:"id"`
	CaseID       string         `json:"case_id"`
	CaseRevision string         `json:"case_revision"`
	Repeat       int            `json:"repeat"`
	Attempt      int            `json:"attempt"`
	Seed         int64          `json:"seed"`
	States       []string       `json:"states"`
	Status       TrialStatus    `json:"status"`
	Reason       string         `json:"reason,omitempty"`
	Cleanup      CleanupStatus  `json:"cleanup"`
	Grades       []Grade        `json:"grades,omitempty"`
	Evidence     EvidenceRecord `json:"evidence"`
	TargetUsage  Usage          `json:"target_usage"`
	UsageError   string         `json:"usage_error,omitempty"`
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
	PairOrder           []string          `json:"pair_order,omitempty"`
	ParentRevision      string            `json:"parent_revision,omitempty"`
}
type ExperimentRecord struct {
	Manifest ExperimentManifest `json:"manifest"`
	Trials   []TrialRecord      `json:"trials"`
}

// Experiment retains private frozen records, including unsuccessful attempts.
type Experiment struct{ record ExperimentRecord }

func (e Experiment) Record() ExperimentRecord { r, _ := cloneJSON(e.record); return r }
func (e Experiment) Revision() string         { return e.record.Manifest.Revision }
func (e Experiment) ID() string               { return e.record.Manifest.ID }
func RestoreExperiment(r ExperimentRecord) (Experiment, error) {
	if r.Manifest.Version != 1 {
		return Experiment{}, ErrUnsupported
	}
	if r.Manifest.State != "sealed" && r.Manifest.State != "incomplete" {
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
		(p.AssertionPolicy != "all" && p.AssertionPolicy != "any") {
		return ErrInvalid
	}
	return nil
}
func validateExperimentRecord(r ExperimentRecord) error {
	m := r.Manifest
	if !validArtifactID(m.ID) || m.Provenance.Target == "" || m.Dataset == "" || m.Mode != "controlled" ||
		m.Projection == "" ||
		m.CapturePolicy == "" ||
		len(m.Cases) == 0 {
		return ErrInvalid
	}
	if err := validatePlan(m.Plan); err != nil {
		return err
	}
	if m.Lifecycle.Fixture == "" || m.Lifecycle.Reset == "" ||
		(m.Lifecycle.Isolation != Isolated && m.Lifecycle.Isolation != SerialShared) {
		return ErrInvalid
	}
	if m.Started.IsZero() || m.Finished.Before(m.Started) || m.OutputCodec.ID == "" || m.OutputCodec.Version == "" ||
		m.CaptureMaxEvents <= 0 ||
		m.CaptureMaxBytes <= 0 ||
		strings.Join(m.States, ",") != "planned,running,"+m.State {
		return ErrInvalid
	}
	if err := validateProvenance(m.Provenance); err != nil {
		return err
	}
	cases := map[string]string{}
	for _, c := range m.Cases {
		if c.ID == "" || c.Revision == "" || cases[c.ID] != "" {
			return ErrConflict
		}
		cases[c.ID] = c.Revision
	}
	seen := map[string]bool{}
	for _, t := range r.Trials {
		if seen[t.ID] || cases[t.CaseID] != t.CaseRevision || t.Repeat < 0 || t.Repeat >= m.Plan.Repeats ||
			t.Attempt < 0 ||
			t.Attempt >= m.Plan.MaxAttempts {
			return ErrInvalid
		}
		seen[t.ID] = true
		if !finiteNonnegative(t.TargetUsage.Units) {
			return ErrInvalid
		}
		switch t.Cleanup.State {
		case "not_needed":
			if len(t.States) != 2 {
				return ErrInvalid
			}
		case "completed":
			if t.Cleanup.Reason != "" {
				return ErrInvalid
			}
		case "failed":
			if t.Cleanup.Reason == "" {
				return ErrInvalid
			}
		default:
			return ErrUnsupported
		}
		chain := strings.Join(t.States, ",")
		full := "queued,preparing,running,collecting,terminal"
		if chain != "queued,terminal" && chain != "queued,preparing,terminal" && chain != full {
			return ErrInvalid
		}
		if (t.Status == Completed || t.Status == TargetError) && chain != full {
			return ErrInvalid
		}
		if chain == full && t.Cleanup.State == "not_needed" {
			return ErrInvalid
		}
		switch t.Status {
		case Completed, TargetError, SetupError, Cancelled, BudgetExhausted:
		default:
			return ErrUnsupported
		}
		if err := ValidateEvidence(t.Evidence); err != nil {
			return err
		}
		for _, g := range t.Grades {
			if err := ValidateGrade(g); err != nil {
				return err
			}
		}
	}
	if len(m.Graders) == 0 {
		return ErrInvalid
	}
	graderIDs := map[string]GraderRevision{}
	for _, g := range m.Graders {
		if g.ID == "" || g.Implementation == "" || g.Rubric == "" {
			return ErrInvalid
		}
		if _, ok := graderIDs[g.ID]; ok {
			return ErrConflict
		}
		graderIDs[g.ID] = g
	}
	slots := map[string]int{}
	attempts := map[string]map[int]TrialRecord{}
	for _, t := range r.Trials {
		key := t.CaseID + "/" + identityPart(t.Repeat)
		slots[key]++
		if attempts[key] == nil {
			attempts[key] = map[int]TrialRecord{}
		}
		if _, exists := attempts[key][t.Attempt]; exists {
			return ErrConflict
		}
		attempts[key][t.Attempt] = t
		if t.ID != m.ID+"/"+t.CaseID+"/"+identityPart(t.Repeat)+"/"+identityPart(t.Attempt) {
			return ErrConflict
		}
		graded := map[string]bool{}
		for _, g := range t.Grades {
			if expected, ok := graderIDs[g.Revision.ID]; !ok || expected != g.Revision || graded[g.Revision.ID] {
				return ErrConflict
			}
			graded[g.Revision.ID] = true
		}
	}
	for _, attemptSet := range attempts {
		for attempt := 0; attempt < len(attemptSet); attempt++ {
			trial, ok := attemptSet[attempt]
			if !ok {
				return ErrInvalid
			}
			if attempt < len(attemptSet)-1 && trial.Status != SetupError {
				return ErrInvalid
			}
		}
	}
	if m.State == "sealed" {
		for _, cs := range m.Cases {
			for repeat := 0; repeat < m.Plan.Repeats; repeat++ {
				if slots[cs.ID+"/"+identityPart(repeat)] == 0 {
					return ErrIncomplete
				}
			}
		}
		for _, t := range r.Trials {
			if t.Status == Cancelled || t.Status == BudgetExhausted {
				return ErrIncomplete
			}
		}
	}
	return nil
}

type RunConfig[I, O, R, E any] struct {
	ID          string
	Dataset     Dataset[I, R]
	Target      Target[I, O, E]
	OutputCodec Codec[O]
	Lifecycle   Lifecycle[E]
	Plan        RunPlan
	Provenance  Provenance
	Capture     CaptureConfig
	Graders     []Grader[I, O, R]
	// Project must return a permitted grading view and is part of identity.
	Project            func(context.Context, Case[I, R], O, EvidenceRecord) (View[I, O, R], error)
	ProjectionRevision string
	Budget             Budget
	PairID             string
	PairOrder          []string
	CriticalEvidence   bool
}

func Run[I, O, R, E any](ctx context.Context, c RunConfig[I, O, R, E]) (Experiment, error) {
	if e := validatePlan(c.Plan); e != nil {
		return Experiment{}, e
	}
	if !validArtifactID(c.ID) || c.Target == nil || c.OutputCodec == nil || c.Lifecycle == nil || c.Project == nil ||
		c.ProjectionRevision == "" ||
		c.Provenance.Target == "" ||
		len(c.Graders) == 0 ||
		c.Dataset.Len()*c.Plan.Repeats > 1000000 {
		return Experiment{}, ErrInvalid
	}
	if err := validateProvenance(c.Provenance); err != nil {
		return Experiment{}, err
	}
	cases, e := c.Dataset.Cases()
	if e != nil {
		return Experiment{}, e
	}
	life := c.Lifecycle.Identity()
	if life.Fixture == "" || life.Reset == "" || (life.Isolation != Isolated && life.Isolation != SerialShared) {
		return Experiment{}, ErrInvalid
	}
	if _, e = NewCapture(c.Capture); e != nil {
		return Experiment{}, e
	}
	m := ExperimentManifest{
		States:              []string{"planned", "running"},
		OutputCodec:         c.OutputCodec.Identity(),
		CaptureRequirements: append([]string{}, c.Capture.RequiredKinds...),
		CaptureMaxEvents:    c.Capture.MaxEvents,
		CaptureMaxBytes:     c.Capture.MaxBytes,
		CriticalEvidence:    c.CriticalEvidence,
		Version:             1,
		ID:                  c.ID,
		State:               "running",
		Mode:                "controlled",
		Dataset:             c.Dataset.Revision(),
		Selection:           c.Dataset.record.Selection,
		Provenance:          c.Provenance,
		Lifecycle:           life,
		Plan:                c.Plan,
		CapturePolicy:       c.Capture.Policy.Revision(),
		CaptureKinds:        append([]string(nil), c.Capture.KnownKinds...),
		Projection:          c.ProjectionRevision,
		Started:             time.Now().UTC(),
		PairID:              c.PairID,
		PairOrder:           c.PairOrder,
	}
	for _, cs := range cases {
		m.Cases = append(m.Cases, CaseIdentity{cs.ID, cs.Revision})
	}
	ids := map[string]bool{}
	for _, g := range c.Graders {
		if g == nil {
			return Experiment{}, ErrInvalid
		}
		rev := g.Revision()
		if rev.ID == "" || rev.Implementation == "" || rev.Rubric == "" || ids[rev.ID] {
			return Experiment{}, ErrInvalid
		}
		ids[rev.ID] = true
		m.Graders = append(m.Graders, rev)
	}
	count := len(cases) * c.Plan.Repeats
	results := make([][]TrialRecord, count)
	type job struct{ index, caseIndex, repeat int }
	jobs := make(chan job)
	runctx, cancel := context.WithCancel(ctx)
	defer cancel()
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
				for attempt := 0; attempt < c.Plan.MaxAttempts; attempt++ {
					// Decode a private input for every attempt; target mutations cannot affect the dataset.
					cs, err := c.Dataset.Cases()
					if err != nil {
						originalRecord := c.Dataset.record.Cases[j.caseIndex]
						failed := codecFailure(
							c,
							Case[I, R]{ID: originalRecord.ID, Revision: originalRecord.Revision},
							j.repeat,
							attempt,
						)
						results[j.index] = append(results[j.index], failed)
						break
					}
					r := runTrial(runctx, c, cs[j.caseIndex], j.repeat, attempt)
					results[j.index] = append(results[j.index], r)
					if r.Status != SetupError || runctx.Err() != nil {
						break
					}
				}
				last := results[j.index][len(results[j.index])-1]
				if c.Plan.StopOnInfrastructure && (last.Status == SetupError || last.UsageError != "") {
					cancel()
				}
			}
		})
	}
	for n := range count {
		jobs <- job{n, n / c.Plan.Repeats, n % c.Plan.Repeats}
	}
	close(jobs)
	wg.Wait()
	r := ExperimentRecord{Manifest: m}
	r.Manifest.State = "sealed"
	for _, group := range results {
		r.Trials = append(r.Trials, group...)
		for _, t := range group {
			if t.Status == Cancelled || t.Status == BudgetExhausted {
				r.Manifest.State = "incomplete"
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
	repeat, attempt int,
) TrialRecord {
	id := c.ID + "/" + cs.ID + "/" + identityPart(repeat) + "/" + identityPart(attempt)
	r := TrialRecord{
		ID:           id,
		CaseID:       cs.ID,
		CaseRevision: cs.Revision,
		Repeat:       repeat,
		Attempt:      attempt,
		Seed:         c.Plan.Seed + int64(repeat),
		States:       []string{"queued"},
		Cleanup:      CleanupStatus{State: "not_needed"},
	}
	cfg := c.Capture
	cfg.RequiredKinds = append(append([]string(nil), cfg.RequiredKinds...), cs.RequiredEvidence...)
	capture, err := NewCapture(cfg)
	if err != nil {
		capture, _ = NewCapture(c.Capture)
		r.Status = SetupError
		r.Reason = "unsupported_case_evidence"
		r.Evidence = capture.Seal()
		r.States = append(r.States, "terminal")
		return r
	}
	finish := func() { r.Evidence = capture.Seal(); r.States = append(r.States, "terminal") }
	if ctx.Err() != nil {
		r.Status = Cancelled
		r.Reason = "cancelled_before_prepare"
		finish()
		return r
	}
	trialctx, cancel := context.WithTimeout(ctx, c.Plan.Timeout)
	defer cancel()
	r.States = append(r.States, "preparing")
	env, err := c.Lifecycle.Prepare(trialctx, id)
	cleanup := func() {
		cleanctx, done := context.WithTimeout(context.WithoutCancel(ctx), c.Plan.CleanupTimeout)
		defer done()
		e := c.Lifecycle.Cleanup(cleanctx, env)
		if e == nil {
			e = cleanctx.Err()
		}
		r.Cleanup = CleanupStatus{State: "completed"}
		if e != nil {
			r.Cleanup = CleanupStatus{State: "failed", Reason: "cleanup_failure"}
		}
	}
	if err == nil {
		err = c.Lifecycle.Reset(trialctx, env)
	}
	if err != nil || trialctx.Err() != nil {
		r.Status = SetupError
		r.Reason = "setup_failure"
		if trialctx.Err() != nil {
			r.Status = Cancelled
			r.Reason = "setup_cancelled"
		}
		cleanup()
		finish()
		return r
	}
	var reservation Reservation
	if c.Budget != nil {
		reservation, err = c.Budget.Reserve(trialctx, id+"/target", c.Plan.DispatchUnits)
		if err != nil {
			r.Status = BudgetExhausted
			r.Reason = "reservation_failure"
			if trialctx.Err() != nil {
				r.Status = Cancelled
			}
			cleanup()
			finish()
			return r
		}
	}
	if trialctx.Err() != nil {
		if c.Budget != nil {
			releaseCtx, done := context.WithTimeout(context.WithoutCancel(ctx), c.Plan.CleanupTimeout)
			if e := c.Budget.Release(releaseCtx, reservation); e != nil {
				r.UsageError = "release_failure"
			}
			done()
		}
		r.Status = Cancelled
		cleanup()
		finish()
		return r
	}
	if c.Budget != nil {
		if e := c.Budget.Claim(trialctx, reservation); e != nil {
			r.Status = BudgetExhausted
			r.Reason = "dispatch_claim_failure"
			if trialctx.Err() != nil {
				r.Status = Cancelled
			}
			cleanup()
			finish()
			return r
		}
	}
	if trialctx.Err() != nil {
		r.Status = Cancelled
		r.Reason = "cancelled_before_dispatch"
		cleanup()
		finish()
		return r
	}
	r.States = append(r.States, "running")
	out, err := c.Target.Run(
		trialctx,
		cs.Input,
		TrialContext[E]{
			ID:           id,
			CaseRevision: cs.Revision,
			Repeat:       repeat,
			Attempt:      attempt,
			Seed:         r.Seed,
			Environment:  env,
			Evidence:     capture,
			Budget:       c.Budget,
		},
	)
	r.TargetUsage = out.Usage
	if !finiteNonnegative(out.Usage.Units) {
		r.TargetUsage = Usage{}
		r.UsageError = "invalid_usage"
	}
	if c.Budget != nil {
		usagectx, done := context.WithTimeout(context.WithoutCancel(ctx), c.Plan.CleanupTimeout)
		if e := c.Budget.Reconcile(usagectx, reservation, r.TargetUsage); e != nil {
			r.UsageError = "reconciliation_failure"
		}
		done()
	}
	r.Status = Completed
	if err != nil {
		r.Status = TargetError
		r.Reason = "target_failure"
	}
	if trialctx.Err() != nil {
		r.Status = Cancelled
		r.Reason = "target_cancelled"
	}
	r.States = append(r.States, "collecting")
	evidence := capture.Seal()
	if r.Status == Completed {
		if c.CriticalEvidence && evidence.State != "sealed" {
			r.Reason = "critical_evidence_loss"
		}
		// Recover original case input after target mutation via private dataset decoding.
		fresh, _ := c.Dataset.Cases()
		var original Case[I, R]
		for _, candidate := range fresh {
			if candidate.ID == cs.ID {
				original = candidate
				break
			}
		}
		outputBytes, outputErr := c.OutputCodec.Encode(out.Output)
		for _, g := range c.Graders {
			freshCases, decodeErr := c.Dataset.Cases()
			if decodeErr != nil {
				r.UsageError = "codec_failure"
				break
			}
			for _, item := range freshCases {
				if item.ID == cs.ID {
					original = item
					break
				}
			}
			var view View[I, O, R]
			projectErr := outputErr
			if projectErr == nil {
				freshOutput, decodeErr := c.OutputCodec.Decode(outputBytes)
				projectErr = decodeErr
				if projectErr == nil {
					freshEvidence, _ := cloneJSON(evidence)
					view, projectErr = c.Project(trialctx, original, freshOutput, freshEvidence)
				}
			}
			if projectErr != nil || (c.CriticalEvidence && evidence.State != "sealed") {
				r.Grades = append(
					r.Grades,
					Grade{
						Revision: g.Revision(),
						Status:   InsufficientEvidence,
						Reasons:  []string{"projection_unavailable"},
					},
				)
				continue
			}
			var gradeReservation Reservation
			if c.Budget != nil {
				gradeReservation, err = c.Budget.Reserve(trialctx, id+"/grader/"+g.Revision().ID, c.Plan.GraderUnits)
				if err != nil {
					r.Grades = append(
						r.Grades,
						Grade{
							Revision: g.Revision(),
							Status:   GraderError,
							Reasons:  []string{"grader_budget_exhausted"},
						},
					)
					continue
				}
			}
			if c.Budget != nil {
				if e := c.Budget.Claim(trialctx, gradeReservation); e != nil {
					r.Grades = append(
						r.Grades,
						Grade{
							Revision: g.Revision(),
							Status:   GraderError,
							Reasons:  []string{"grader_dispatch_claim_failure"},
						},
					)
					continue
				}
			}
			grade := Assess(trialctx, []Grader[I, O, R]{g}, view)[0]
			r.Grades = append(r.Grades, grade)
			if c.Budget != nil {
				usagectx, done := context.WithTimeout(context.WithoutCancel(ctx), c.Plan.CleanupTimeout)
				if e := c.Budget.Reconcile(usagectx, gradeReservation, grade.Usage); e != nil {
					r.UsageError = "grader_reconciliation_failure"
				}
				done()
			}
		}
	}
	cleanup()
	finish()
	return r
}

// RunPaired records seeded side order and allocates separate lifecycle handles.
func RunPaired[I, O, R, E any](
	ctx context.Context,
	baseline, candidate RunConfig[I, O, R, E],
	pairID string,
) (Experiment, Experiment, error) {
	if pairID == "" || baseline.Dataset.Revision() != candidate.Dataset.Revision() {
		return Experiment{}, Experiment{}, ErrInvalid
	}
	if baseline.ID == candidate.ID {
		return Experiment{}, Experiment{}, ErrConflict
	}
	order := []string{baseline.ID, candidate.ID}
	reverse := baseline.Plan.Seed%2 != 0
	if reverse {
		order[0], order[1] = order[1], order[0]
	}
	baseline.PairID = pairID
	candidate.PairID = pairID
	baseline.PairOrder = order
	candidate.PairOrder = order
	var b, c Experiment
	var e error
	if reverse {
		c, e = Run(ctx, candidate)
		if e == nil {
			b, e = Run(ctx, baseline)
		}
	} else {
		b, e = Run(ctx, baseline)
		if e == nil {
			c, e = Run(ctx, candidate)
		}
	}
	return b, c, e
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
	capture, _ := NewCapture(c.Capture)
	return TrialRecord{
		ID:           c.ID + "/" + cs.ID + "/" + identityPart(repeat) + "/" + identityPart(attempt),
		CaseID:       cs.ID,
		CaseRevision: cs.Revision,
		Repeat:       repeat,
		Attempt:      attempt,
		Seed:         c.Plan.Seed + int64(repeat),
		States:       []string{"queued", "terminal"},
		Status:       SetupError,
		Reason:       "codec_failure",
		Cleanup:      CleanupStatus{State: "not_needed"},
		Evidence:     capture.Seal(),
	}
}
