package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/skosovsky/evaly"
	"github.com/skosovsky/evaly/internal/fixtures"
	"github.com/skosovsky/evaly/observation"
)

func main() {
	ctx := context.Background()
	baseline, _, err := fixtures.WorkflowConfig("workflow-baseline", "text-only")
	check(err)
	candidate, _, err := fixtures.WorkflowConfig("workflow-candidate", "alternative")
	check(err)
	var saved evaly.SavedView[fixtures.WorkflowInput, fixtures.WorkflowOutput, int]
	originalProject := candidate.Project
	candidate.Project = func(ctx context.Context, c evaly.Case[fixtures.WorkflowInput, int], out fixtures.WorkflowOutput, e evaly.EvidenceRecord) (evaly.View[fixtures.WorkflowInput, fixtures.WorkflowOutput, int], error) {
		view, err := originalProject(ctx, c, out, e)
		if err != nil {
			return view, err
		}
		saved, err = evaly.SaveView(
			view,
			candidate.ProjectionRevision,
			fixtures.WorkflowInputCodec(),
			fixtures.WorkflowOutputCodec(),
			fixtures.WorkflowReferenceCodec(),
		)
		return view, err
	}
	b, c, err := evaly.RunPaired(ctx, baseline, candidate, "workflow-pair")
	check(err)
	comparison, err := evaly.Compare(
		b,
		c,
		evaly.AssertionObjective{ID: "workflow-outcome", Revision: "1", Policy: "all"},
		fixtures.Gate(),
	)
	check(err)
	fmt.Printf(
		"confident text quality %.0f; external credit quality %.0f; gate %s\n",
		comparison.MatchedBaselineMean,
		comparison.MatchedCandidateMean,
		comparison.Verdict,
	)
	effect, store, err := fixtures.WorkflowConfig("workflow-after-effect", "effect-error")
	check(err)
	failed, err := evaly.Run(ctx, effect)
	check(err)
	audit, _, _, _, _ := store.Snapshot()
	fmt.Printf(
		"after effect: %s, usage %.0f, external effects %d, retained events %d\n",
		failed.Record().Trials[0].Status,
		failed.Record().Trials[0].TargetUsage.Units,
		len(audit),
		len(failed.Record().Trials[0].Evidence.Events),
	)
	directory, err := os.MkdirTemp("", "evaly-workflow-")
	check(err)
	defer os.RemoveAll(directory)
	artifacts, err := evaly.OpenFileStore(directory)
	check(err)
	check(evaly.SaveExperiment(ctx, artifacts, c))
	check(evaly.SaveSavedView(ctx, artifacts, "workflow-view", saved))
	reopened, err := evaly.OpenFileStore(directory)
	check(err)
	restored, err := evaly.LoadExperiment(ctx, reopened, candidate.ID)
	check(err)
	saved, err = evaly.LoadSavedView(
		ctx,
		reopened,
		"workflow-view",
		fixtures.WorkflowInputCodec(),
		fixtures.WorkflowOutputCodec(),
		fixtures.WorkflowReferenceCodec(),
	)
	check(err)
	first, err := evaly.Rescore(
		ctx,
		saved,
		fixtures.WorkflowGraders("external-balance-v1"),
		restored.Revision(),
		"",
		"rescore",
	)
	check(err)
	second, err := evaly.Rescore(
		ctx,
		saved,
		fixtures.WorkflowGraders("external-balance-v2"),
		restored.Revision(),
		first.Revision,
		"rescore",
	)
	check(err)
	fmt.Printf(
		"offline rescore: new lineage %t; source unchanged %t\n",
		second.Parent == first.Revision && second.Revision != first.Revision,
		restored.Revision() == c.Revision(),
	)
	budget, err := evaly.NewMemoryBudget(1)
	check(err)
	worker, err := observation.Start(
		ctx,
		observation.Config[fixtures.WorkflowInput, fixtures.WorkflowOutput, int]{
			Capacity:    1,
			Concurrency: 1,
			Deadline:    time.Second,
			Clock:       observation.RealClock{},
			Graders:     fixtures.WorkflowGraders("external-balance-v2"),
			Budget:      budget,
			GraderUnits: 1,
		},
	)
	check(err)
	defer worker.Cancel(ctx)
	obs, err := observation.New(
		"workflow-observation",
		second.Revision,
		observation.Sampling{Rule: "all", Population: "offline-fixture", Window: "example"},
		saved,
	)
	check(err)
	future, status := worker.Enqueue(obs)
	if status != "accepted" {
		log.Fatal(status)
	}
	result := <-future.Result
	fmt.Printf(
		"online assessment: %s, paid grades %d, skipped %d\n",
		result.Assessment.State,
		len(result.Assessment.Grades),
		len(result.Assessment.Skipped),
	)
	fmt.Println("live host agent/judge check: not performed; no credentials or paid calls")
}
func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
