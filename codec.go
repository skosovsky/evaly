package evaly

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"unicode/utf8"
)

// Codec explicitly owns domain serialization. Encodings must be canonical JSON.
type Codec[T any] interface {
	Identity() CodecIdentity
	Encode(T) ([]byte, error)
	Decode([]byte) (T, error)
}

type CodecIdentity struct {
	ID      string `json:"id"`
	Version string `json:"version"`
}

// JSONCodec supports finite, acyclic JSON values. Object keys are sorted and
// duplicate keys, invalid UTF-8 and trailing data are rejected.
type JSONCodec[T any] struct {
	ID      string `json:"ID"`
	Version string `json:"Version"`
}

func (c JSONCodec[T]) Identity() CodecIdentity { return CodecIdentity(c) }
func (c JSONCodec[T]) Encode(v T) ([]byte, error) {
	if c.ID == "" || c.Version == "" {
		return nil, ErrInvalid
	}
	if err := validateJSONValue(reflect.ValueOf(v), map[uintptr]bool{}); err != nil {
		return nil, err
	}
	b, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("%w: encode: %w", ErrInvalid, err)
	}
	return CanonicalJSON(b)
}
func (c JSONCodec[T]) Decode(b []byte) (T, error) {
	var v T
	if _, err := CanonicalJSON(b); err != nil {
		return v, err
	}
	err := json.Unmarshal(b, &v)
	if err != nil {
		return v, fmt.Errorf("%w: decode: %w", ErrInvalid, err)
	}
	return v, nil
}

// CanonicalJSON preserves number lexemes while sorting all object keys.
func CanonicalJSON(b []byte) ([]byte, error) {
	if !utf8.Valid(b) {
		return nil, fmt.Errorf("%w: invalid UTF-8", ErrInvalid)
	}
	d := json.NewDecoder(bytes.NewReader(b))
	d.UseNumber()
	v, err := readJSON(d)
	if err != nil {
		return nil, fmt.Errorf("%w: JSON: %w", ErrInvalid, err)
	}
	if _, err = d.Token(); !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("%w: trailing JSON", ErrInvalid)
	}
	return json.Marshal(v)
}
func readJSON(d *json.Decoder) (any, error) {
	tok, err := d.Token()
	if err != nil {
		return nil, err
	}
	delim, ok := tok.(json.Delim)
	if !ok {
		return tok, nil
	}
	switch delim {
	case '{':
		return readJSONObject(d)
	case '[':
		return readJSONArray(d)
	default:
		return nil, ErrInvalid
	}
}
func digest(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }
func canonical(v any) ([]byte, error) {
	b, e := json.Marshal(v)
	if e != nil {
		return nil, e
	}
	return CanonicalJSON(b)
}
func cloneJSON[T any](v T) (T, error) {
	var out T
	b, e := json.Marshal(v)
	if e != nil {
		return out, e
	}
	e = json.Unmarshal(b, &out)
	return out, e
}
func identityPart(n int) string { return strconv.Itoa(n) }

// Validation is part of the explicit stock codec, never implicit store decoding.
func validateJSONValue(v reflect.Value, visited map[uintptr]bool) error {
	if !v.IsValid() {
		return nil
	}
	if v.CanInterface() {
		if _, ok := reflect.TypeAssert[json.Marshaler](v); ok {
			return nil
		}
	}
	switch v.Kind() {
	case reflect.String:
		if !utf8.ValidString(v.String()) {
			return ErrInvalid
		}
	case reflect.Map:
		return validateJSONMap(v, visited)
	case reflect.Pointer, reflect.Interface:
		return validateJSONPointer(v, visited)
	case reflect.Slice, reflect.Array:
		return validateJSONSequence(v, visited)
	case reflect.Struct:
		return validateJSONStruct(v, visited)
	case reflect.Invalid,
		reflect.Bool,
		reflect.Int,
		reflect.Int8,
		reflect.Int16,
		reflect.Int32,
		reflect.Int64,
		reflect.Uint,
		reflect.Uint8,
		reflect.Uint16,
		reflect.Uint32,
		reflect.Uint64,
		reflect.Uintptr,
		reflect.Float32,
		reflect.Float64,
		reflect.Complex64,
		reflect.Complex128,
		reflect.Chan,
		reflect.Func,
		reflect.UnsafePointer:
	}
	return nil
}

func readJSONObject(d *json.Decoder) (any, error) {
	m := map[string]any{}
	for d.More() {
		k, e := d.Token()
		if e != nil {
			return nil, e
		}
		key, ok := k.(string)
		if !ok {
			return nil, ErrInvalid
		}
		if _, ok = m[key]; ok {
			return nil, fmt.Errorf("duplicate key %q", key)
		}
		v, e := readJSON(d)
		if e != nil {
			return nil, e
		}
		m[key] = v
	}
	closing, e := d.Token()
	if e != nil || closing != json.Delim('}') {
		return nil, ErrInvalid
	}
	return m, nil
}

func readJSONArray(d *json.Decoder) (any, error) {
	a := []any{}
	for d.More() {
		v, e := readJSON(d)
		if e != nil {
			return nil, e
		}
		a = append(a, v)
	}
	closing, e := d.Token()
	if e != nil || closing != json.Delim(']') {
		return nil, ErrInvalid
	}
	return a, nil
}

func validateJSONMap(v reflect.Value, visited map[uintptr]bool) error {
	if v.Type().Key().Kind() != reflect.String {
		return ErrInvalid
	}
	if v.IsNil() {
		return nil
	}
	ptr := v.Pointer()
	if visited[ptr] {
		return ErrInvalid
	}
	visited[ptr] = true
	defer delete(visited, ptr)
	iter := v.MapRange()
	for iter.Next() {
		if err := validateJSONValue(iter.Key(), visited); err != nil {
			return err
		}
		if err := validateJSONValue(iter.Value(), visited); err != nil {
			return err
		}
	}
	return nil
}

func validateJSONPointer(v reflect.Value, visited map[uintptr]bool) error {
	if v.IsNil() {
		return nil
	}
	if v.Kind() == reflect.Pointer {
		ptr := v.Pointer()
		if visited[ptr] {
			return ErrInvalid
		}
		visited[ptr] = true
		defer delete(visited, ptr)
	}
	return validateJSONValue(v.Elem(), visited)
}

func validateJSONSequence(v reflect.Value, visited map[uintptr]bool) error {
	if v.Kind() == reflect.Slice && !v.IsNil() {
		ptr := v.Pointer()
		if visited[ptr] {
			return ErrInvalid
		}
		visited[ptr] = true
		defer delete(visited, ptr)
	}
	for i := range v.Len() {
		if err := validateJSONValue(v.Index(i), visited); err != nil {
			return err
		}
	}
	return nil
}

func validateJSONStruct(v reflect.Value, visited map[uintptr]bool) error {
	for i := range v.NumField() {
		if v.Type().Field(i).IsExported() {
			if err := validateJSONValue(v.Field(i), visited); err != nil {
				return err
			}
		}
	}
	return nil
}
