package evaly

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

type captureMemoryPolicy struct {
	calls   int
	project func(Event) (Event, error)
}

func (*captureMemoryPolicy) Revision() string { return "capture-memory" }
func (p *captureMemoryPolicy) Project(_ context.Context, e Event) (Event, error) {
	p.calls++
	return p.project(e)
}

//nolint:gocognit // Table independently covers callback and error precedence boundaries.
func TestCaptureSaturatedCallbackErrorPrecedence(t *testing.T) {
	policyError := errors.New("policy failed")
	cases := []struct {
		name      string
		prepare   func(*Event)
		project   func(Event) (Event, error)
		cancelled bool
		want      error
		callbacks int
	}{
		{name: "capacity", project: func(e Event) (Event, error) { return e, nil }, want: ErrIncomplete, callbacks: 1},
		{
			name:      "policy-error",
			project:   func(Event) (Event, error) { return Event{}, policyError },
			want:      policyError,
			callbacks: 1,
		},
		{
			name:      "metadata",
			project:   func(e Event) (Event, error) { e.Sequence++; return e, nil },
			want:      ErrInvalid,
			callbacks: 1,
		},
		{
			name:      "duplicate-output",
			project:   func(e Event) (Event, error) { e.Payload = json.RawMessage(`{"a":1,"a":2}`); return e, nil },
			want:      ErrInvalid,
			callbacks: 1,
		},
		{
			name:      "unsafe-reference",
			project:   func(e Event) (Event, error) { e.References = []string{"https://host/?token=secret"}; return e, nil },
			want:      ErrInvalid,
			callbacks: 1,
		},
		{
			name:      "malformed-input",
			prepare:   func(e *Event) { e.Payload = json.RawMessage(`{`) },
			project:   func(e Event) (Event, error) { return e, nil },
			want:      ErrInvalid,
			callbacks: 0,
		},
		{
			name:      "unsupported",
			prepare:   func(e *Event) { e.Version = 2 },
			project:   func(e Event) (Event, error) { return e, nil },
			want:      ErrUnsupported,
			callbacks: 0,
		},
		{
			name:      "conflict",
			prepare:   func(e *Event) { e.Sequence = 1 },
			project:   func(e Event) (Event, error) { return e, nil },
			want:      ErrConflict,
			callbacks: 0,
		},
		{
			name:      "cancelled",
			cancelled: true,
			project:   func(e Event) (Event, error) { return e, nil },
			want:      context.Canceled,
			callbacks: 0,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange.
			p := &captureMemoryPolicy{project: func(e Event) (Event, error) { return e, nil }}
			c, err := NewCapture(CaptureConfig{Policy: p, KnownKinds: []string{"output"}, MaxEvents: 1, MaxBytes: 1024})
			if err != nil {
				t.Fatal(err)
			}
			e := Event{
				Version:       1,
				Sequence:      1,
				Kind:          "output",
				CorrelationID: "trial",
				Payload:       json.RawMessage(`{"x":1}`),
			}
			if err = c.Record(context.Background(), e); err != nil {
				t.Fatal(err)
			}
			p.calls = 0
			p.project = tc.project
			e.Sequence = 2
			if tc.prepare != nil {
				tc.prepare(&e)
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if tc.cancelled {
				cancel()
			}
			// Act.
			err = c.Record(ctx, e)
			r := c.Seal()
			// Assert.
			if !errors.Is(err, tc.want) || p.calls != tc.callbacks {
				t.Fatalf("error %v want %v; callbacks %d want %d", err, tc.want, p.calls, tc.callbacks)
			}
			if len(r.Events) != 1 || ValidateEvidence(r) != nil {
				t.Fatalf("invalid retained evidence: %+v", r)
			}
		})
	}
}

