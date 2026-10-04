package main

import (
	"context"
	"fmt"
	"log"

	"github.com/skosovsky/evaly"
	"github.com/skosovsky/evaly/internal/fixtures"
)

func main() {
	c, e := fixtures.CRMConfig("crm-example", false)
	if e != nil {
		log.Fatal(e)
	}
	experiment, e := evaly.Run(context.Background(), c)
	if e != nil {
		log.Fatal(e)
	}
	for _, trial := range experiment.Record().Trials {
		for _, grade := range trial.Grades {
			fmt.Printf("%s: %s %v\n", trial.ID, grade.Status, grade.Assertions)
		}
	}
}
