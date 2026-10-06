package evaly

import "context"

func prepareTrial[I, O, R, E any](
	trialctx context.Context,
	c RunConfig[I, O, R, E],
	id string,
	r *TrialRecord,
) (E, bool) {
	r.States = append(r.States, "preparing")
	env, err := c.Lifecycle.Prepare(trialctx, id)
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
		return env, false
	}
	return env, true
}
func cleanupTrial[I, O, R, E any](ctx context.Context, c RunConfig[I, O, R, E], env E, r *TrialRecord) {
	cleanctx, done := context.WithTimeout(context.WithoutCancel(ctx), c.Plan.CleanupTimeout)
	defer done()
	e := c.Lifecycle.Cleanup(cleanctx, env)
	if e == nil {
		e = cleanctx.Err()
	}
	r.Cleanup = CleanupStatus{State: completedState, Reason: ""}
	if e != nil {
		r.Cleanup = CleanupStatus{State: failedState, Reason: "cleanup_failure"}
	}
}

func authorizeTrial[I, O, R, E any](
	trialctx, ctx context.Context,
	c RunConfig[I, O, R, E],
	id string,
	r *TrialRecord,
) (Reservation, bool) {
	var reservation Reservation
	var err error
	if c.Budget != nil {
		reservation, err = c.Budget.Reserve(trialctx, id+"/target", c.Plan.DispatchUnits)
		if err != nil {
			r.Status = BudgetExhausted
			r.Reason = "reservation_failure"
			if trialctx.Err() != nil {
				r.Status = Cancelled
			}
			return reservation, false
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
		return reservation, false
	}
	if c.Budget != nil {
		if e := c.Budget.Claim(trialctx, reservation); e != nil {
			r.Status = BudgetExhausted
			r.Reason = "dispatch_claim_failure"
			if trialctx.Err() != nil {
				r.Status = Cancelled
			}
			return reservation, false
		}
	}
	if trialctx.Err() != nil {
		r.Status = Cancelled
		r.Reason = "cancelled_before_dispatch"
		return reservation, false
	}
	return reservation, true
}

func dispatchTrial[I, O, R, E any](
	trialctx, ctx context.Context,
	c RunConfig[I, O, R, E],
	cs Case[I, R],
	caseIndex int,
	id string,
	env E,
	capture *Capture,
	reservation Reservation,
	r *TrialRecord,
) {
	r.States = append(r.States, "running")
	out, err := c.Target.Run(
		trialctx,
		cs.Input,
		TrialContext[E]{
			ID:           id,
			CaseRevision: cs.Revision,
			Repeat:       r.Repeat,
			Attempt:      r.Attempt,
			Seed:         r.Seed,
			Environment:  env,
			Evidence:     capture,
			Budget:       c.Budget,
		},
	)
	r.TargetUsage = out.Usage
	if !finiteNonnegative(out.Usage.Units) {
		r.TargetUsage = Usage{Known: false, Units: 0}
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
		gradeTrial(trialctx, ctx, c, caseIndex, out.Output, evidence, r)
	}
}
