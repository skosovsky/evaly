// Baseline behavioral repro; assertions deliberately describe reviewed defects.
package main

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/skosovsky/evaly"
)

func main() {
	codec := evaly.JSONCodec[map[string]any]{ID: "map", Version: "1"}
	value := map[string]any{"id": int64(9007199254740993)}
	snapshot, err := evaly.SealSnapshot(value, codec)
	codecMust(err)
	decoded, err := snapshot.Value()
	codecMust(err)
	if decoded["id"] != float64(9007199254740992) {
		panic("baseline large-number loss not reproduced")
	}
	reference := evaly.JSONCodec[int]{ID: "int", Version: "1"}
	dataset, err := (evaly.DatasetDraft[map[string]any, int]{Selection: "all", Cases: []evaly.Case[map[string]any, int]{{ID: "case", Input: value}}}).Seal(codec, reference)
	codecMust(err)
	_, err = evaly.RestoreDataset(dataset.Record(), codec, reference)
	if !errors.Is(err, evaly.ErrCorrupt) {
		panic("baseline own-dataset restore corruption not reproduced")
	}
	type interior struct {
		N int
		P *int
	}
	pointer := &interior{N: 1}
	pointer.P = &pointer.N
	raw, err := json.Marshal(pointer)
	codecMust(err)
	if string(raw) != `{"N":1,"P":1}` {
		panic("bad interior fixture")
	}
	_, err = (evaly.JSONCodec[*interior]{ID: "p", Version: "1"}).Encode(pointer)
	if !errors.Is(err, evaly.ErrInvalid) {
		panic("baseline interior alias rejection not reproduced")
	}
	slice := make([]any, 1)
	slice[0] = slice[:0]
	raw, err = json.Marshal(slice)
	codecMust(err)
	if string(raw) != `[[]]` {
		panic("bad slice fixture")
	}
	_, err = (evaly.JSONCodec[[]any]{ID: "s", Version: "1"}).Encode(slice)
	if !errors.Is(err, evaly.ErrInvalid) {
		panic("baseline slice alias rejection not reproduced")
	}
	fmt.Println("F04/F05 reproduced: numeric loss, own-record corruption, acyclic alias rejection")
}

func codecMust(err error) {
	if err != nil {
		panic(err)
	}
}
