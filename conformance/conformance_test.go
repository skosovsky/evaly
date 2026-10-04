package conformance_test

import (
	"testing"

	"github.com/skosovsky/evaly"
	"github.com/skosovsky/evaly/conformance"
	"github.com/skosovsky/evaly/internal/fixtures"
)

func TestReferenceAdapters(t *testing.T) {
	t.Run("structural", func(t *testing.T) {
		// Arrange: constructor-created budget is configured; zero/typed nil are not.
		valid, err := evaly.NewMemoryBudget(2)
		if err != nil {
			t.Fatal(err)
		}
		var absent *evaly.MemoryBudget
		// Act / Assert: reusable conformance does not consume budget.
		conformance.Structural(t, valid, &evaly.MemoryBudget{}, absent,
			evaly.LifecycleFuncs[int]{}, evaly.TargetFunc[int, int, int](nil))
		if valid.Used() != 0 {
			t.Fatal("validation consumed budget")
		}
	})
	t.Run("file", func(t *testing.T) {
		dir := t.TempDir()
		conformance.Artifact(t, func() (evaly.ArtifactStore, error) { return evaly.OpenFileStore(dir) })
	})
	t.Run("budget", func(t *testing.T) {
		conformance.Budget(t, func() (evaly.Budget, error) { return evaly.NewMemoryBudget(2) })
	})
	t.Run("codec", func(t *testing.T) {
		conformance.Codec(t, fixtures.InputCodec(), fixtures.Calculation{Left: 1, Right: 2})
	})
	t.Run("target", func(t *testing.T) {
		c, e := fixtures.CalculationConfig("conformance", "good", "")
		if e != nil {
			t.Fatal(e)
		}
		conformance.Target(t, c)
		conformance.Lifecycle(t, c.Lifecycle)
		cases, e := c.Dataset.Cases()
		if e != nil {
			t.Fatal(e)
		}
		capture, e := evaly.NewCapture(c.Capture)
		if e != nil {
			t.Fatal(e)
		}
		conformance.Grader(
			t,
			c.Graders[0],
			func() (evaly.View[fixtures.Calculation, fixtures.CalculationOutput, int], error) {
				return evaly.View[fixtures.Calculation, fixtures.CalculationOutput, int]{
					Case:     cases[0],
					Output:   fixtures.CalculationOutput{Sum: 3},
					Evidence: capture.Seal(),
				}, nil
			},
		)
	})
	for _, dedup := range []bool{false, true} {
		t.Run("export", func(t *testing.T) {
			sink := &evaly.MemoryExport{Dedup: dedup}
			conformance.Export(t, sink, func() int { return len(sink.Records()) })
		})
	}
}
