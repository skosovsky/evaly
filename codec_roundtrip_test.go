package evaly_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"math"
	"reflect"
	"sync"
	"testing"

	"github.com/skosovsky/evaly"
)

func checkJSONRoundtrip[T any](t *testing.T, value T) {
	t.Helper()
	// Arrange
	codec := evaly.JSONCodec[T]{ID: "roundtrip", Version: "1"}
	refCodec := evaly.JSONCodec[int]{ID: "reference", Version: "1"}
	original, err := codec.Encode(value)
	if err != nil {
		t.Fatal(err)
	}
	// Act
	snapshot, err := evaly.SealSnapshot(value, codec)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := snapshot.Value()
	if err != nil {
		t.Fatal(err)
	}
	again, err := codec.Encode(decoded)
	if err != nil {
		t.Fatal(err)
	}
	draft := evaly.DatasetDraft[T, int]{Selection: "all", Cases: []evaly.Case[T, int]{{ID: "case", Input: value}}}
	dataset, err := draft.Seal(codec, refCodec)
	if err != nil {
		t.Fatal(err)
	}
	restored, err := evaly.RestoreDataset(dataset.Record(), codec, refCodec)
	// Assert
	if err != nil || !bytes.Equal(original, again) || !reflect.DeepEqual(dataset.Record(), restored.Record()) {
		t.Fatal(err, string(original), string(again))
	}
	resealed, err := evaly.SealSnapshot(decoded, codec)
	if err != nil || resealed.Revision() != snapshot.Revision() {
		t.Fatal(err, resealed.Revision(), snapshot.Revision())
	}
}

func TestJSONCodecExactGenericNumbers(t *testing.T) {
	for _, number := range []json.Number{"9007199254740993", "9223372036854775807", "1.0", "1e0", "1E+12", "1e400"} {
		t.Run(string(number), func(t *testing.T) {
			value := map[string]any{"id": number, "nested": []any{map[string]any{"number": number}}}
			checkJSONRoundtrip(t, value)
			codec := evaly.JSONCodec[map[string]any]{ID: "map", Version: "1"}
			// Act
			raw, err := codec.Encode(value)
			if err != nil {
				t.Fatal(err)
			}
			decoded, err := codec.Decode(raw)
			// Assert: generic numbers have explicit type and unchanged spelling.
			if err != nil || decoded["id"] != number {
				t.Fatal(err, decoded)
			}
		})
	}
	checkJSONRoundtrip(t, map[string]any{"id": int64(9007199254740993), "max": int64(math.MaxInt64)})
	checkJSONRoundtrip(t, struct{ N int64 }{N: math.MaxInt64})
}

func TestJSONCodecNumericLexicalIdentityAndTypedOverflow(t *testing.T) {
	// Arrange
	codec := evaly.JSONCodec[any]{ID: "number", Version: "1"}
	first, err := evaly.SealSnapshot[any](json.Number("1"), codec)
	if err != nil {
		t.Fatal(err)
	}
	// Act
	second, err := evaly.SealSnapshot[any](json.Number("1.0"), codec)
	_, overflow := (evaly.JSONCodec[int64]{ID: "int64", Version: "1"}).Decode([]byte("9223372036854775808"))
	// Assert
	if err != nil || first.Revision() == second.Revision() || !errors.Is(overflow, evaly.ErrInvalid) {
		t.Fatal(err, overflow)
	}
}

func TestJSONCodecAcyclicAliases(t *testing.T) {
	// Arrange
	type value struct {
		N int
		P *int
	}
	interior := &value{N: 1}
	interior.P = &interior.N
	slice := make([]any, 1)
	slice[0] = slice[:0]
	shared := map[string]any{"name": "same"}
	// Act / Assert
	checkJSONRoundtrip(t, interior)
	checkJSONRoundtrip(t, slice)
	checkJSONRoundtrip(t, map[string]any{"left": shared, "right": shared})
}

func TestJSONCodecActualCycles(t *testing.T) {
	// Arrange
	type node struct{ Next *node }
	pointer := &node{}
	pointer.Next = pointer
	object := map[string]any{}
	object["self"] = object
	slice := make([]any, 1)
	slice[0] = slice
	codec := evaly.JSONCodec[any]{ID: "cycle", Version: "1"}
	for _, value := range []any{pointer, object, slice} {
		// Act
		_, err := codec.Encode(value)
		// Assert
		if !errors.Is(err, evaly.ErrInvalid) {
			t.Fatal(err)
		}
	}
}

