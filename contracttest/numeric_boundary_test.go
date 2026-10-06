package contracttest

import (
	"encoding/json"
	"math/big"
	"testing"
)

func TestSchemaBoundaryMutationsUseExactTokens(t *testing.T) {
	for _, boundary := range []string{"9223372036854775807", "-9223372036854775808", "18446744073709551615", "1.7976931348623157e308", "-1.7976931348623157e308", "0.1", "1e-324", "0"} {
		t.Run(boundary, func(t *testing.T) {
			// Arrange: synthetic schemas include boundaries where float64 ±1 is unchanged.
			schema := map[string]any{"minimum": json.Number(boundary), "maximum": json.Number(boundary)}
			// Act: use the actual shared corpus generator.
			mutations := schemaBoundMutations(schema, nil)
			// Assert: each mutation is a numeric token exactly outside its lexical bound.
			if len(mutations) != 2 {
				t.Fatal(mutations)
			}
			original, ok := new(big.Rat).SetString(boundary)
			if !ok {
				t.Fatal(boundary)
			}
			for _, mutation := range mutations {
				verifyExactBoundaryMutation(t, mutation, original)
			}
		})
	}
}

func verifyExactBoundaryMutation(t *testing.T, mutation mutation, original *big.Rat) {
	t.Helper()
	token, ok := mutation.value.(json.Number)
	if !ok {
		t.Fatalf("boundary mutation lost its token: %T", mutation.value)
	}
	shifted, ok := new(big.Rat).SetString(token.String())
	if !ok {
		t.Fatal(token)
	}
	difference := new(big.Rat).Sub(shifted, original)
	want := new(big.Rat).SetInt64(1)
	if mutation.label == "minimum" {
		want.Neg(want)
	}
	if difference.Cmp(want) != 0 {
		t.Fatal(mutation.label, token, difference)
	}
}
