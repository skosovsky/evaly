package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/skosovsky/evaly"
	"github.com/skosovsky/evaly/internal/fixtures"
)

func TestCLIExitCodesAndSealedBaseline(t *testing.T) {
	// Arrange.
	bin := filepath.Join(t.TempDir(), "evaly")
	build := exec.Command("go", "build", "-o", bin, ".")
	if b, e := build.CombinedOutput(); e != nil {
		t.Fatalf("build %v: %s", e, b)
	}
	store := t.TempDir()
	policyFile := filepath.Join(t.TempDir(), "policy.json")
	policy := evaly.ComparisonPolicy{
		Version:       1,
		ObjectiveKind: "assertion",
		Objective:     fixtures.Objective().Identity(),
		Gate:          fixtures.Gate(),
	}
	raw, err := json.Marshal(policy)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(policyFile, raw, 0600); err != nil {
		t.Fatal(err)
	}
	run := func(want int, args ...string) []byte {
		t.Helper()
		if args[0] == "compare" {
			args = append(args, "--policy", policyFile)
		}
		cmd := exec.Command(bin, args...)
		b, e := cmd.CombinedOutput()
		code := 0
		exit := &exec.ExitError{}
		if errors.As(e, &exit) {
			code = exit.ExitCode()
		} else if e != nil {
			t.Fatal(e)
		}
		if code != want {
			t.Fatalf("exit %d want %d: %s", code, want, b)
		}
		return b
	}
	// Act / Assert.
	for _, behavior := range []string{"good", "bad", "partial", "judge-error"} {
		run(0, "fixture", "--store", store, "--id", behavior, "--behavior", behavior)
	}
	pass := run(0, "compare", "--store", store, "--baseline", "good", "--candidate", "good")
	fail := run(1, "compare", "--store", store, "--baseline", "good", "--candidate", "bad")
	inconclusive := run(2, "compare", "--store", store, "--baseline", "good", "--candidate", "partial")
	run(2, "compare", "--store", store, "--baseline", "good", "--candidate", "judge-error")
	run(3, "compare", "--store", store, "--baseline", "missing", "--candidate", "good")
	run(3, "fixture", "--store", store, "--id", "../bad")
	if bytes.Contains(fail, []byte("evaly fixture")) || !bytes.Contains(fail, []byte("case case-4 revision")) ||
		!bytes.Contains(fail, []byte("trial bad/case-4/0/0")) ||
		!bytes.Contains(fail, []byte("evidence ")) {
		t.Fatal("missing host trial/evidence coordinates", string(fail))
	}
	// Act / Assert: an explicit caller threshold changes the gate result.
	policy.Gate.MinimumQuality = 1.1
	raw, _ = json.Marshal(policy)
	if err = os.WriteFile(policyFile, raw, 0600); err != nil {
		t.Fatal(err)
	}
	run(3, "compare", "--store", store, "--baseline", "good", "--candidate", "good")
	policy.Gate.MinimumQuality = .5
	policy.Gate.MaximumRegression = 1
	raw, _ = json.Marshal(policy)
	if err = os.WriteFile(policyFile, raw, 0600); err != nil {
		t.Fatal(err)
	}
	run(0, "compare", "--store", store, "--baseline", "good", "--candidate", "bad")
	for _, badPolicy := range []string{
		`{"version":2}`, `{"version":1}`, `{"version":1,"objective_kind":"callback"}`,
		string(bytes.ReplaceAll(raw, []byte(`"all-declared-v1"`), []byte(`"host-selector-v1"`))),
		string(bytes.ReplaceAll(raw, []byte(`"minimum_matched_cases":1`), []byte(`"minimum_matched_cases":1.5`))),
		string(bytes.ReplaceAll(raw, []byte(`"minimum_matched_cases":1`), []byte(`"minimum_matched_cases":0`))),
		string(bytes.ReplaceAll(raw, []byte(`"minimum_quality":0.5`), []byte(`"minimum_quality":1e999`))),
	} {
		if err = os.WriteFile(policyFile, []byte(badPolicy), 0600); err != nil {
			t.Fatal(err)
		}
		run(3, "compare", "--store", store, "--baseline", "good", "--candidate", "good")
	}
	// Missing --policy must never infer a fixture policy.
	missingOutput, missingErr := exec.Command(bin, "compare", "--store", store, "--baseline", "good", "--candidate", "good").
		CombinedOutput()
	missingExit := &exec.ExitError{}
	if !errors.As(missingErr, &missingExit) || missingExit.ExitCode() != 3 {
		t.Fatalf("missing policy: %v %s", missingErr, missingOutput)
	}
	if !bytes.Contains(pass, []byte("evaly pass")) || !bytes.Contains(fail, []byte("evaly fail")) ||
		!bytes.Contains(inconclusive, []byte("scored 2/4")) {
		t.Fatal("report/exit mismatch")
	}
}

