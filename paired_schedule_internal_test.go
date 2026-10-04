package evaly

import (
	"errors"
	"testing"
)

func TestPairScheduleRejectsForgedProvenance(t *testing.T) {
	for _, variant := range []string{"algorithm", "seed", "limit", "first", "missing_slot", "foreign_side", "shared", "pair_id"} {
		t.Run(variant, func(t *testing.T) {
			// Arrange.
			m := ExperimentManifest{
				ID: "baseline", PairID: "pair",
				Plan:      RunPlan{Seed: -2, Repeats: 2, Concurrency: 3},
				Lifecycle: LifecycleIdentity{Isolation: Isolated},
				Cases:     []CaseIdentity{{ID: "case", Revision: "revision"}},
			}
			schedule := newPairSchedule(m, "baseline", "candidate")
			m.PairSchedule = &schedule
			switch variant {
			case "algorithm":
				schedule.Revision = "unknown"
			case "seed":
				schedule.Seed++
			case "limit":
				schedule.Concurrency++
			case "first":
				schedule.Slots[0].First, schedule.Slots[0].Second = schedule.Slots[0].Second, schedule.Slots[0].First
			case "missing_slot":
				schedule.Slots = schedule.Slots[1:]
			case "foreign_side":
				schedule.Baseline = "foreign"
			case "shared":
				m.Lifecycle.Isolation = SerialShared
			case "pair_id":
				m.PairID = ""
			}
			// Act.
			err := validatePairSchedule(m)
			// Assert.
			if !errors.Is(err, ErrInvalid) {
				t.Fatal(variant, err)
			}
		})
	}
}
