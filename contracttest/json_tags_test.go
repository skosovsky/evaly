package contracttest

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"
)

func TestExplicitFieldNamesPreserveSerialization(t *testing.T) {
	// Arrange: all sixteen genuine service formats, including populated and absent collections.
	values := generatedWireValues(t)
	if len(values) != 16 {
		t.Fatal("incomplete wire inventory", len(values))
	}
	for name, value := range values {
		t.Run(name, func(t *testing.T) {
			// Act: remove only tags equal to Go's default exported field name.
			current, err := json.Marshal(value)
			if err != nil {
				t.Fatal(err)
			}
			legacy, err := json.Marshal(legacyJSONValue(reflect.ValueOf(value)).Interface())
			if err != nil {
				t.Fatal(err)
			}
			// Assert: names, order, presence, raw payloads and canonical revisions survive.
			if !bytes.Equal(current, legacy) {
				t.Fatalf("explicit tags changed bytes\ncurrent: %s\nlegacy: %s", current, legacy)
			}
		})
	}
}

func legacyJSONType(original reflect.Type) reflect.Type {
	if original.Implements(reflect.TypeFor[json.Marshaler]()) {
		return original
	}
	switch {
	case original.Kind() == reflect.Pointer:
		return reflect.PointerTo(legacyJSONType(original.Elem()))
	case original.Kind() == reflect.Slice:
		return reflect.SliceOf(legacyJSONType(original.Elem()))
	case original.Kind() == reflect.Map:
		return reflect.MapOf(original.Key(), legacyJSONType(original.Elem()))
	case original.Kind() == reflect.Struct:
		fields := make([]reflect.StructField, 0, original.NumField())
		for field := range original.Fields() {
			if !field.IsExported() {
				continue
			}
			if field.Tag.Get("json") == field.Name {
				field.Tag = ""
			}
			field.Type = legacyJSONType(field.Type)
			fields = append(fields, field)
		}
		return reflect.StructOf(fields)
	default:
		return original
	}
}

func legacyJSONValue(original reflect.Value) reflect.Value {
	typ := legacyJSONType(original.Type())
	if typ == original.Type() {
		return original
	}
	out := reflect.New(typ).Elem()
	switch {
	case original.Kind() == reflect.Pointer:
		if !original.IsNil() {
			out.Set(reflect.New(typ.Elem()))
			out.Elem().Set(legacyJSONValue(original.Elem()))
		}
	case original.Kind() == reflect.Slice:
		if !original.IsNil() {
			out.Set(reflect.MakeSlice(typ, original.Len(), original.Len()))
		}
		for index := range original.Len() {
			out.Index(index).Set(legacyJSONValue(original.Index(index)))
		}
	case original.Kind() == reflect.Map:
		if !original.IsNil() {
			out.Set(reflect.MakeMapWithSize(typ, original.Len()))
		}
		iterator := original.MapRange()
		for iterator.Next() {
			out.SetMapIndex(iterator.Key(), legacyJSONValue(iterator.Value()))
		}
	case original.Kind() == reflect.Struct:
		for index := range out.NumField() {
			name := typ.Field(index).Name
			out.Field(index).Set(legacyJSONValue(original.FieldByName(name)))
		}
	}
	return out
}
