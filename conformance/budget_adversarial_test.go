package conformance_test

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/skosovsky/evaly"
	"github.com/skosovsky/evaly/conformance"
)

type noOpClaimBudget struct{ evaly.Budget }

func (noOpClaimBudget) Claim(context.Context, evaly.Reservation) error { return nil }

func TestBudgetSuiteRejectsNoOpClaim(t *testing.T) {
	// Arrange: adapter has real accounting, but falsely authorizes every Claim.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestBudgetNoOpClaimProcess$", "-test.v")
	command.Env = append(os.Environ(), "EVALY_NOOP_CLAIM=1")
	// Act.
	output, err := command.CombinedOutput()
	// Assert: stock Budget conformance must reject both serial and concurrent redispatch.
	if ctx.Err() != nil {
		t.Fatal("child did not finish", ctx.Err())
	}
	if err == nil || !strings.Contains(string(output), "duplicate claim authorized") ||
		!strings.Contains(string(output), "claim must authorize exactly once") {
		t.Fatal(err, string(output))
	}
}

func TestBudgetNoOpClaimProcess(t *testing.T) {
	if os.Getenv("EVALY_NOOP_CLAIM") != "1" {
		return
	}
	// Arrange: each protocol check gets a fresh two-unit adapter.
	factory := func() (evaly.Budget, error) {
		budget, err := evaly.NewMemoryBudget(2)
		return noOpClaimBudget{Budget: budget}, err
	}
	// Act / Assert: expected failure is inspected by the parent process.
	conformance.Budget(t, factory)
}
