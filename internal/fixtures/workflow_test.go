package fixtures

import (
	"context"
	"errors"
	"testing"

	"github.com/skosovsky/evaly"
)

func TestWorkflowLifecycleResetRestoresTypedState(t *testing.T) {
	// Arrange.
	c, store, err := WorkflowConfig("reset-workflow", "refund")
	if err != nil {
		t.Fatal(err)
	}
	env, err := c.Lifecycle.Prepare(context.Background(), "reset-account")
	if err != nil {
		t.Fatal(err)
	}
	store.Apply(env.Namespace, "refund", 100)
	// Act.
	err = c.Lifecycle.Reset(context.Background(), env)
	// Assert: external state resets while audit still proves the preceding effect.
	state := store.Read(env.Namespace)
	audit, active, _, prepared, cleaned := store.Snapshot()
	if err != nil || state != (WorkflowState{Eligible: true}) || len(audit) != 1 || active != 1 || prepared != 1 ||
		cleaned != 0 {
		t.Fatal(err, state, audit, active, prepared, cleaned)
	}
	if err = c.Lifecycle.Cleanup(context.Background(), env); err != nil {
		t.Fatal(err)
	}
	_, active, _, _, cleaned = store.Snapshot()
	if active != 0 || cleaned != 1 {
		t.Fatal(active, cleaned)
	}
}
func TestWorkflowRejectsUnknownModeBeforeCapabilities(t *testing.T) {
	// Arrange and act.
	c, store, err := WorkflowConfig("unknown-workflow", "unrecognized")
	// Assert.
	if !errors.Is(err, evaly.ErrUnsupported) || store != nil || c.Target != nil || c.Lifecycle != nil {
		t.Fatal(c, store, err)
	}
}