func TestCLINumericPolicyForHostTarget(t *testing.T) {
	// Arrange: host-owned target metadata and a numeric metric, not a fixture gate.
	ctx := context.Background()
	directory := t.TempDir()
	store, err := evaly.OpenFileStore(directory)
	if err != nil {
		t.Fatal(err)
	}
	descriptor := evaly.ObjectiveIdentity{
		ID:                  "latency",
		Revision:            "1",
		Unit:                "ms",
		ScaleRevision:       "milliseconds-v1",
		Minimum:             0,
		Maximum:             100,
		Direction:           "lower",
		EligibilityRevision: "all-declared-v1",
		MissingnessRevision: "all-repeats-required-v1",
		AggregationRevision: "repeat-mean-case-mean-v1",
		SourceGrader:        "host-metric",
		SourceMetric:        "latency",
	}
	for id, value := range map[string]float64{"host-baseline": 50, "host-candidate": 10} {
		c, e := fixtures.CalculationConfig(id, "good", "")
		if e != nil {
			t.Fatal(e)
		}
		c.Provenance.Target = "customer-agent-build-42"
		c.Graders = []evaly.Grader[fixtures.Calculation, fixtures.CalculationOutput, int]{
			evaly.GraderFunc[fixtures.Calculation, fixtures.CalculationOutput, int]{
				Identity: evaly.GraderRevision{ID: "host-metric", Implementation: "1", Rubric: "1"},
				Evaluate: func(context.Context, evaly.View[fixtures.Calculation, fixtures.CalculationOutput, int]) (evaly.Grade, error) {
					return evaly.Grade{
						Status: evaly.Scored,
						Metrics: []evaly.Metric{
							{
								Name:          "latency",
								Unit:          "ms",
								ScaleRevision: "milliseconds-v1",
								Minimum:       0,
								Maximum:       100,
								Direction:     "lower",
								Value:         value,
							},
						},
						Usage: evaly.Usage{Known: true},
					}, nil
				},
			},
		}
		experiment, e := evaly.Run(ctx, c)
		if e != nil {
			t.Fatal(e)
		}
		if e = evaly.SaveExperiment(ctx, store, experiment); e != nil {
			t.Fatal(e)
		}
	}
	policy := evaly.ComparisonPolicy{
		Version:       1,
		ObjectiveKind: "numeric",
		Objective:     descriptor,
		Gate: evaly.GatePolicy{
			Revision:               "customer-gate-v1",
			MinimumMatchedCases:    4,
			MinimumMatchedCoverage: 1,
			MinimumCoverage:        1,
			MinimumQuality:         20,
			MaximumRegression:      0,
			BootstrapSamples:       100,
			Seed:                   7,
		},
	}
	raw, err := json.Marshal(policy)
	if err != nil {
		t.Fatal(err)
	}
	policyFile := filepath.Join(t.TempDir(), "policy.json")
	if err = os.WriteFile(policyFile, raw, 0600); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(t.TempDir(), "evaly")
	if output, e := exec.Command("go", "build", "-o", bin, ".").CombinedOutput(); e != nil {
		t.Fatalf("build: %v %s", e, output)
	}
	// Act.
	output, err := exec.Command(bin, "compare", "--store", directory, "--baseline", "host-baseline", "--candidate", "host-candidate", "--policy", policyFile).
		CombinedOutput()
	// Assert.
	if err != nil {
		t.Fatalf("numeric comparison: %v %s", err, output)
	}
	if bytes.Contains(output, []byte("evaly fixture")) || !bytes.Contains(output, []byte("customer-agent-build-42")) ||
		!bytes.Contains(output, []byte("evaly pass")) {
		t.Fatalf("incorrect host report: %s", output)
	}
}