func TestJSONCodecIgnoredFieldsAndUTF8(t *testing.T) {
	// Arrange
	type payload struct {
		Value   string         `json:"value"`
		Cache   map[string]any `json:"-"`
		Invalid string         `json:"-"`
	}
	cache := map[string]any{}
	cache["self"] = cache
	valid := payload{Value: "hello", Cache: cache, Invalid: string([]byte{0xff})}
	// Act / Assert
	checkJSONRoundtrip(t, valid)
	codec := evaly.JSONCodec[any]{ID: "utf8", Version: "1"}
	for _, value := range []any{string([]byte{0xff}), map[string]string{string([]byte{0xff}): "value"}, map[string]string{"key": string([]byte{0xff})}} {
		_, err := codec.Encode(value)
		if !errors.Is(err, evaly.ErrInvalid) {
			t.Fatal(err)
		}
	}
}

type nonReversibleJSON struct{ N int }

func (v nonReversibleJSON) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]int{"renamed": v.N})
}

func TestJSONCodecRejectsNonReversibleCustomEncoding(t *testing.T) {
	// Arrange
	codec := evaly.JSONCodec[nonReversibleJSON]{ID: "custom", Version: "1"}
	// Act
	_, err := evaly.SealSnapshot(nonReversibleJSON{N: 1}, codec)
	// Assert
	if !errors.Is(err, evaly.ErrInvalid) {
		t.Fatal(err)
	}
}

func TestJSONCodecOwnedBuffersAndConcurrentDecode(t *testing.T) {
	// Arrange
	codec := evaly.JSONCodec[map[string]any]{ID: "owned", Version: "1"}
	source := map[string]any{"nested": map[string]any{"id": json.Number("9007199254740993")}}
	original, err := codec.Encode(source)
	if err != nil {
		t.Fatal(err)
	}
	kept := bytes.Clone(original)
	var wg sync.WaitGroup
	// Act
	for range 16 {
		wg.Go(func() {
			value, decode := codec.Decode(original)
			if decode != nil {
				t.Error(decode)
				return
			}
			value["nested"].(map[string]any)["id"] = json.Number("0")
			if _, encode := codec.Encode(value); encode != nil {
				t.Error(encode)
			}
		})
	}
	wg.Wait()
	// Assert: outputs do not alias source bytes or subsequent Encode calls.
	if !bytes.Equal(original, kept) {
		t.Fatal(string(original), string(kept))
	}
	decoded, err := codec.Decode(original)
	if err != nil || decoded["nested"].(map[string]any)["id"] != json.Number("9007199254740993") {
		t.Fatal(err, decoded)
	}
}

func TestJSONCodecAnonymousExportedChildrenUTF8(t *testing.T) {
	// Arrange: encoding/json serializes exported children of an unexported embed.
	type embedded struct{ Value string }
	type outer struct{ embedded }
	invalid := outer{Value: string([]byte{0xff})}
	codec := evaly.JSONCodec[outer]{ID: "embed", Version: "1"}
	// Act
	_, err := codec.Encode(invalid)
	// Assert
	if !errors.Is(err, evaly.ErrInvalid) {
		t.Fatal(err)
	}
	checkJSONRoundtrip(t, outer{Value: "valid"})
}

type marshaledStringKey string

func (marshaledStringKey) MarshalJSON() ([]byte, error) {
	return []byte(`"custom"`), nil
}

func TestJSONCodecStringMapKeyCannotBypassUTF8(t *testing.T) {
	// Arrange: map key encoding ignores its MarshalJSON method.
	value := map[marshaledStringKey]string{marshaledStringKey(string([]byte{0xff})): "value"}
	codec := evaly.JSONCodec[map[marshaledStringKey]string]{ID: "keys", Version: "1"}
	// Act
	_, err := codec.Encode(value)
	// Assert
	if !errors.Is(err, evaly.ErrInvalid) {
		t.Fatal(err)
	}
}

type pointerMarshaledString string

func (*pointerMarshaledString) MarshalJSON() ([]byte, error) {
	return []byte(`"custom"`), nil
}

func (v *pointerMarshaledString) UnmarshalJSON([]byte) error {
	*v = pointerMarshaledString(string([]byte{0xff}))
	return nil
}

func TestJSONCodecAddressablePointerMarshalerOwnsSourceValidation(t *testing.T) {
	// Arrange: the field is addressable when encoded via the outer pointer.
	type outer struct{ Value pointerMarshaledString }
	value := &outer{Value: pointerMarshaledString(string([]byte{0xff}))}
	// Act / Assert: stable custom output is reversible despite opaque source bytes.
	checkJSONRoundtrip(t, value)
}
