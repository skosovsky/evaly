// Package fixtures contains offline executable reference targets and graders.
package fixtures

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/skosovsky/evaly"
)

type Calculation struct {
	Left  int `json:"left"`
	Right int `json:"right"`
}
type CalculationOutput struct {
	Sum int `json:"sum"`
}
type Environment struct {
	Namespace string `json:"Namespace"`
	Ready     bool   `json:"Ready"`
}

func InputCodec() evaly.JSONCodec[Calculation] {
	return evaly.JSONCodec[Calculation]{ID: "calculation-input", Version: "1"}
}
func OutputCodec() evaly.JSONCodec[CalculationOutput] {
	return evaly.JSONCodec[CalculationOutput]{ID: "calculation-output", Version: "1"}
}
func ReferenceCodec() evaly.JSONCodec[int] {
	return evaly.JSONCodec[int]{ID: "integer-reference", Version: "1"}
}

func CalculationConfig(
	id, behavior, onlyCase string,
) (evaly.RunConfig[Calculation, CalculationOutput, int, *Environment], error) {
	var zeroGraderRevision evaly.GraderRevision
	var zeroUsage evaly.Usage
	if behavior != "good" && behavior != "bad" && behavior != "partial" && behavior != "judge-error" {
		return evaly.RunConfig[Calculation, CalculationOutput, int, *Environment]{}, evaly.ErrInvalid
	}
	cases := calculationCases(onlyCase)
	dataset, e := (evaly.DatasetDraft[Calculation, int]{Cases: cases, Selection: assertionAll, ParentRevision: ""}).Seal(
		InputCodec(),
		ReferenceCodec(),
	)
	if e != nil {
		return evaly.RunConfig[Calculation, CalculationOutput, int, *Environment]{}, e
	}
	c := evaly.RunConfig[Calculation, CalculationOutput, int, *Environment]{
		ID:          id,
		OutputCodec: OutputCodec(),
		Dataset:     dataset,
		Plan: evaly.RunPlan{
			Repeats:         1,
			Concurrency:     2,
			Timeout:         time.Second,
			CleanupTimeout:  time.Second,
			MaxAttempts:     1,
			AssertionPolicy: assertionAll,
			DispatchUnits:   1, Seed: 0, StopOnInfrastructure: false, GraderUnits: 0,
		},
		Provenance: evaly.Provenance{
			Target: behavior + "-v1",
			Model:  absentCapabilityRevision,
			Prompt: absentCapabilityRevision,
			Tools:  absentCapabilityRevision,
			Policy: "fixture-v1", Provider: nil, Unknown: nil,
		},
		Capture: evaly.CaptureConfig{
			Policy:     evaly.FieldPolicy{ID: "safe-v1", Allowed: nil, KeepReferences: false},
			KnownKinds: []string{toolEvidenceKind, outcomeEvidenceKind},
			MaxEvents:  calculationMaxEvents,
			MaxBytes:   fixtureMaxBytes, RequiredKinds: nil,
		},
		ProjectionRevision: "calculation-v1",
		Target:             nil,
		Lifecycle:          nil,
		Graders:            nil,
		Project:            nil,
		Budget:             nil,
		CriticalEvidence:   false,
	}
	c.Lifecycle = evaly.LifecycleFuncs[*Environment]{
		IdentityValue: evaly.LifecycleIdentity{Fixture: "calculation-v1", Reset: "empty-v1", Isolation: evaly.Isolated},
		PrepareFunc: func(ctx context.Context, namespace string) (*Environment, error) {
			env := &Environment{Namespace: namespace, Ready: false}
			if behavior == "partial" &&
				(strings.Contains(namespace, "/case-3/") || strings.Contains(namespace, "/case-4/")) {
				return env, errors.New("fixture unavailable")
			}
			return env, ctx.Err()
		},
		ResetFunc: func(ctx context.Context, e *Environment) error { e.Ready = true; return ctx.Err() },
		CleanupFunc: func(ctx context.Context, e *Environment) error {
			if e != nil {
				e.Ready = false
			}
			return ctx.Err()
		},
	}
	c.Target = evaly.TargetFunc[Calculation, CalculationOutput, *Environment](
		func(ctx context.Context, i Calculation, t evaly.TrialContext[*Environment]) (evaly.TargetResult[CalculationOutput], error) {
			return calculationInvocation(ctx, i, t, behavior)
		},
	)
	c.Project = func(ctx context.Context, cs evaly.Case[Calculation, int], o CalculationOutput, e evaly.EvidenceRecord) (evaly.View[Calculation, CalculationOutput, int], error) {
		return evaly.View[Calculation, CalculationOutput, int]{Case: cs, Output: o, Evidence: e}, ctx.Err()
	}
	c.Graders = []evaly.Grader[Calculation, CalculationOutput, int]{
		evaly.GraderFunc[Calculation, CalculationOutput, int]{
			Identity: evaly.GraderRevision{
				ID:             "sum",
				Implementation: goImplementationRevision,
				Rubric:         "exact-v1",
				Model:          "",
				Prompt:         "",
				Configuration:  "",
			},
			Evaluate: func(ctx context.Context, v evaly.View[Calculation, CalculationOutput, int]) (evaly.Grade, error) {
				return gradeCalculation(ctx, v, &behavior, &zeroGraderRevision, &zeroUsage)
			},
		},
	}
	return c, nil
}
func Gate() evaly.GatePolicy {
	return evaly.GatePolicy{
		Revision:               "fixture-v1",
		MinimumCoverage:        1,
		MinimumMatchedCoverage: 1,
		MinimumMatchedCases:    1,
		QualityThreshold:       minimumCalculationQuality,
		MaximumRegression:      0,
		BootstrapSamples:       fixtureBootstrapSamples,
		Seed:                   fixtureBootstrapSeed,
	}
}

