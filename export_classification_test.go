package evaly

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
)

type classificationSink struct {
	err    error
	calls  int
	mutate bool
}

func (*classificationSink) Capabilities() ExportCapabilities {
	return ExportCapabilities{EnvelopeVersion: 1}
}
func (s *classificationSink) Deliver(_ context.Context, r DeliveryRecord) error {
	s.calls++
	if s.mutate {
		r.Artifact.Data[0] = '!'
		r.Artifact.Extensions["host"] = json.RawMessage(`"changed"`)
	}
	return s.err
}

func TestExportClassifiesWrappedErrorsWithoutRetainingDetails(t *testing.T) {
	for _, tc := range []struct {
		err    error
		reason string
	}{
		{ErrInvalid, "invalid"}, {ErrCorrupt, "invalid"}, {ErrUnsealed, "invalid"},
		{ErrConflict, "conflict"}, {ErrUnsupported, "unsupported"},
		{context.Canceled, "cancelled"}, {context.DeadlineExceeded, "cancelled"},
		{ErrDelivery, "delivery_failure"}, {errors.New("remote-private-secret"), "delivery_failure"},
	} {
		t.Run(tc.err.Error(), func(t *testing.T) {
			// Arrange.
			env, err := NewEnvelope("evidence", "delivery", map[string]string{"state": "sealed"})
			if err != nil {
				t.Fatal(err)
			}
			sink := &classificationSink{err: fmt.Errorf("remote-private-secret: %w", tc.err)}
			// Act.
			d := Export(context.Background(), sink, DeliveryRecord{ObservationID: "obs", Artifact: env})
			// Assert.
			if d.State != "failed" || d.Reason != tc.reason || sink.calls != 1 {
				t.Fatal(d, sink.calls)
			}
		})
	}
}

func TestExportPreflightAndDetachedDelivery(t *testing.T) {
	// Arrange.
	env, err := NewEnvelope("evidence", "delivery", map[string]string{"state": "sealed"})
	if err != nil {
		t.Fatal(err)
	}
	env.Extensions = map[string]json.RawMessage{"host": json.RawMessage(`"original"`)}
	env.Checksum = checksumEnvelope(env)
	r := DeliveryRecord{ObservationID: "obs", Artifact: env}
	before, _ := canonical(r)
	sink := &classificationSink{mutate: true}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	var typedNil *classificationSink
	// Act.
	nilDelivery := Export(context.Background(), typedNil, r)
	cancelled := Export(ctx, sink, r)
	delivered := Export(context.Background(), sink, r)
	after, _ := canonical(r)
	// Assert.
	if nilDelivery.Reason != "invalid" || cancelled.Reason != "cancelled" || sink.calls != 1 ||
		delivered.State != "delivered" ||
		string(before) != string(after) {
		t.Fatal(nilDelivery, cancelled, delivered, sink.calls)
	}
}

func TestExportConflictAndRetryPreserveArtifact(t *testing.T) {
	// Arrange.
	env, _ := NewEnvelope("evidence", "delivery", map[string]string{"state": "sealed"})
	other, _ := NewEnvelope("evidence", "delivery", map[string]string{"state": "incomplete"})
	sink := &MemoryExport{Dedup: true, Fail: true}
	r := DeliveryRecord{ObservationID: "obs", Artifact: env}
	// Act.
	failed := Export(context.Background(), sink, r)
	sink.Fail = false
	delivered := Export(context.Background(), sink, r)
	retry := Export(context.Background(), sink, r)
	conflict := Export(context.Background(), sink, DeliveryRecord{ObservationID: "obs", Artifact: other})
	// Assert.
	if failed.Reason != "delivery_failure" || delivered.State != "delivered" || retry.State != "delivered" ||
		conflict.Reason != "conflict" ||
		len(sink.Records()) != 1 ||
		sink.Records()[0].Artifact.Checksum != env.Checksum {
		t.Fatal(failed, delivered, retry, conflict, sink.Records())
	}
}
