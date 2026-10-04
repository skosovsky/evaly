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
	c, e := fixtures.CalculationConfig("observation-fixture", "good", "")
	if e != nil {
		log.Fatal(e)
	}
	cases, e := c.Dataset.Cases()
	if e != nil {
		log.Fatal(e)
	}
	capture, e := evaly.NewCapture(c.Capture)
	if e != nil {
		log.Fatal(e)
	}
	saved, e := evaly.SaveView(
		evaly.View[fixtures.Calculation, fixtures.CalculationOutput, int]{
			Case:     cases[0],
			Output:   fixtures.CalculationOutput{Sum: 3},
			Evidence: capture.Seal(),
		},
		c.ProjectionRevision,
		fixtures.InputCodec(),
		fixtures.OutputCodec(),
		fixtures.ReferenceCodec(),
	)
	if e != nil {
		log.Fatal(e)
	}
	directory, e := os.MkdirTemp("", "evaly-rescore-")
	if e != nil {
		log.Fatal(e)
	}
	defer os.RemoveAll(directory)
	store, e := evaly.OpenFileStore(directory)
	if e != nil {
		log.Fatal(e)
	}
	if e = evaly.SaveSavedView(context.Background(), store, "saved", saved); e != nil {
		log.Fatal(e)
	}
	reopened, e := evaly.OpenFileStore(directory)
	if e != nil {
		log.Fatal(e)
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
		log.Fatal(e)
	}
	o, e := observation.New(
		"already-performed",
		"",
		observation.Sampling{Rule: "all", Reason: "fixture", Population: "staging", Window: "staging-window"},
		saved,
	)
	if e != nil {
		log.Fatal(e)
	}
	worker, e := observation.Start(
		context.Background(),
		observation.Config[fixtures.Calculation, fixtures.CalculationOutput, int]{
			Capacity:    1,
			Concurrency: 1,
			Deadline:    time.Second,
			Clock:       observation.RealClock{},
			Graders:     c.Graders,
		},
	)
	if e != nil {
		log.Fatal(e)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	defer func() {
		if err := worker.Cancel(ctx); err != nil {
			log.Print(err)
		}
	}()
	future, status := worker.Enqueue(o)
	fmt.Println("enqueue:", status)
	if status == "accepted" {
		result := <-future.Result
		fmt.Println("observation:", result.State, result.Assessment.Grades[0].Status)
	}
}