func Objective() evaly.AssertionObjective {
	return evaly.AssertionObjective{ID: "assertion-pass", Revision: "1", Policy: assertionAll}
}

func gradeCalculation(
	ctx context.Context,
	v evaly.View[Calculation, CalculationOutput, int],
	behavior *string,
	zeroGraderRevision *evaly.
		GraderRevision,
	zeroUsage *evaly.
		Usage,
) (evaly.Grade, error) {
	if (*behavior) == "judge-error" {
		return evaly.Grade{}, context.DeadlineExceeded
	}
	if v.Case.Reference == nil {
		return evaly.Grade{
			Status:       evaly.NotApplicable,
			Reasons:      []string{"missing_reference"},
			Dispatched:   false,
			Revision:     (*zeroGraderRevision),
			Metrics:      nil,
			Assertions:   nil,
			EvidenceRefs: nil,
			Usage:        (*zeroUsage),
		}, nil
	}
	return evaly.Grade{
		Status: evaly.Scored,
		Assertions: []evaly.Assertion{
			{Name: "sum_equal", Pass: v.Output.Sum == *v.Case.Reference, Reason: ""},
		},
		Usage: evaly.Usage{
			Known: true,
			Units: 0,
		},
		Dispatched:   false,
		Revision:     (*zeroGraderRevision),
		Metrics:      nil,
		Reasons:      nil,
		EvidenceRefs: nil,
	}, ctx.Err()
}

func calculationCases(onlyCase string) []evaly.Case[Calculation, int] {
	cases := []evaly.Case[Calculation, int]{}
	for i := 1; i <= 4; i++ {
		caseID := "case-" + strconv.Itoa(i)
		if onlyCase != "" && onlyCase != caseID {
			continue
		}
		ref := i + 2
		cases = append(
			cases,
			evaly.Case[Calculation, int]{
				ID:               caseID,
				Input:            Calculation{i, 2},
				Reference:        &ref,
				Revision:         "",
				Metadata:         nil,
				RequiredEvidence: nil,
				Generation:       nil,
			},
		)
	}
	return cases
}

func calculationInvocation(
	ctx context.Context,
	i Calculation,
	t evaly.TrialContext[*Environment],
	behavior string,
) (evaly.TargetResult[CalculationOutput], error) {
	if !t.Environment.Ready {
		return evaly.TargetResult[CalculationOutput]{}, evaly.ErrInvalid
	}
	sum := i.Left + i.Right
	if behavior == "bad" && i.Left == 4 {
		sum++
	}
	return evaly.TargetResult[CalculationOutput]{
		Output: CalculationOutput{sum},
		Usage:  evaly.Usage{Known: true, Units: 1},
	}, ctx.Err()
}
