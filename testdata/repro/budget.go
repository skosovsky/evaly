// Baseline behavioral repro; deliberately asserts the reviewed bad behavior.
package main

import (
	"context"
	"fmt"

	"github.com/skosovsky/evaly"
)

func main() {
	ctx := context.Background()
	b, err := evaly.NewMemoryBudget(1)
	must(err)
	r, err := b.Reserve(ctx, "full", 1)
	must(err)
	must(b.Claim(ctx, r))
	_, err = b.Reserve(ctx, "overflow", 1e-17)
	must(err) // baseline wrongly admits this positive liability
	if b.Used() != 1 {
		panic("unexpected baseline exhausted-budget result")
	}
	b, err = evaly.NewMemoryBudget(1e16)
	must(err)
	r, err = b.Reserve(ctx, "large", 1e16)
	must(err)
	must(b.Claim(ctx, r))
	must(b.Reconcile(ctx, r, evaly.Usage{Known: true, Units: 1}))
	if b.Used() != 0 {
		panic("unexpected baseline actual-usage result")
	}
	fmt.Println("F03 reproduced: admitted overspend; known actual liability lost")
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