//nolint:gocognit // Boundary corpus checks normalization and ownership together.
func TestCapturePolicyInputNormalizationAndIsolation(t *testing.T) {
	cases := []Event{
		{
			Version:       1,
			Sequence:      1,
			Kind:          "output",
			CorrelationID: "trial",
			Payload:       json.RawMessage(" {\"text\":\"<>&\\u2028\",\"n\":1e+02} \n"),
			References:    []string{"artifact:local"},
		},
		{
			Version:       1,
			Sequence:      1,
			Kind:          "output",
			CorrelationID: string([]byte{0xff, 0xfe}),
			Payload:       json.RawMessage(`null`),
			References:    []string{string([]byte{0xff, 0xfe})},
		},
		{Version: 1, Sequence: 1, Kind: "output", Payload: json.RawMessage{}, References: []string{}},
		{Version: 1, Sequence: 1, Kind: "output", Payload: json.RawMessage(`{"text":"\ud800"}`)},
		{Version: 1, Sequence: 1, Kind: "output", Payload: json.RawMessage{'"', 0xff, '"'}},
	}
	for i, input := range cases {
		t.Run(string(rune('a'+i)), func(t *testing.T) {
			// Arrange.
			want, err := cloneJSON(input)
			if err != nil {
				t.Fatal(err)
			}
			originalPayload := append([]byte(nil), input.Payload...)
			originalReferences := append([]string(nil), input.References...)
			var observed Event
			p := &captureMemoryPolicy{project: func(e Event) (Event, error) {
				observed, _ = cloneJSON(e)
				if len(e.Payload) > 0 {
					e.Payload[0] = '!'
				}
				if len(e.References) > 0 {
					e.References[0] = "changed"
				}
				return Event{}, errors.New("stop")
			}}
			c, err := NewCapture(CaptureConfig{Policy: p, KnownKinds: []string{"output"}, MaxEvents: 1, MaxBytes: 1024})
			if err != nil {
				t.Fatal(err)
			}
			// Act.
			_ = c.Record(context.Background(), input)
			// Assert.
			if p.calls != 1 || !reflect.DeepEqual(observed, want) {
				t.Fatalf("policy input mismatch: %#v want %#v", observed, want)
			}
			if string(input.Payload) != string(originalPayload) ||
				!reflect.DeepEqual(append([]string(nil), input.References...), originalReferences) {
				t.Fatal("policy mutated caller input")
			}
		})
	}
}

func TestCaptureReturnedRecordAndPolicyOutputIsolation(t *testing.T) {
	// Arrange.
	input := Event{
		Version:  1,
		Sequence: 1,
		Kind:     "output",
		Payload:  json.RawMessage(`{"nested":{"list":[1,2]},"secret":"drop"}`),
	}
	var output Event
	p := &captureMemoryPolicy{project: func(e Event) (Event, error) {
		e.Payload = json.RawMessage(`{"nested":{"list":[1,2]}}`)
		e.References = []string{"artifact:local"}
		output = e
		return e, nil
	}}
	c, err := NewCapture(CaptureConfig{Policy: p, KnownKinds: []string{"output"}, MaxEvents: 1, MaxBytes: 1024})
	if err != nil {
		t.Fatal(err)
	}
	// Act.
	err = c.Record(context.Background(), input)
	output.Payload[0] = '!'
	output.References[0] = "changed"
	first := c.Seal()
	want, _ := cloneJSON(first)
	first.Events[0].Payload[0] = '!'
	first.Events[0].References[0] = "changed"
	first.Coverage["output"] = false
	second := c.Seal()
	// Assert.
	if err != nil || !reflect.DeepEqual(second, want) || ValidateEvidence(second) != nil {
		t.Fatalf("private state was changed: %v", err)
	}
	if string(second.Events[0].Payload) != `{"nested":{"list":[1,2]}}` || second.Redactions != 1 {
		t.Fatal("privacy projection not retained")
	}
}

func TestCaptureCanonicalRetentionSize(t *testing.T) {
	for _, payload := range []json.RawMessage{nil, {}, json.RawMessage(`null`), json.RawMessage(`{"s":"<>&\u2028","n":1e+02,"nested":[{},[]]}`)} {
		// Arrange.
		input := Event{
			Version:       1,
			Sequence:      1,
			Kind:          "output",
			CorrelationID: "<trial>",
			Payload:       payload,
			References:    []string{"artifact:local"},
		}
		projected, err := validateProjectedEvent(input, input)
		if err != nil {
			t.Fatal(err)
		}
		canonicalBytes, err := canonical(projected)
		if err != nil {
			t.Fatal(err)
		}
		p := &captureMemoryPolicy{project: func(e Event) (Event, error) { return e, nil }}
		c, err := NewCapture(
			CaptureConfig{Policy: p, KnownKinds: []string{"output"}, MaxEvents: 1, MaxBytes: len(canonicalBytes)},
		)
		if err != nil {
			t.Fatal(err)
		}
		// Act.
		err = c.Record(context.Background(), input)
		record := c.Seal()
		// Assert.
		if err != nil || record.Truncated || c.bytes != len(canonicalBytes) || ValidateEvidence(record) != nil {
			t.Fatalf("retention size drift: %v got %d want %d", err, c.bytes, len(canonicalBytes))
		}
	}
}

