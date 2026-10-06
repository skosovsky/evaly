package conformance_test

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/skosovsky/evaly"
	"github.com/skosovsky/evaly/conformance"
)

func TestBaselineLifecycleLeak(t *testing.T) {
	// Arrange: a successfully returned handle, then Reset failure in a child process.
	marker := filepath.Join(t.TempDir(), "cleanup")
	command := exec.Command(os.Args[0], "-test.run=^TestBaselineLifecycleChild$", "-test.v")
	command.Env = append(os.Environ(), "EVALY_BASELINE_CLEANUP="+marker)
	// Act.
	output, err := command.CombinedOutput()
	// Assert: original suite fails Reset but does not clean the owned handle.
	if err == nil || !strings.Contains(string(output), "baseline_reset_failure") {
		t.Fatal(err, string(output))
	}
	if _, err := os.Stat(marker); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("expected leaked handle; cleanup marker exists", err)
	}
	t.Log("F10 reproduced: Reset failed, Cleanup marker absent")
}

func TestBaselineLifecycleChild(t *testing.T) {
	marker := os.Getenv("EVALY_BASELINE_CLEANUP")
	if marker == "" {
		return
	}
	lifecycle := evaly.LifecycleFuncs[int]{
		IdentityValue: evaly.LifecycleIdentity{Fixture: "fixture-v1", Reset: "reset-v1", Isolation: evaly.SerialShared},
		PrepareFunc:   func(context.Context, string) (int, error) { return 7, nil },
		ResetFunc:     func(context.Context, int) error { return errors.New("baseline_reset_failure") },
		CleanupFunc:   func(context.Context, int) error { return os.WriteFile(marker, []byte("cleanup"), 0600) },
	}
	conformance.Lifecycle(t, lifecycle)
}

type baselineNoOpClaim struct{ evaly.Budget }

func (baselineNoOpClaim) Claim(context.Context, evaly.Reservation) error { return nil }

func TestBaselineBudgetAcceptsNoOpClaim(t *testing.T) {
	// Arrange: genuine atomic accounting with an intentionally meaningless Claim.
	factory := func() (evaly.Budget, error) {
		budget, err := evaly.NewMemoryBudget(2)
		return baselineNoOpClaim{budget}, err
	}
	// Act / Assert: original suite incorrectly accepts this adapter.
	conformance.Budget(t, factory)
	t.Log("D51 reproduced: no-op Claim adapter passed the suite")
}
