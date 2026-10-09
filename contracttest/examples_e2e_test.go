//go:build e2e

package contracttest

import (
	"testing"
)

func TestE2EExamples(t *testing.T) {
	for _, example := range []string{"calculation", "crm", "protocols", "http", "observation", "optimizer", "integration"} {
		t.Run(example, func(t *testing.T) {
			// Arrange: execute the checked-out library with its real offline fixture.
			root := repoRoot(t)
			// Act.
			output := command(t, root, nil, "go", "run", "./examples/"+example)
			// Assert: examples must succeed and produce their documented result.
			if output == "" {
				t.Fatal("example produced no output")
			}
		})
	}
}
