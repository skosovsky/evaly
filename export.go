package evaly

import (
	"context"
	"sync"
)

type DeliveryRecord struct {
	ObservationID string   `json:"observation_id"`
	Artifact      Envelope `json:"artifact"`
}
type ExportCapabilities struct {
	Deduplication   bool
	EnvelopeVersion int
	BooleanOnly     bool
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
	out := Delivery{ID: r.ObservationID, State: "failed", Semantics: "at_least_once_with_possible_duplicates"}
	if s == nil || r.ObservationID == "" {
		out.Reason = "invalid_delivery"
		return out
	}
	caps := s.Capabilities()
	if caps.Deduplication {
		out.Semantics = "deduplicated_by_observation_identity"
	}
	if caps.EnvelopeVersion != 1 || caps.BooleanOnly {
		out.Reason = "unsupported_lossy_mapping"
		return out
	}
	if e := ValidateEnvelope(r.Artifact); e != nil {
		out.Reason = "invalid_artifact"
		return out
	}
	if e := s.Deliver(ctx, r); e != nil {
		out.Reason = "sink_unavailable"
		return out
	}
	out.State = "delivered"
	return out
}

// MemoryExport is a local reference sink with explicit optional deduplication.
type MemoryExport struct {
	mu      sync.Mutex
	Dedup   bool
	Fail    bool
	records []DeliveryRecord
}

func (s *MemoryExport) Capabilities() ExportCapabilities {
	return ExportCapabilities{Deduplication: s.Dedup, EnvelopeVersion: 1}
}
func (s *MemoryExport) Deliver(ctx context.Context, r DeliveryRecord) error {
	if e := ctx.Err(); e != nil {
		return e
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.Fail {
		return ErrUnsupported
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
	r, _ := cloneJSON(s.records)
	return r
}

type InteropCapabilities struct {
	Version                                                    int
	Outcome, ResetIdentity, Evidence, RichStatus, MetricScales bool
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
