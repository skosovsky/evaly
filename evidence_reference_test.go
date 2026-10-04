package evaly

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestReferenceContractAcrossCaptureRestoreAndGrades(t *testing.T) {
	for _, tc := range []struct {
		name, ref string
		valid     bool
	}{
		{"artifact", "artifact-123", true},
		{"urn", "urn:uuid:abc-123", true},
		{"object", "s3://bucket/trace-123?version=2", true},
		{"http", "https://host/trace?id=42", true},
		{"escaped-path", "artifact/path%23part", true},
		{"semicolon", "https://host/trace?token=private-secret;ignored=x", false},
		{"fragment", "https://host/trace#access_token=private-secret", false},
		{"plain-fragment", "urn:trace:abc#part", false},
		{"empty-fragment", "artifact#", false},
		{"userinfo", "https://user:private-secret@host/trace", false},
		{"encoded-key", "https://host/?%74oken=private-secret", false},
		{"case", "https://host/?Access_Token=private-secret", false},
		{"signed-uri", "s3://bucket/path?X-Amz-Signature=private-secret", false},
		{"sas-uri", "https://host/path?sig=private-secret", false},
		{"malformed-query", "artifact?ok=1&bad=%ZZ", false},
		{"empty", "", false},
		{"space", "artifact id", false},
		{"control", "artifact\x00id", false},
		{"malformed-uri", "urn:trace:%ZZ", false},
		{"missing-host", "https:/trace", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange.
			c, err := NewCapture(
				CaptureConfig{
					Policy:     FieldPolicy{ID: "refs", KeepReferences: true},
					KnownKinds: []string{"tool"},
					MaxEvents:  2,
					MaxBytes:   4096,
				},
			)
			if err != nil {
				t.Fatal(err)
			}
			event := Event{Version: 1, Sequence: 1, Kind: "tool", References: []string{tc.ref}}
			r := rehashEvidence(
				EvidenceRecord{
					Version:         1,
					State:           "sealed",
					Policy:          "refs",
					Events:          []Event{event},
					Coverage:        map[string]bool{"tool": true},
					ReplayAvailable: true,
				},
			)
			g := Grade{
				Revision:     GraderRevision{ID: "g", Implementation: "1", Rubric: "1"},
				Status:       NotApplicable,
				EvidenceRefs: []string{tc.ref},
			}
			// Act.
			captureErr := c.Record(context.Background(), event)
			captured := c.Seal()
			restoreErr, gradeErr := ValidateEvidence(r), ValidateGrade(g)
			// Assert.
			for _, err := range []error{captureErr, restoreErr, gradeErr} {
				if (err == nil) != tc.valid {
					t.Fatalf("valid=%v error=%v", tc.valid, err)
				}
				if err != nil && (!errors.Is(err, ErrInvalid) || strings.Contains(err.Error(), "private-secret")) {
					t.Fatal(err)
				}
			}
			if !tc.valid &&
				(len(captured.Events) != 0 || captured.State != "incomplete" || CompleteFor(captured, "tool")) {
				t.Fatal(captured)
			}
		})
	}
}

func TestDefaultPolicyDropsUntrustedReferences(t *testing.T) {
	// Arrange.
	c, err := NewCapture(
		CaptureConfig{Policy: FieldPolicy{ID: "drop"}, KnownKinds: []string{"tool"}, MaxEvents: 1, MaxBytes: 1024},
	)
	if err != nil {
		t.Fatal(err)
	}
	// Act.
	err = c.Record(
		context.Background(),
		Event{Version: 1, Sequence: 1, Kind: "tool", References: []string{"https://host/#access_token=private-secret"}},
	)
	r := c.Seal()
	// Assert: classification precedes retention.
	if err != nil || len(r.Events) != 1 || len(r.Events[0].References) != 0 || r.Redactions != 1 {
		t.Fatal(err, r)
	}
}
