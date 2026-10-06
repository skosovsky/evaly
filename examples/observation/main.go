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
	c, e := fixtures.CalculationConfig("observation-fixture", "good", "")
	if e != nil {
		return e
	}
	cases, e := c.Dataset.Cases()
	if e != nil {
		return e
	}
	capture, e := evaly.NewCapture(c.Capture)
	if e != nil {
		return e
	}
	saved, e := evaly.SaveView(
		evaly.View[fixtures.Calculation, fixtures.CalculationOutput, int]{
			Case:     cases[0],
			Output:   fixtures.CalculationOutput{Sum: expectedSum},
			Evidence: capture.Seal(),
		},
		c.ProjectionRevision,
		fixtures.InputCodec(),
		fixtures.OutputCodec(),
		fixtures.ReferenceCodec(),
	)
	if e != nil {
		return e
	}
	directory, e := os.MkdirTemp("", "evaly-rescore-")
	if e != nil {
		return e
	}
	defer func() {
		if cleanupErr := os.RemoveAll(directory); cleanupErr != nil {
			log.Print(cleanupErr)
		}
	}()
	store, e := evaly.OpenFileStore(directory)
	if e != nil {
		return e
	}
	if e = evaly.SaveSavedView(context.Background(), store, "saved", saved); e != nil {
		return e
	}
	reopened, e := evaly.OpenFileStore(directory)
	if e != nil {
		return e
	}
	saved, e = evaly.LoadSavedView(
		context.Background(),
		reopened,
		"saved",
		fixtures.InputCodec(),
		fixtures.OutputCodec(),
		fixtures.ReferenceCodec(),
	)
	if e != nil {
		return e
	}
	o, e := observation.New(
		"already-performed",
		"",
		observation.Sampling{
			Rule:         "all",
			Reason:       "fixture",
			Population:   "staging",
			Window:       "staging-window",
			Probability:  nil,
			OutcomeDelay: 0,
		},
		saved,
	)
	if e != nil {
		return e
	}
	worker, e := observation.Start(
		context.Background(),
		observation.Config[fixtures.Calculation, fixtures.CalculationOutput, int]{
			Capacity:    1,
			Concurrency: 1,
			Deadline:    time.Second,
			Clock:       observation.RealClock{},
			Graders:     c.Graders, Budget: nil, GraderUnits: 0,
		},
	)
	if e != nil {
		return e
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	defer func() {
		if err := worker.Cancel(context.WithoutCancel(ctx)); err != nil {
			log.Print(err)
		}
	}()
	future, status := worker.Enqueue(o)
	fmt.Println("enqueue:", status)
	if status == "accepted" {
		result := <-future.Result
		fmt.Println("observation:", result.State, result.Assessment.Grades[0].Status)
	}
	return nil
}
