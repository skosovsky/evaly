package evaly

import (
	"context"
	"errors"
	"sync"
)

type DeliveryRecord struct {
	ObservationID string   `json:"observation_id"`
	Artifact      Envelope `json:"artifact"`
}
type ExportCapabilities struct {
	Deduplication   bool `json:"Deduplication"`
	EnvelopeVersion int  `json:"EnvelopeVersion"`
	BooleanOnly     bool `json:"BooleanOnly"`
}
type ExportSink interface {
	Capabilities() ExportCapabilities
	Deliver(context.Context, DeliveryRecord) error
}
type Delivery struct {
	ID        string `json:"id"`
	State     string `json:"state"`
	Semantics string `json:"semantics"`
	Reason    string `json:"reason,omitempty"`
}

// Export never mutates the source artifact's verdict. Retry keeps observation ID.
func Export(ctx context.Context, s ExportSink, r DeliveryRecord) Delivery {
	out := Delivery{
		ID:        r.ObservationID,
		State:     failedState,
		Semantics: "at_least_once_with_possible_duplicates",
		Reason:    "",
	}
	if ValidatePort(s) != nil || r.ObservationID == "" {
		out.Reason = invalidState
		return out
	}
	if ctx.Err() != nil {
		out.Reason = cancelledState
		return out
	}
	caps := s.Capabilities()
	if caps.Deduplication {
		out.Semantics = "deduplicated_by_observation_identity"
	}
	if caps.EnvelopeVersion != 1 || caps.BooleanOnly {
		out.Reason = "unsupported"
		return out
	}
	if e := ValidateEnvelope(r.Artifact); e != nil {
		out.Reason = deliveryReason(e)
		return out
	}
	cloned, err := cloneJSON(r)
	if err != nil {
		out.Reason = invalidState
		return out
	}
	if ctx.Err() != nil {
		out.Reason = cancelledState
		return out
	}
	if e := s.Deliver(ctx, cloned); e != nil {
		out.Reason = deliveryReason(e)
		return out
	}
	out.State = "delivered"
	return out
}

func deliveryReason(err error) string {
	switch {
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return cancelledState
	case errors.Is(err, ErrConflict):
		return "conflict"
	case errors.Is(err, ErrUnsupported):
		return "unsupported"
	case errors.Is(err, ErrInvalid), errors.Is(err, ErrCorrupt), errors.Is(err, ErrUnsealed):
		return invalidState
	default:
		return "delivery_failure"
	}
}

// MemoryExport is a local reference sink with explicit optional deduplication.
// MemoryExport is a process-local reference sink, not a durable outbox.
// Configure Dedup and Fail before use and do not mutate them concurrently.
type MemoryExport struct {
	mu      sync.Mutex
	Dedup   bool `json:"Dedup"`
	Fail    bool `json:"Fail"`
	records []DeliveryRecord
}

func (s *MemoryExport) Capabilities() ExportCapabilities {
	return ExportCapabilities{Deduplication: s.Dedup, EnvelopeVersion: 1, BooleanOnly: false}
}
func (s *MemoryExport) Deliver(ctx context.Context, r DeliveryRecord) error {
	if e := ctx.Err(); e != nil {
		return e
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.Fail {
		return ErrDelivery
	}
	if s.Dedup {
		for _, old := range s.records {
			if old.ObservationID == r.ObservationID {
				if old.Artifact.Checksum != r.Artifact.Checksum {
					return ErrConflict
				}
				return nil
			}
		}
	}
	cp, e := cloneJSON(r)
	if e != nil {
		return e
	}
	s.records = append(s.records, cp)
	return nil
}
func (s *MemoryExport) Records() []DeliveryRecord {
	s.mu.Lock()
	defer s.mu.Unlock()
	return mustCloneJSON(s.records)
}

type InteropCapabilities struct {
	Version       int  `json:"Version"`
	Outcome       bool `json:"Outcome"`
	ResetIdentity bool `json:"ResetIdentity"`
	Evidence      bool `json:"Evidence"`
	RichStatus    bool `json:"RichStatus"`
	MetricScales  bool `json:"MetricScales"`
}
type LossReport struct {
	Supported bool     `json:"supported"`
	Losses    []string `json:"losses"`
}

// CheckMapping explicitly rejects lost guarantees rather than collapsing to booleans.
func CheckMapping(c InteropCapabilities) LossReport {
	r := LossReport{Supported: true, Losses: []string{}}
	if c.Version != 1 {
		r.Losses = append(r.Losses, "schema_version")
	}
	if !c.Outcome {
		r.Losses = append(r.Losses, "outcome")
	}
	if !c.ResetIdentity {
		r.Losses = append(r.Losses, "reset_identity")
	}
	if !c.Evidence {
		r.Losses = append(r.Losses, "evidence")
	}
	if !c.RichStatus {
		r.Losses = append(r.Losses, "status")
	}
	if !c.MetricScales {
		r.Losses = append(r.Losses, "metric_scales")
	}
	r.Supported = len(r.Losses) == 0
	return r
}
