package evaly

import "context"

type trialGrading[I, O, R, E any] struct {
	config    RunConfig[I, O, R, E]
	ctx       context.Context
	parent    context.Context
	index     int
	evidence  EvidenceRecord
	record    *TrialRecord
	encoded   []byte
	encodeErr error
}

func gradeTrial[I, O, R, E any](
	ctx, parent context.Context,
	c RunConfig[I, O, R, E],
	index int,
	output O,
	evidence EvidenceRecord,
	record *TrialRecord,
) {
	if c.CriticalEvidence && evidence.State != sealedState {
		record.Reason = "critical_evidence_loss"
	}
	var grading trialGrading[I, O, R, E]
	grading.config = c
	grading.ctx = ctx
	grading.parent = parent
	grading.index = index
	grading.evidence = evidence
	grading.record = record
	grading.encoded, grading.encodeErr = c.OutputCodec.Encode(output)
	for _, grader := range c.Graders {
		if !grading.evaluate(grader) {
			break
		}
	}
}
func (g *trialGrading[I, O, R, E]) freshView() (View[I, O, R], error) {
	var zero View[I, O, R]
	if g.encodeErr != nil {
		return zero, g.encodeErr
	}
	original, err := g.config.Dataset.CaseAt(g.index)
	if err != nil {
		return zero, err
	}
	output, err := g.config.OutputCodec.Decode(append([]byte(nil), g.encoded...))
	if err != nil {
		return zero, err
	}
	evidence, err := cloneJSON(g.evidence)
	if err != nil {
		return zero, err
	}
	return g.config.Project(g.ctx, original, output, evidence)
}
func (g *trialGrading[I, O, R, E]) fail(reason string) bool {
	g.record.UsageError = reason
	return false
}
func (g *trialGrading[I, O, R, E]) reserve(grader Grader[I, O, R]) (Reservation, bool) {
	var reservation Reservation
	if g.config.Budget == nil {
		return reservation, true
	}
	var err error
	reservation, err = g.config.Budget.Reserve(
		g.ctx,
		g.record.ID+"/grader/"+grader.Revision().ID,
		g.config.Plan.GraderUnits,
	)
	if err != nil {
		return reservation, g.fail("grader_budget_exhausted")
	}
	if err = g.config.Budget.Claim(g.ctx, reservation); err != nil {
		return reservation, g.fail("grader_dispatch_claim_failure")
	}
	return reservation, true
}
func (g *trialGrading[I, O, R, E]) evaluate(grader Grader[I, O, R]) bool {
	if g.ctx.Err() != nil {
		return g.fail("grading_cancelled")
	}
	if g.config.CriticalEvidence && g.evidence.State != sealedState {
		return g.fail("projection_unavailable")
	}
	view, err := g.freshView()
	if err != nil {
		return g.fail("projection_unavailable")
	}
	supplied := false
	permitted := func() (View[I, O, R], error) {
		if supplied {
			return View[I, O, R]{}, ErrConflict
		}
		supplied = true
		return view, nil
	}
	reservation, allowed := g.reserve(grader)
	if !allowed {
		return false
	}
	if g.ctx.Err() != nil {
		return g.fail("grading_cancelled")
	}
	grade := Assess(g.ctx, []Grader[I, O, R]{grader}, permitted)[0]
	if !grade.Dispatched {
		return g.fail("grading_not_dispatched")
	}
	g.record.Grades = append(g.record.Grades, grade)
	if !g.reconcile(reservation, grade.Usage) {
		return false
	}
	return !g.config.Plan.StopOnInfrastructure || grade.Status != GraderError
}
func (g *trialGrading[I, O, R, E]) reconcile(reservation Reservation, usage Usage) bool {
	if g.config.Budget == nil {
		return true
	}
	ctx, cancel := context.WithTimeout(context.WithoutCancel(g.parent), g.config.Plan.CleanupTimeout)
	defer cancel()
	if err := g.config.Budget.Reconcile(ctx, reservation, usage); err != nil {
		return g.fail("grader_reconciliation_failure")
	}
	return true
}
