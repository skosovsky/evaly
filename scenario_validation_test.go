package evaly

import (
	"context"
	"testing"
	"time"
)

type completedStep struct{}

func (completedStep) Revision() string { return "completed-step-v1" }
func (completedStep) Step(context.Context, int, ScenarioContext) (int, int, bool, error) {
	return 1, 1, true, nil
}

func TestScenarioRestoreRejectsContradictoryTrajectory(t *testing.T) {
	// Arrange: a complete encoded scenario with a valid codec contract.
	sc, oc := JSONCodec[int]{ID: "s", Version: "1"}, JSONCodec[int]{ID: "o", Version: "1"}
	initial, err := RunScenario(
		context.Background(),
		completedStep{},
		0,
		ScenarioPlan{Mode: "search", MaxSteps: 2, Timeout: time.Second},
		sc,
		oc,
	)
	if err != nil {
		t.Fatal(err)
	}
	for _, variant := range []string{"completed_zero_steps", "completed_missing_output", "step_limit_early", "generation_mode", "generation_identity"} {
		t.Run(variant, func(t *testing.T) {
			r, _ := cloneJSON(initial)
			switch variant {
			case "completed_zero_steps":
				r.Steps = 0
				r.Outputs = nil
			case "completed_missing_output":
				r.Outputs = nil
			case "step_limit_early":
				r.Stop = "step_limit"
			case "generation_mode":
				r.Plan.Generation = &Generation{Generator: "g", Mode: "replay"}
			case "generation_identity":
				r.Plan.Generation = &Generation{Mode: "search"}
			}
			r.Revision = ""
			b, _ := canonical(r)
			r.Revision = digest(b)
			// Act.
			_, restoreErr := RestoreScenario(r, sc, oc)
			_, draftErr := DraftFromScenario(Case[int, int]{ID: "a", Input: 1}, Generation{Generator: "g"}, r)
			// Assert: valid content hashes cannot authorize contradictory bounded trajectories.
			if restoreErr == nil || draftErr == nil {
				t.Fatalf("malformed %s accepted: restore=%v draft=%v", variant, restoreErr, draftErr)
			}
		})
	}
}
