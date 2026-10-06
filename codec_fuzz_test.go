package evaly_test

import (
	"bytes"
	"testing"

	"github.com/skosovsky/evaly"
)

func FuzzJSONCodecCanonicalRoundtrip(f *testing.F) {
	f.Add([]byte(`{"id":9007199254740993,"nested":[1.0,1e0]}`))
	f.Add([]byte(`9223372036854775807`))
	f.Add([]byte(`{"duplicate":1,"duplicate":2}`))
	f.Fuzz(func(t *testing.T, raw []byte) {
		// Arrange: only valid canonical JSON participates in the roundtrip.
		expected, err := evaly.CanonicalJSON(raw)
		if err != nil {
			t.Skip()
		}
		codec := evaly.JSONCodec[any]{ID: "fuzz", Version: "1"}
		// Act
		decoded, err := codec.Decode(expected)
		if err != nil {
			t.Fatal(err)
		}
		actual, err := codec.Encode(decoded)
		// Assert
		if err != nil || !bytes.Equal(expected, actual) {
			t.Fatal(err, string(expected), string(actual))
		}
	})
}
