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
	"strings"
	"unicode/utf8"
)

// Codec explicitly owns domain serialization. Encodings must be canonical JSON.
// Encode returns caller-owned bytes that remain stable after return. Decode returns
// fresh owned mutable values and must not retain or mutate input bytes. Identity,
// validation and callback behavior must stay stable during use; shared codecs must
// be concurrency-safe. Evaly cannot enforce these promises on arbitrary host code.
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
// duplicate keys, invalid UTF-8 and trailing data are rejected. Generic numbers
// decode as [json.Number], preserving their lexemes; typed overflow is an error.
// Encode verifies canonical reversibility before success. Custom marshalers must
// have stable behavior and matching decode semantics; see docs/codecs.md for the
// conservative source-string validation boundary.
type JSONCodec[T any] struct {
	ID      string `json:"ID"`
	Version string `json:"Version"`
}

func (c JSONCodec[T]) Identity() CodecIdentity { return CodecIdentity(c) }
func (c JSONCodec[T]) Encode(v T) ([]byte, error) {
	if c.ID == "" || c.Version == "" {
		return nil, ErrInvalid
	}
	b, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("%w: encode: %w", ErrInvalid, err)
	}
	// The encoder owns JSON field selection and cycle semantics. Reflection below
	// only validates source strings, which encoding/json otherwise replaces.
	if err = validateJSONValue(reflect.ValueOf(v), map[jsonVisit]bool{}); err != nil {
		return nil, err
	}
	b, err = CanonicalJSON(b)
	if err != nil {
		return nil, err
	}
	decoded, err := c.Decode(b)
	if err != nil {
		return nil, err
	}
	roundtrip, err := json.Marshal(decoded)
	if err != nil {
		return nil, fmt.Errorf("%w: re-encode: %w", ErrInvalid, err)
	}
	roundtrip, err = CanonicalJSON(roundtrip)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(b, roundtrip) {
		return nil, fmt.Errorf("%w: JSON representation is not reversible", ErrInvalid)
	}
	return b, nil
}
func (c JSONCodec[T]) Decode(b []byte) (T, error) {
	var v T
	if _, err := CanonicalJSON(b); err != nil {
		return v, err
	}
	d := json.NewDecoder(bytes.NewReader(b))
	d.UseNumber()
	err := d.Decode(&v)
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

// jsonVisit bounds source-string traversal, not JSON cycle classification.
// Type distinguishes interior pointers; slice extent distinguishes overlapping views.
type jsonVisit struct {
	pointer uintptr
	typ     reflect.Type
	length  int
}

// Source strings are checked conservatively in exported fields except json:"-".
// Custom marshalers own their source validation; emitted JSON is checked separately.
func validateJSONValue(v reflect.Value, visited map[jsonVisit]bool) error {
	if !v.IsValid() {
		return nil
	}
	if v.CanInterface() {
		if _, ok := reflect.TypeAssert[json.Marshaler](v); ok {
			return nil
		}
	}
	if v.CanAddr() && v.Addr().CanInterface() {
		if _, ok := reflect.TypeAssert[json.Marshaler](v.Addr()); ok {
			return nil
		}
	}
	switch v.Kind() {
	case reflect.String:
		if !utf8.ValidString(v.String()) {
			return ErrInvalid
		}
	case reflect.Interface:
		if !v.IsNil() {
			return validateJSONValue(v.Elem(), visited)
		}
	case reflect.Map, reflect.Pointer, reflect.Slice:
		if v.IsNil() {
			return nil
		}
		key := jsonVisit{pointer: v.Pointer(), typ: v.Type(), length: 0}
		if v.Kind() == reflect.Slice {
			key.length = v.Len()
		}
		if visited[key] {
			return nil
		}
		visited[key] = true
		return validateJSONChildren(v, visited)
	case reflect.Array, reflect.Struct:
		return validateJSONChildren(v, visited)
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

func validateJSONChildren(v reflect.Value, visited map[jsonVisit]bool) error {
	switch v.Kind() {
	case reflect.Pointer:
		return validateJSONValue(v.Elem(), visited)
	case reflect.Map:
		return validateJSONMapStrings(v, visited)
	case reflect.Slice, reflect.Array:
		for i := range v.Len() {
			if err := validateJSONValue(v.Index(i), visited); err != nil {
				return err
			}
		}
	case reflect.Struct:
		return validateJSONStructStrings(v, visited)
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
		reflect.UnsafePointer,
		reflect.Interface,
		reflect.String:
	}
	return nil
}

func validateJSONMapStrings(v reflect.Value, visited map[jsonVisit]bool) error {
	if v.Type().Key().Kind() != reflect.String {
		return ErrInvalid
	}
	iter := v.MapRange()
	for iter.Next() {
		// String map keys bypass MarshalJSON in encoding/json.
		if !utf8.ValidString(iter.Key().String()) {
			return ErrInvalid
		}
		if err := validateJSONValue(iter.Value(), visited); err != nil {
			return err
		}
	}
	return nil
}

func validateJSONStructStrings(v reflect.Value, visited map[jsonVisit]bool) error {
	for i := range v.NumField() {
		field := v.Type().Field(i)
		fieldType := field.Type
		if fieldType.Kind() == reflect.Pointer {
			fieldType = fieldType.Elem()
		}
		visible := field.IsExported() || (field.Anonymous && fieldType.Kind() == reflect.Struct)
		if visible && strings.Split(field.Tag.Get("json"), ",")[0] != "-" {
			if err := validateJSONValue(v.Field(i), visited); err != nil {
				return err
			}
		}
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

// mustCloneJSON copies private JSON wire data. A failure is an internal invariant
// breach: constructors validate the representation before storing these records.
func mustCloneJSON[T any](v T) T {
	out, err := cloneJSON(v)
	if err != nil {
		panic(err)
	}
	return out
}
