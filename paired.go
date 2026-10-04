package evaly

import (
	"context"
	"sync"
	"time"
)

// PairSlot declares the within-pair order, not timing between concurrent pairs.
type PairSlot struct {
	CaseID       string `json:"case_id"`
	CaseRevision string `json:"case_revision"`
	Repeat       int    `json:"repeat"`
	First        string `json:"first"`
	Second       string `json:"second"`
}

// PairSchedule records the entire planned schedule, including undispatched slots.
type PairSchedule struct {
	Revision    string     `json:"revision"`
	Seed        int64      `json:"seed"`
	Concurrency int        `json:"concurrency"`
	Baseline    string     `json:"baseline"`
	Candidate   string     `json:"candidate"`
	Slots       []PairSlot `json:"slots"`
}

func pairFirst(seed int64, ordinal int, baseline, candidate string) (string, string) {
	// Fixed SplitMix64, independent of Go's random package or process state.
	x := uint64(seed) + uint64(ordinal+1)*0x9e3779b97f4a7c15
	x = (x ^ (x >> 30)) * 0xbf58476d1ce4e5b9
	x = (x ^ (x >> 27)) * 0x94d049bb133111eb
	x ^= x >> 31
	if x&1 != 0 {
		return candidate, baseline
	}
	return baseline, candidate
}

func newPairSchedule(m ExperimentManifest, baseline, candidate string) PairSchedule {
	s := PairSchedule{
		Revision: "case-repeat-v1", Seed: m.Plan.Seed, Concurrency: m.Plan.Concurrency,
		Baseline: baseline, Candidate: candidate, Slots: make([]PairSlot, 0, len(m.Cases)*m.Plan.Repeats),
	}
	for _, cs := range m.Cases {
		for repeat := range m.Plan.Repeats {
			first, second := pairFirst(s.Seed, len(s.Slots), baseline, candidate)
			s.Slots = append(s.Slots, PairSlot{cs.ID, cs.Revision, repeat, first, second})
		}
	}
	return s
}

func validatePairSchedule(m ExperimentManifest) error {
	if m.PairSchedule == nil {
		if m.PairID != "" {
			return ErrInvalid
		}
		return nil
	}
	s := m.PairSchedule
	if !validArtifactID(m.PairID) || !validArtifactID(s.Baseline) || !validArtifactID(s.Candidate) ||
		s.Baseline == s.Candidate || (m.ID != s.Baseline && m.ID != s.Candidate) || m.Lifecycle.Isolation != Isolated {
		return ErrInvalid
	}
	expected := newPairSchedule(m, s.Baseline, s.Candidate)
	a, err := canonical(s)
	if err != nil {
		return ErrInvalid
	}
	b, err := canonical(expected)
	if err != nil || string(a) != string(b) {
		return ErrInvalid
	}
	return nil
}

func pairedConfigurationCompatible(a, b ExperimentManifest) bool {
	if a.Dataset != b.Dataset || a.Selection != b.Selection || a.Lifecycle != b.Lifecycle ||
		a.Lifecycle.Isolation != Isolated || a.Projection != b.Projection || a.Plan != b.Plan ||
		a.OutputCodec != b.OutputCodec || a.CapturePolicy != b.CapturePolicy ||
		a.CaptureMaxEvents != b.CaptureMaxEvents || a.CaptureMaxBytes != b.CaptureMaxBytes ||
		a.CriticalEvidence != b.CriticalEvidence {
		return false
	}
	for _, pair := range [][2]any{
		{a.Cases, b.Cases}, {a.Graders, b.Graders},
		{a.CaptureRequirements, b.CaptureRequirements}, {a.CaptureKinds, b.CaptureKinds},
	} {
		x, err := canonical(pair[0])
		if err != nil {
			return false
		}
		y, err := canonical(pair[1])
		if err != nil || string(x) != string(y) {
			return false
		}
	}
	return true
}

func executeSlot[I, O, R, E any](ctx context.Context, c RunConfig[I, O, R, E], caseIndex, repeat int) []TrialRecord {
	var results []TrialRecord
	for attempt := range c.Plan.MaxAttempts {
		cs, err := c.Dataset.CaseAt(caseIndex)
		if err != nil {
			original := c.Dataset.record.Cases[caseIndex]
			results = append(
				results,
				codecFailure(c, Case[I, R]{ID: original.ID, Revision: original.Revision}, repeat, attempt),
			)
			break
		}
		r := runTrial(ctx, c, cs, caseIndex, repeat, attempt)
		results = append(results, r)
		if r.Status != SetupError || r.Cleanup.State == "failed" || ctx.Err() != nil {
			break
		}
	}
	return results
}

func pairedRecord(m ExperimentManifest, results [][]TrialRecord) ExperimentRecord {
	r := ExperimentRecord{Manifest: m}
	r.Manifest.State = "sealed"
	for _, group := range results {
		r.Trials = append(r.Trials, group...)
		for _, trial := range group {
			if trial.Status == Cancelled || trial.Status == BudgetExhausted || trial.Status == InfrastructureStop {
				r.Manifest.State = "incomplete"
			}
		}
	}
	r.Manifest.States = append(r.Manifest.States, r.Manifest.State)
	r.Manifest.Finished = time.Now().UTC()
	return r
}

// RunPaired interleaves both sides of every case/repeat with a seeded side order.
// Concurrency bounds active pairs; order across concurrent workers is unspecified.
func RunPaired[I, O, R, E any](
	ctx context.Context,
	baseline, candidate RunConfig[I, O, R, E],
	pairID string,
) (Experiment, Experiment, error) {
	if err := ValidateRunConfig(baseline); err != nil {
		return Experiment{}, Experiment{}, err
	}
	if err := ValidateRunConfig(candidate); err != nil {
		return Experiment{}, Experiment{}, err
	}
	if baseline.ID == candidate.ID {
		return Experiment{}, Experiment{}, ErrConflict
	}
	bm, cm := experimentManifest(baseline), experimentManifest(candidate)
	if !validArtifactID(pairID) || !pairedConfigurationCompatible(bm, cm) {
		return Experiment{}, Experiment{}, ErrInvalid
	}
	schedule := newPairSchedule(bm, baseline.ID, candidate.ID)
	bm.PairID, cm.PairID = pairID, pairID
	bm.PairSchedule, cm.PairSchedule = &schedule, &schedule
	count := len(schedule.Slots)
	br, cr := make([][]TrialRecord, count), make([][]TrialRecord, count)
	runctx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)
	jobs := make(chan int)
	workers := min(schedule.Concurrency, count)
	var wg sync.WaitGroup
	for range workers {
		wg.Go(func() {
			for ordinal := range jobs {
				slot := schedule.Slots[ordinal]
				configs := []RunConfig[I, O, R, E]{baseline, candidate}
				if slot.First == candidate.ID {
					configs[0], configs[1] = candidate, baseline
				}
				for _, config := range configs {
					trials := executeSlot(runctx, config, ordinal/config.Plan.Repeats, slot.Repeat)
					if config.ID == baseline.ID {
						br[ordinal] = trials
					} else {
						cr[ordinal] = trials
					}
					if config.Plan.StopOnInfrastructure && trialInfrastructureFailure(trials[len(trials)-1]) {
						cancel(errInfrastructureStopped)
					}
				}
			}
		})
	}
	for ordinal := range count {
		jobs <- ordinal
	}
	close(jobs)
	wg.Wait()
	b, err := freezeExperiment(pairedRecord(bm, br))
	if err != nil {
		return b, Experiment{}, err
	}
	c, err := freezeExperiment(pairedRecord(cm, cr))
	return b, c, err
}
