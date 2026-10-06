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
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	ctx := context.Background()
	baseline, _, err := fixtures.WorkflowConfig("workflow-baseline", "text-only")
	if err != nil {
		return err
	}
	candidate, _, err := fixtures.WorkflowConfig("workflow-candidate", "alternative")
	if err != nil {
		return err
	}
	var saved evaly.SavedView[fixtures.WorkflowInput, fixtures.WorkflowOutput, int]
	originalProject := candidate.Project
	candidate.Project = func(ctx context.Context, c evaly.Case[fixtures.WorkflowInput, int], out fixtures.WorkflowOutput, e evaly.EvidenceRecord) (evaly.View[fixtures.WorkflowInput, fixtures.WorkflowOutput, int], error) {
		view, errLocal := originalProject(ctx, c, out, e)
		if errLocal != nil {
			return view, errLocal
		}
		saved, errLocal = evaly.SaveView(
			view,
			candidate.ProjectionRevision,
			fixtures.WorkflowInputCodec(),
			fixtures.WorkflowOutputCodec(),
			fixtures.WorkflowReferenceCodec(),
		)
		return view, errLocal
	}
	b, c, err := evaly.RunPaired(ctx, baseline, candidate, "workflow-pair")
	if err != nil {
		return err
	}
	comparison, err := evaly.Compare(
		b,
		c,
		evaly.AssertionObjective{ID: "workflow-outcome", Revision: "1", Policy: "all"},
		fixtures.Gate(),
	)
	if err != nil {
		return err
	}
	fmt.Printf(
		"confident text quality %.0f; external credit quality %.0f; gate %s\n",
		comparison.MatchedBaselineMean,
		comparison.MatchedCandidateMean,
		comparison.Verdict,
	)
	if err = effectExample(ctx); err != nil {
		return err
	}
	return artifactExample(ctx, c, candidate.ID, saved)
}

func effectExample(ctx context.Context) error {
	effect, store, err := fixtures.WorkflowConfig("workflow-after-effect", "effect-error")
	if err != nil {
		return err
	}
	failed, err := evaly.Run(ctx, effect)
	if err != nil {
		return err
	}
	audit, _, _, _, _ := store.Snapshot()
	fmt.Printf(
		"after effect: %s, usage %.0f, external effects %d, retained events %d\n",
		failed.Record().Trials[0].Status,
		failed.Record().Trials[0].TargetUsage.Units,
		len(audit),
		len(failed.Record().Trials[0].Evidence.Events),
	)
	return nil
}

func artifactExample(
	ctx context.Context,
	c evaly.Experiment,
	candidateID string,
	saved evaly.SavedView[fixtures.WorkflowInput, fixtures.WorkflowOutput, int],
) error {
	directory, err := os.MkdirTemp("", "evaly-workflow-")
	if err != nil {
		return err
	}
	defer func() {
		if cleanupErr := os.RemoveAll(directory); cleanupErr != nil {
			log.Print(cleanupErr)
		}
	}()
	artifacts, err := evaly.OpenFileStore(directory)
	if err != nil {
		return err
	}
	if err = evaly.SaveExperiment(ctx, artifacts, c); err != nil {
		return err
	}
	if err = evaly.SaveSavedView(ctx, artifacts, "workflow-view", saved); err != nil {
		return err
	}
	reopened, err := evaly.OpenFileStore(directory)
	if err != nil {
		return err
	}
	restored, err := evaly.LoadExperiment(ctx, reopened, candidateID)
	if err != nil {
		return err
	}
	saved, err = evaly.LoadSavedView(
		ctx,
		reopened,
		"workflow-view",
		fixtures.WorkflowInputCodec(),
		fixtures.WorkflowOutputCodec(),
		fixtures.WorkflowReferenceCodec(),
	)
	if err != nil {
		return err
	}
	first, err := evaly.Rescore(
		ctx,
		saved,
		fixtures.WorkflowGraders("external-balance-v1"),
		restored.Revision(),
		"",
		"rescore",
	)
	if err != nil {
		return err
	}
	second, err := evaly.Rescore(
		ctx,
		saved,
		fixtures.WorkflowGraders("external-balance-v2"),
		restored.Revision(),
		first.Revision,
		"rescore",
	)
	if err != nil {
		return err
	}
	fmt.Printf(
		"offline rescore: new lineage %t; source unchanged %t\n",
		second.Parent == first.Revision && second.Revision != first.Revision,
		restored.Revision() == c.Revision(),
	)
	return onlineExample(ctx, second, saved)
}

func onlineExample(
	ctx context.Context,
	second evaly.Assessment,
	saved evaly.SavedView[fixtures.WorkflowInput, fixtures.WorkflowOutput, int],
) error {
	budget, err := evaly.NewMemoryBudget(1)
	if err != nil {
		return err
	}
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
	if err != nil {
		return err
	}
	defer func() {
		if cleanupErr := worker.Cancel(context.WithoutCancel(ctx)); cleanupErr != nil {
			log.Print(cleanupErr)
		}
	}()
	obs, err := observation.New(
		"workflow-observation",
		second.Revision,
		observation.Sampling{
			Rule:         "all",
			Population:   "offline-fixture",
			Window:       "example",
			Reason:       "",
			Probability:  nil,
			OutcomeDelay: 0,
		},
		saved,
	)
	if err != nil {
		return err
	}
	future, status := worker.Enqueue(obs)
	if status != "accepted" {
		return fmt.Errorf("enqueue: %s", status)
	}
	result := <-future.Result
	fmt.Printf(
		"online assessment: %s, paid grades %d, skipped %d\n",
		result.Assessment.State,
		len(result.Assessment.Grades),
		len(result.Assessment.Skipped),
	)
	fmt.Println("live host agent/judge check: not performed; no credentials or paid calls")
	return nil
}
