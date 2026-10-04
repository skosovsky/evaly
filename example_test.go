package evaly_test

import (
	"context"
	"fmt"

	"github.com/skosovsky/evaly"
)

func ExampleDatasetDraft_Seal() {
	draft := evaly.DatasetDraft[int, int]{Selection: "all", Cases: []evaly.Case[int, int]{{ID: "one", Input: 1}}}
	codec := evaly.JSONCodec[int]{ID: "integer", Version: "1"}
	snapshot, err := draft.Seal(codec, codec)
	if err != nil {
		panic(err)
	}
	draft.Cases[0].Input = 99
	cases, err := snapshot.Cases()
	if err != nil {
		panic(err)
	}
	fmt.Println(cases[0].Input)
	// Output: 1
}
func ExampleMemoryBudget() {
	budget, err := evaly.NewMemoryBudget(1)
	if err != nil {
		panic(err)
	}
	reservation, err := budget.Reserve(context.Background(), "dispatch-1", 1)
	if err != nil {
		panic(err)
	}
	if err = budget.Reconcile(context.Background(), reservation, evaly.Usage{}); err != nil {
		panic(err)
	}
	_, err = budget.Reserve(context.Background(), "dispatch-2", 1)
	fmt.Println(err)
	// Output: evaly: budget exhausted
}
