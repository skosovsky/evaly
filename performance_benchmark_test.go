package evaly

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

func auditEvent(size int) Event {
	return Event{
		Version:       1,
		Sequence:      1,
		Kind:          "output",
		CorrelationID: "trial",
		Payload:       json.RawMessage(`{"text":"` + strings.Repeat("x", size) + `"}`),
		References:    []string{"artifact:local"},
	}
}
func auditCapture(t testing.TB, maxEvents int) *Capture {
	t.Helper()
	c, err := NewCapture(
		CaptureConfig{
			Policy: FieldPolicy{
				ID:             "audit",
				Allowed:        map[string][]string{"output": {"text"}},
				KeepReferences: true,
			},
			KnownKinds: []string{"output"},
			MaxEvents:  maxEvents,
			MaxBytes:   1 << 20,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

// Only a diagnostic comparison on already-valid typed Event values, not a
// replacement for cloneJSON's validation/encoding semantics on arbitrary input.
func auditCopyEvent(e Event) Event {
	e.Payload = append(json.RawMessage(nil), e.Payload...)
	e.References = append([]string(nil), e.References...)
	return e
}
func TestAuditEventCopyIsolation(t *testing.T) {
	// Arrange.
	original := auditEvent(128)
	// Act.
	direct := auditCopyEvent(original)
	reference, err := cloneJSON(original)
	// Assert.
	if err != nil || !reflect.DeepEqual(direct, reference) {
		t.Fatalf("valid event mismatch: %v", err)
	}
	direct.Payload[0] = '!'
	direct.References[0] = "changed"
	if original.Payload[0] != '{' || original.References[0] != "artifact:local" {
		t.Fatal("alias")
	}
}
func TestAuditSaturatedCaptureBounded(t *testing.T) {
	// Arrange.
	c := auditCapture(t, 1)
	e := auditEvent(1024)
	if err := c.Record(context.Background(), e); err != nil {
		t.Fatal(err)
	}
	// Act.
	for i := 2; i < 102; i++ {
		e.Sequence = i
		if err := c.Record(context.Background(), e); !errors.Is(err, ErrIncomplete) {
			t.Fatal(err)
		}
	}
	r := c.Seal()
	// Assert.
	if len(r.Events) != 1 || !r.Truncated || len(r.Errors) > maxEvidenceDiagnostics {
		t.Fatal("unbounded capture")
	}
	if err := ValidateEvidence(r); err != nil {
		t.Fatal(err)
	}
}
func BenchmarkAuditEventClone(b *testing.B) {
	for _, size := range []int{128, 16384} {
		b.Run(strconv.Itoa(size), func(b *testing.B) {
			e := auditEvent(size)
			b.Run("JSON", func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					out, err := cloneJSON(e)
					if err != nil || len(out.Payload) != len(e.Payload) {
						b.Fatal(err)
					}
				}
			})
			b.Run("TypedValidDiagnostic", func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					out := auditCopyEvent(e)
					if len(out.Payload) != len(e.Payload) {
						b.Fatal("length")
					}
				}
			})
		})
	}
}

//nolint:gocognit // Keep the original audit benchmark invocation and fixtures identical.
func BenchmarkAuditCapture(b *testing.B) {
	for _, size := range []int{128, 16384} {
		b.Run(strconv.Itoa(size), func(b *testing.B) {
			e := auditEvent(size)
			b.Run("FreshRecordAndSeal", func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					c := auditCapture(b, 1)
					if err := c.Record(context.Background(), e); err != nil {
						b.Fatal(err)
					}
					if len(c.Seal().Events) != 1 {
						b.Fatal("events")
					}
				}
			})
			b.Run("SaturatedRecord", func(b *testing.B) {
				c := auditCapture(b, 1)
				if err := c.Record(context.Background(), e); err != nil {
					b.Fatal(err)
				}
				b.ReportAllocs()
				for b.Loop() {
					e.Sequence++
					if err := c.Record(context.Background(), e); !errors.Is(err, ErrIncomplete) {
						b.Fatal(err)
					}
				}
			})
		})
	}
}
func BenchmarkAuditArtifactID(b *testing.B) {
	b.Run("Current", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			if !validArtifactID("experiment_123") {
				b.Fatal("ID")
			}
		}
	})
	re := regexp.MustCompile(`^[a-zA-Z0-9_-]{1,128}$`)
	b.Run("CompiledDiagnostic", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			if !re.MatchString("experiment_123") {
				b.Fatal("ID")
			}
		}
	})
}
func TestAuditArtifactIDCompiledEquivalent(t *testing.T) {
	// Arrange.
	re := regexp.MustCompile(`^[a-zA-Z0-9_-]{1,128}$`)
	corpus := []string{
		"",
		"a",
		"experiment_123",
		"a-b",
		"a/b",
		"é",
		"foo\n",
		"a b",
		strings.Repeat("a", 128),
		strings.Repeat("a", 129),
		"\x00",
	}
	for _, id := range corpus {
		// Act.
		want, got := re.MatchString(id), validArtifactID(id)
		// Assert.
		if want != got {
			t.Fatalf("ID %q: got %v want %v", id, got, want)
		}
	}
}
func BenchmarkAuditNewEnvelope(b *testing.B) {
	// Arrange.
	value := struct {
		Result string `json:"result"`
	}{"ok"}
	b.ReportAllocs()
	for b.Loop() {
		// Act.
		envelope, err := NewEnvelope("evidence", "experiment_123", value)
		// Assert.
		if err != nil || envelope.Checksum == "" {
			b.Fatal(err)
		}
	}
}