func TestCaptureSealConfigurationStringNormalization(t *testing.T) {
	cases := []CaptureConfig{
		{
			Policy:     FieldPolicy{ID: string([]byte{0xff, 0xfe})},
			KnownKinds: []string{"output"},
			MaxEvents:  1,
			MaxBytes:   1024,
		},
		{
			Policy:     FieldPolicy{ID: "policy"},
			KnownKinds: []string{string([]byte{0xff})},
			MaxEvents:  1,
			MaxBytes:   1024,
		},
	}
	for _, config := range cases {
		// Arrange.
		c, err := NewCapture(config)
		if err != nil {
			t.Fatal(err)
		}
		// Act.
		record := c.Seal()
		want, err := cloneJSON(c.record)
		// Assert.
		if err != nil || !reflect.DeepEqual(record, want) {
			t.Fatalf("config normalization drift: %#v want %#v (%v)", record, want, err)
		}
	}
}

func FuzzCaptureInputNormalization(f *testing.F) {
	f.Add([]byte(` {"n":1e+02,"text":"<>&\u2028"} `), "trial", "artifact:local")
	f.Add([]byte{'"', 0xff, '"'}, string([]byte{0xff, 0xfe}), string([]byte{0xff}))
	f.Add([]byte(`{"a":1,"a":2}`), "trial", "artifact:local")
	f.Add([]byte{}, "", "")
	f.Fuzz(func(t *testing.T, payload []byte, correlation, reference string) {
		// Arrange.
		input := Event{
			Version:       1,
			Sequence:      1,
			Kind:          "output",
			CorrelationID: correlation,
			Payload:       payload,
			References:    []string{reference},
		}
		want, cloneErr := cloneJSON(input)
		var observed Event
		stop := errors.New("stop")
		policy := &captureMemoryPolicy{project: func(e Event) (Event, error) { observed = e; return Event{}, stop }}
		capture, err := NewCapture(
			CaptureConfig{Policy: policy, KnownKinds: []string{"output"}, MaxEvents: 1, MaxBytes: len(payload) + 1024},
		)
		if err != nil {
			t.Fatal(err)
		}
		// Act.
		err = capture.Record(context.Background(), input)
		// Assert.
		if cloneErr != nil {
			if policy.calls != 0 || !errors.Is(err, ErrInvalid) {
				t.Fatalf("invalid payload called policy: %d %v", policy.calls, err)
			}
			return
		}
		if policy.calls != 1 || !errors.Is(err, stop) || !reflect.DeepEqual(observed, want) {
			t.Fatalf("normalization mismatch: %#v want %#v (%v)", observed, want, err)
		}
	})
}

func TestCaptureSealRejectsNormalizedCoverageKeyCollision(t *testing.T) {
	// Arrange.
	c, err := NewCapture(CaptureConfig{
		Policy:     FieldPolicy{ID: "policy"},
		KnownKinds: []string{string([]byte{0xff}), string([]byte{0xfe})},
		MaxEvents:  1, MaxBytes: 1024,
	})
	if err != nil {
		t.Fatal(err)
	}
	// Act.
	var recovered any
	func() {
		defer func() { recovered = recover() }()
		c.Seal()
	}()
	// Assert.
	if recovered == nil {
		t.Fatal("Seal accepted coverage keys that collide after JSON string normalization")
	}
}

func captureMemoryNestedPayload(depth int, mixed bool) json.RawMessage {
	var payload strings.Builder
	for i := range depth {
		if mixed && i%2 == 0 {
			payload.WriteString(`{"x":`)
		} else {
			payload.WriteByte('[')
		}
	}
	payload.WriteByte('0')
	for i := depth - 1; i >= 0; i-- {
		if mixed && i%2 == 0 {
			payload.WriteByte('}')
		} else {
			payload.WriteByte(']')
		}
	}
	return json.RawMessage(payload.String())
}

