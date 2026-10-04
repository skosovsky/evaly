package evaly

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	"github.com/skosovsky/evaly/internal/wirecontract"
)

// DecodeWire checks the published structural contract before decoding a service
// envelope. Call its typed Restore or Validate function for semantic invariants.
// Domain types must be decoded with their host-owned Codec instead.
func DecodeWire[T any](raw []byte) (T, error) {
	var result T
	t := reflect.TypeFor[T]()
	if !wirecontract.Supported(t) {
		return result, fmt.Errorf("%w: unsupported wire type", ErrUnsupported)
	}
	canonical, e := CanonicalJSON(raw)
	if e != nil {
		return result, fmt.Errorf("%w: wire JSON", ErrInvalid)
	}
	var value any
	d := json.NewDecoder(bytes.NewReader(canonical))
	d.UseNumber()
	if e = d.Decode(&value); e != nil {
		return result, fmt.Errorf("%w: wire JSON", ErrInvalid)
	}
	value, e = wirecontract.Validate(t, value)
	if e != nil {
		if errors.Is(e, wirecontract.ErrVersion) {
			return result, fmt.Errorf("%w: wire version", ErrUnsupported)
		}
		return result, fmt.Errorf("%w: wire structure", ErrInvalid)
	}
	normalized, e := json.Marshal(value)
	if e != nil {
		return result, fmt.Errorf("%w: wire structure", ErrInvalid)
	}
	if e = json.Unmarshal(normalized, &result); e != nil {
		return result, fmt.Errorf("%w: wire value", ErrInvalid)
	}
	return result, nil
}
