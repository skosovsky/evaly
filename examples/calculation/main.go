package main

import (
	"context"
	"fmt"
	"log"

	"github.com/skosovsky/evaly"
	"github.com/skosovsky/evaly/internal/fixtures"
)

func main() {
	ctx := context.Background()
	b, e := fixtures.CalculationConfig("baseline", "good", "")
	if e != nil {
		log.Fatal(e)
	}
	c, e := fixtures.CalculationConfig("candidate", "bad", "")
	if e != nil {
		log.Fatal(e)
	}
	baseline, candidate, e := evaly.RunPaired(ctx, b, c, "comparison-1")
	if e != nil {
		log.Fatal(e)
	}
	comparison, e := evaly.Compare(baseline, candidate, fixtures.Objective(), fixtures.Gate())
	if e != nil {
		log.Fatal(e)
	}
	fmt.Print(evaly.Report(comparison))
}
