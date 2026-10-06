package evaly

import "strings"

func validateExperimentManifest(m ExperimentManifest) error {
	if m.Version != experimentWireRevision {
		return ErrUnsupported
	}
	if !validArtifactID(m.ID) || m.Provenance.Target == "" || m.Dataset == "" || m.Mode != controlledMode ||
		m.Projection == "" ||
		m.CapturePolicy == "" ||
		len(m.Cases) == 0 {
		return ErrInvalid
	}
	if err := validatePlan(m.Plan); err != nil {
		return err
	}
	if len(m.Cases) > 1000000/m.Plan.Repeats {
		return ErrInvalid
	}
	if err := validatePairSchedule(m); err != nil {
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
	return nil
}

func experimentCases(m ExperimentManifest) (map[string]string, error) {
	cases := map[string]string{}
	for _, c := range m.Cases {
		if c.ID == "" || c.Revision == "" || cases[c.ID] != "" {
			return nil, ErrConflict
		}
		cases[c.ID] = c.Revision
	}
	return cases, nil
}

func validateExperimentTrials(r ExperimentRecord, cases map[string]string) error {
	m := r.Manifest
	seen := map[string]bool{}
	for _, t := range r.Trials {
		if seen[t.ID] || cases[t.CaseID] != t.CaseRevision || t.Repeat < 0 || t.Repeat >= m.Plan.Repeats ||
			t.Attempt < 0 ||
			t.Attempt >= m.Plan.MaxAttempts {
			return ErrInvalid
		}
		seen[t.ID] = true
		if err := validateTrialRecord(t); err != nil {
			return err
		}
	}
	return nil
}

func experimentGraders(m ExperimentManifest) (map[string]GraderRevision, error) {
	if len(m.Graders) == 0 {
		return nil, ErrInvalid
	}
	graderIDs := map[string]GraderRevision{}
	for _, g := range m.Graders {
		if g.ID == "" || g.Implementation == "" || g.Rubric == "" {
			return nil, ErrInvalid
		}
		if _, ok := graderIDs[g.ID]; ok {
			return nil, ErrConflict
		}
		graderIDs[g.ID] = g
	}
	return graderIDs, nil
}

func validateTrialGrading(r ExperimentRecord, graderIDs map[string]GraderRevision) error {
	for _, trial := range r.Trials {
		if err := validateOneTrialGrading(trial, graderIDs); err != nil {
			return err
		}
	}
	return nil
}

func validateExperimentAttempts(r ExperimentRecord, graderIDs map[string]GraderRevision) error {
	m := r.Manifest
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
	if err := validateAttemptContinuity(attempts); err != nil {
		return err
	}
	return validateExperimentSlots(r, slots)
}

func validateTrialRecord(t TrialRecord) error {
	if !finiteNonnegative(t.TargetUsage.Units) {
		return ErrInvalid
	}
	if err := validateTrialCleanup(t); err != nil {
		return err
	}
	chain := strings.Join(t.States, ",")
	full := "queued,preparing,running,collecting,terminal"
	if chain != "queued,terminal" && chain != "queued,preparing,terminal" && chain != full {
		return ErrInvalid
	}
	if (t.Status == Completed || t.Status == TargetError) && chain != full {
		return ErrInvalid
	}
	if chain == full && t.Cleanup.State == cleanupNotNeeded {
		return ErrInvalid
	}
	switch t.Status {
	case Completed, TargetError, SetupError, Cancelled, BudgetExhausted, InfrastructureStop:
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
	return nil
}
func validateTrialCleanup(t TrialRecord) error {
	switch t.Cleanup.State {
	case cleanupNotNeeded:
		if len(t.States) != 2 {
			return ErrInvalid
		}
	case completedState:
		if t.Cleanup.Reason != "" {
			return ErrInvalid
		}
	case failedState:
		if t.Cleanup.Reason == "" {
			return ErrInvalid
		}
	default:
		return ErrUnsupported
	}
	return nil
}

func validateOneTrialGrading(t TrialRecord, graderIDs map[string]GraderRevision) error {
	if t.GradingState != gradingComplete && t.GradingState != gradingPartial {
		return ErrInvalid
	}
	represented := map[string]bool{}
	for _, g := range t.Grades {
		if !g.Dispatched {
			return ErrInvalid
		}
		represented[g.Revision.ID] = true
	}
	for _, skipped := range t.SkippedGraders {
		if skipped.Reason == "" || graderIDs[skipped.Revision.ID] != skipped.Revision ||
			represented[skipped.Revision.ID] {
			return ErrInvalid
		}
		represented[skipped.Revision.ID] = true
	}
	if len(represented) != len(graderIDs) {
		return ErrIncomplete
	}
	if t.GradingState == gradingComplete && (len(t.SkippedGraders) > 0 || t.GradingStopReason != "") {
		return ErrInvalid
	}
	if t.GradingState == gradingPartial && t.GradingStopReason == "" {
		return ErrInvalid
	}
	return nil
}

func validateAttemptContinuity(attempts map[string]map[int]TrialRecord) error {
	for _, attemptSet := range attempts {
		for attempt := range len(attemptSet) {
			trial, ok := attemptSet[attempt]
			if !ok {
				return ErrInvalid
			}
			if attempt < len(attemptSet)-1 && trial.Status != SetupError {
				return ErrInvalid
			}
		}
	}
	return nil
}

func validateExperimentSlots(r ExperimentRecord, slots map[string]int) error {
	m := r.Manifest
	if m.PairSchedule != nil {
		for _, slot := range m.PairSchedule.Slots {
			if slots[slot.CaseID+"/"+identityPart(slot.Repeat)] == 0 {
				return ErrIncomplete
			}
		}
	}
	if m.State != sealedState {
		return nil
	}
	for _, cs := range m.Cases {
		for repeat := range m.Plan.Repeats {
			if slots[cs.ID+"/"+identityPart(repeat)] == 0 {
				return ErrIncomplete
			}
		}
	}
	for _, t := range r.Trials {
		if t.Status == Cancelled || t.Status == BudgetExhausted || t.Status == InfrastructureStop {
			return ErrIncomplete
		}
	}
	return nil
}