//nolint:gocognit // Independently compare input, projected, saturated and container-kind boundaries.
func TestCaptureDeepEventBoundary(t *testing.T) {
	for _, mixed := range []bool{false, true} {
		for _, depth := range []int{9998, 9999, 10000} {
			for _, mode := range []string{"input", "projected-fresh", "projected-saturated"} {
				t.Run(fmt.Sprintf("mixed=%t/%s/%d", mixed, mode, depth), func(t *testing.T) {
					// Arrange.
					payload := captureMemoryNestedPayload(depth, mixed)
					policy := &captureMemoryPolicy{project: func(e Event) (Event, error) { return e, nil }}
					capture, err := NewCapture(
						CaptureConfig{Policy: policy, KnownKinds: []string{"output"}, MaxEvents: 1, MaxBytes: 1 << 20},
					)
					if err != nil {
						t.Fatal(err)
					}
					input := Event{Version: 1, Sequence: 1, Kind: "output", Payload: json.RawMessage(`0`)}
					if mode == "projected-saturated" {
						if err = capture.Record(context.Background(), input); err != nil {
							t.Fatal(err)
						}
						input.Sequence = 2
					}
					if mode == "input" {
						input.Payload = payload
					} else {
						policy.project = func(e Event) (Event, error) { e.Payload = payload; return e, nil }
					}
					policy.calls = 0
					var wantErr error
					wantCalls := 1
					switch {
					case mode == "input" && depth == 10000:
						wantErr = ErrInvalid
						wantCalls = 0
					case depth == 10000:
						projected := input
						projected.Payload = payload
						projected, err = validateProjectedEvent(input, projected)
						if err != nil {
							t.Fatal(err)
						}
						_, wantErr = canonical(projected)
						if wantErr == nil {
							t.Fatal("depth fixture must fail at whole Event boundary")
						}
					case mode == "projected-saturated":
						wantErr = ErrIncomplete
					}
					// Act.
					err = capture.Record(context.Background(), input)
					// Assert.
					if policy.calls != wantCalls {
						t.Fatalf("callbacks %d want %d", policy.calls, wantCalls)
					}
					if wantErr == nil {
						if err != nil {
							t.Fatal(err)
						}
					} else if err == nil || err.Error() != wantErr.Error() {
						t.Fatalf("error %v want %v", err, wantErr)
					}
					wantRetained := 0
					if mode == "projected-saturated" || wantErr == nil {
						wantRetained = 1
					}
					if len(capture.record.Events) != wantRetained {
						t.Fatalf("retained %d want %d", len(capture.record.Events), wantRetained)
					}
				})
			}
		}
	}
}

//nolint:gocognit // Compare legacy error, panic and successful snapshot behavior at depth boundaries.
func TestCaptureDeepSealBoundary(t *testing.T) {
	for _, depth := range []int{9996, 9997, 9998, 9999} {
		t.Run(strconv.Itoa(depth), func(t *testing.T) {
			// Arrange.
			policy := &captureMemoryPolicy{project: func(e Event) (Event, error) { return e, nil }}
			capture, err := NewCapture(
				CaptureConfig{Policy: policy, KnownKinds: []string{"output"}, MaxEvents: 1, MaxBytes: 1 << 20},
			)
			if err != nil {
				t.Fatal(err)
			}
			if err = capture.Record(
				context.Background(),
				Event{Version: 1, Sequence: 1, Kind: "output", Payload: captureMemoryNestedPayload(depth, false)},
			); err != nil {
				t.Fatal(err)
			}
			legacy := capture.record
			legacy.State = sealedState
			canonicalBytes, legacyErr := canonical(legacy)
			var want EvidenceRecord
			if legacyErr == nil {
				legacy.Revision = digest(canonicalBytes)
				want, legacyErr = cloneJSON(legacy)
			}
			// Act.
			var recovered any
			var got EvidenceRecord
			func() { defer func() { recovered = recover() }(); got = capture.Seal() }()
			// Assert.
			if legacyErr != nil {
				if recovered == nil {
					t.Fatal("Seal bypassed legacy depth failure")
				}
				return
			}
			if recovered != nil || !reflect.DeepEqual(got, want) {
				t.Fatalf("Seal depth drift: panic=%v", recovered)
			}
		})
	}
}

func TestCaptureInvalidMetadataCanonicalRetentionSize(t *testing.T) {
	// Arrange.
	input := Event{
		Version:       1,
		Sequence:      1,
		Kind:          "output",
		CorrelationID: string([]byte{0xff, 0xfe}),
		Payload:       json.RawMessage(`{"n":1e+02}`),
	}
	canonicalBytes, err := canonical(input)
	if err != nil {
		t.Fatal(err)
	}
	policy := &captureMemoryPolicy{project: func(e Event) (Event, error) {
		// A host may return the original metadata after the input JSON copy
		// normalized its policy argument; identity validation permits this.
		e.CorrelationID = input.CorrelationID
		return e, nil
	}}
	capture, err := NewCapture(
		CaptureConfig{Policy: policy, KnownKinds: []string{"output"}, MaxEvents: 1, MaxBytes: len(canonicalBytes)},
	)
	if err != nil {
		t.Fatal(err)
	}
	// Act.
	err = capture.Record(context.Background(), input)
	record := capture.Seal()
	// Assert.
	if err != nil || capture.bytes != len(canonicalBytes) || len(record.Events) != 1 {
		t.Fatalf("metadata retention size drift: err=%v size=%d want=%d", err, capture.bytes, len(canonicalBytes))
	}
	if ValidateEvidence(record) != nil {
		t.Fatal("normalized metadata evidence is invalid")
	}
}
