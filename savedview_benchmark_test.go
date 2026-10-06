package evaly

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

// BenchmarkSavedViewValidation separates immutable record validation from fresh
// decoding. It measures local JSON costs, not provider latency or judge quality.
func BenchmarkSavedViewValidation(b *testing.B) {
	for _, size := range []int{128, 16 * 1024} {
		b.Run(fmt.Sprintf("bytes_%d", size), func(b *testing.B) {
			saved, codec := benchmarkSavedView(b, size)
			record := saved.Record()
			b.Run("restore", func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					// Act / Assert.
					if _, err := RestoreSavedView(record, codec, codec, codec); err != nil {
						b.Fatal(err)
					}
				}
			})
			b.Run("fresh_view", func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					if _, err := saved.View(); err != nil {
						b.Fatal(err)
					}
				}
			})
			benchmarkSavedViewRescore(b, saved)
		})
	}
}

func benchmarkSavedView(b *testing.B, size int) (SavedView[string, string, string], JSONCodec[string]) {
	b.Helper()
	// Arrange.
	capture, err := NewCapture(CaptureConfig{
		Policy: FieldPolicy{ID: "benchmark"}, MaxEvents: 1, MaxBytes: 1024,
	})
	if err != nil {
		b.Fatal(err)
	}
	codec := JSONCodec[string]{ID: "string", Version: "1"}
	view := View[string, string, string]{
		Case:   Case[string, string]{ID: "case", Revision: "1", Input: strings.Repeat("x", size)},
		Output: strings.Repeat("y", size), Evidence: capture.Seal(),
	}
	saved, err := SaveView(view, "permitted", codec, codec, codec)
	if err != nil {
		b.Fatal(err)
	}
	return saved, codec
}

func benchmarkSavedViewRescore(b *testing.B, saved SavedView[string, string, string]) {
	b.Helper()
	for _, count := range []int{1, 8} {
		graders := make([]Grader[string, string, string], count)
		for i := range graders {
			graders[i] = GraderFunc[string, string, string]{
				Identity: GraderRevision{ID: fmt.Sprintf("grader_%d", i), Implementation: "1", Rubric: "1"},
				Evaluate: func(context.Context, View[string, string, string]) (Grade, error) {
					return Grade{Status: Scored, Assertions: []Assertion{{Name: "pass", Pass: true}}}, nil
				},
			}
		}
		b.Run(fmt.Sprintf("rescore_%d", count), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				if _, err := Rescore(
					context.Background(),
					saved,
					graders,
					"source",
					"",
					"rescore",
				); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
