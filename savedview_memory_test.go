package evaly

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"
)

func memorySavedView(t *testing.T) SavedView[any, any, any] {
	t.Helper()
	capture, err := NewCapture(CaptureConfig{
		Policy: FieldPolicy{
			ID:             "safe",
			Allowed:        map[string][]string{"tool": {"nested", "number"}},
			KeepReferences: true,
		},
		KnownKinds: []string{"tool"},
		MaxEvents:  2,
		MaxBytes:   4096,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err = capture.Record(context.Background(), Event{
		Version: 1, Sequence: 1, Kind: "tool", Payload: json.RawMessage(`{"nested":{"items":[1,2]},"number":1e+02}`),
		References: []string{"https://example.com/evidence"},
	}); err != nil {
		t.Fatal(err)
	}
	var reference any = map[string]any{"items": []any{json.Number("-0")}}
	codec := JSONCodec[any]{ID: "any", Version: "1"}
	saved, err := SaveView(View[any, any, any]{
		Case: Case[any, any]{
			ID:        "case",
			Revision:  "revision",
			Input:     map[string]any{"items": []any{json.Number("1e+02")}},
			Reference: &reference,
			Metadata: map[string]string{
				"label": "original",
			},
			RequiredEvidence: []string{"tool"},
			Generation:       &Generation{Generator: "original"},
		},
		Output: map[string]any{"items": []any{json.Number("1.00")}}, Evidence: capture.Seal(),
	}, "permitted", codec, codec, codec)
	if err != nil {
		t.Fatal(err)
	}
	return saved
}

func TestSavedViewOwnedCopiesAndNumberLexemes(t *testing.T) {
	// Arrange.
	saved := memorySavedView(t)
	original := saved.Record()
	view, err := saved.View()
	if err != nil {
		t.Fatal(err)
	}
	record := saved.Record()
	// Act: mutate every nested mutable field exposed by either accessor.
	view.Case.Input.(map[string]any)["items"].([]any)[0] = "changed"
	view.Output.(map[string]any)["items"].([]any)[0] = "changed"
	(*view.Case.Reference).(map[string]any)["items"].([]any)[0] = "changed"
	view.Case.Metadata["label"] = "changed"
	view.Case.RequiredEvidence[0] = "changed"
	view.Case.Generation.Generator = "changed"
	view.Evidence.Events[0].Payload[0] = '!'
	view.Evidence.Events[0].References[0] = "changed"
	view.Evidence.Coverage["tool"] = false
	record.Case.Input[0] = '!'
	record.Case.Reference[0] = '!'
	record.Output[0] = '!'
	record.Case.Metadata["label"] = "changed"
	record.Case.RequiredEvidence[0] = "changed"
	record.Case.Generation.Generator = "changed"
	record.Evidence.Events[0].Payload[0] = '!'
	record.Evidence.Events[0].References[0] = "changed"
	record.Evidence.Coverage["tool"] = false
	fresh, err := saved.View()
	// Assert.
	if err != nil || !reflect.DeepEqual(saved.Record(), original) {
		t.Fatalf("mutable access escaped into private state: %v", err)
	}
	for _, pair := range []struct {
		value  any
		lexeme string
	}{
		{fresh.Case.Input.(map[string]any)["items"].([]any)[0], "1e+02"},
		{fresh.Output.(map[string]any)["items"].([]any)[0], "1.00"},
		{(*fresh.Case.Reference).(map[string]any)["items"].([]any)[0], "-0"},
	} {
		if pair.value != json.Number(pair.lexeme) {
			t.Fatalf("number changed: %#v, want %s", pair.value, pair.lexeme)
		}
	}
	if string(fresh.Evidence.Events[0].Payload) != `{"nested":{"items":[1,2]},"number":1e+02}` {
		t.Fatalf("evidence numeric lexeme changed: %s", fresh.Evidence.Events[0].Payload)
	}
}

func TestSavedViewPrivateCopyMatchesJSONNormalization(t *testing.T) {
	for _, empty := range []bool{false, true} {
		t.Run(map[bool]string{false: "nil", true: "empty"}[empty], func(t *testing.T) {
			// Arrange: omitempty fields normalize to nil, required Events retains [].
			capture, err := NewCapture(CaptureConfig{Policy: FieldPolicy{ID: "safe"}, MaxEvents: 1, MaxBytes: 1024})
			if err != nil {
				t.Fatal(err)
			}
			view := View[int, int, int]{Case: Case[int, int]{ID: "case", Revision: "1"}, Evidence: capture.Seal()}
			if empty {
				view.Case.Metadata = map[string]string{}
				view.Case.RequiredEvidence = []string{}
			}
			codec := JSONCodec[int]{ID: "int", Version: "1"}
			saved, err := SaveView(view, "permitted", codec, codec, codec)
			if err != nil {
				t.Fatal(err)
			}
			want, err := cloneJSON(saved.record)
			if err != nil {
				t.Fatal(err)
			}
			// Act.
			got := saved.Record()
			fresh, err := saved.View()
			// Assert.
			if err != nil || !reflect.DeepEqual(got, want) || fresh.Case.Metadata != nil ||
				fresh.Case.RequiredEvidence != nil || fresh.Case.Reference != nil || fresh.Evidence.Events == nil {
				t.Fatalf("JSON normalization changed: got %#v want %#v, view %#v, error %v", got, want, fresh, err)
			}
		})
	}
}

type savedViewCountingCodec struct {
	JSONCodec[int]

	encodes, decodes *int
}

func (c savedViewCountingCodec) Encode(v int) ([]byte, error) {
	*c.encodes++
	return c.JSONCodec.Encode(v)
}
func (c savedViewCountingCodec) Decode(b []byte) (int, error) {
	*c.decodes++
	return c.JSONCodec.Decode(b)
}

func TestSavedViewCodecCallbackCounts(t *testing.T) {
	// Arrange.
	capture, err := NewCapture(CaptureConfig{Policy: FieldPolicy{ID: "safe"}, MaxEvents: 1, MaxBytes: 1024})
	if err != nil {
		t.Fatal(err)
	}
	encodes, decodes := 0, 0
	codec := savedViewCountingCodec{JSONCodec[int]{ID: "int", Version: "1"}, &encodes, &decodes}
	reference := 3
	saved, err := SaveView(View[int, int, int]{
		Case: Case[int, int]{
			ID:        "case",
			Revision:  "1",
			Input:     1,
			Reference: &reference,
		},
		Output:   2,
		Evidence: capture.Seal(),
	}, "permitted", codec, codec, codec)
	if err != nil {
		t.Fatal(err)
	}
	graders := []Grader[int, int, int]{GraderFunc[int, int, int]{
		Identity: GraderRevision{ID: "grader", Implementation: "1", Rubric: "1"},
		Evaluate: func(context.Context, View[int, int, int]) (Grade, error) {
			return Grade{Status: Scored, Assertions: []Assertion{{Name: "ok", Pass: true}}}, nil
		},
	}}
	// Act / Assert: externally restored codecs retain the full decode/re-encode check.
	encodes, decodes = 0, 0
	if _, err := RestoreSavedView(saved.Record(), codec, codec, codec); err != nil || encodes != 3 || decodes != 3 {
		t.Fatalf("restore callbacks: encode=%d decode=%d error=%v", encodes, decodes, err)
	}
	encodes, decodes = 0, 0
	if _, err := saved.View(); err != nil || encodes != 0 || decodes != 3 {
		t.Fatalf("view callbacks: encode=%d decode=%d error=%v", encodes, decodes, err)
	}
	encodes, decodes = 0, 0
	if _, err := Rescore(
		context.Background(),
		saved,
		graders,
		"source",
		"",
		"rescore",
	); err != nil || encodes != 3 ||
		decodes != 6 {
		t.Fatalf("rescore callbacks: encode=%d decode=%d error=%v", encodes, decodes, err)
	}
}

func TestSavedViewRestoreStillRejectsInvalidRecords(t *testing.T) {
	for _, malformed := range []string{`{"x":1,"x":2}`, `{} {}`, "\"\xff\"", `1e999999999999999999999`, `{"x":`} {
		t.Run(malformed, func(t *testing.T) {
			// Arrange.
			saved := memorySavedView(t)
			record := saved.Record()
			record.Case.Input = json.RawMessage(malformed)
			// Act.
			_, err := RestoreSavedView(record, saved.input, saved.output, saved.reference)
			// Assert.
			if err == nil {
				t.Fatal("invalid external saved projection accepted")
			}
		})
	}
	// Arrange.
	saved := memorySavedView(t)
	record := saved.Record()
	record.Evidence.Revision = "tampered"
	// Act.
	_, err := RestoreSavedView(record, saved.input, saved.output, saved.reference)
	// Assert.
	if !errors.Is(err, ErrCorrupt) {
		t.Fatalf("tampered evidence accepted: %v", err)
	}
}

func TestZeroSavedViewRecordPreservesJSONNormalization(t *testing.T) {
	// Arrange.
	var saved SavedView[int, int, int]
	want, err := cloneJSON(saved.record)
	if err != nil {
		t.Fatal(err)
	}
	// Act.
	got := saved.Record()
	// Assert.
	if !reflect.DeepEqual(got, want) || string(got.Case.Input) != "null" || string(got.Output) != "null" {
		t.Fatalf("zero record normalization changed: got %#v want %#v", got, want)
	}
}

func TestSavedViewTrustedBuiltInDecodeParity(t *testing.T) {
	for _, raw := range []string{`null`, `{"nested":[1e+02,-0,1.00,true,"<html>",null]}`, `[]`, `{}`, `""`, `"\u2028"`} {
		t.Run(raw, func(t *testing.T) {
			// Arrange: private bytes have crossed canonical validation already.
			codec := JSONCodec[any]{ID: "any", Version: "1"}
			canonical, err := CanonicalJSON([]byte(raw))
			if err != nil {
				t.Fatal(err)
			}
			want, err := codec.Decode(canonical)
			if err != nil {
				t.Fatal(err)
			}
			// Act.
			got, valueErr := decodeSavedJSON[any](codec, canonical)
			pointerGot, pointerErr := decodeSavedJSON[any](&codec, canonical)
			// Assert.
			if valueErr != nil || pointerErr != nil || !reflect.DeepEqual(got, want) ||
				!reflect.DeepEqual(pointerGot, want) {
				t.Fatalf(
					"built-in trusted decode changed: value=%#v pointer=%#v want=%#v errors=%v/%v",
					got,
					pointerGot,
					want,
					valueErr,
					pointerErr,
				)
			}
		})
	}
}

func TestSavedViewTrustedBuiltInDecodeRetainsTypedOverflow(t *testing.T) {
	// Arrange: this is valid canonical JSON but cannot decode into the requested type.
	codec := JSONCodec[int8]{ID: "int8", Version: "1"}
	raw := []byte(`128`)
	_, want := codec.Decode(raw)
	// Act.
	_, got := decodeSavedJSON[int8](codec, raw)
	// Assert.
	if !errors.Is(want, ErrInvalid) || !errors.Is(got, ErrInvalid) || got.Error() != want.Error() {
		t.Fatalf("typed overflow behavior changed: got %v want %v", got, want)
	}
}
