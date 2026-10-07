package recipes

import (
	"context"
	"sync"

	"github.com/skosovsky/prompty"

	"github.com/skosovsky/evaly"
)

// CaptureBridge is trial-local. Policy projection occurs inside the evidence sink
// before retention. It does not infer full coverage from accepted local frames.
type CaptureBridge struct {
	mu      sync.Mutex
	Sink    evaly.EvidenceSink
	Sampled bool
}

func (b *CaptureBridge) Record(ctx context.Context, event evaly.Event, complete bool) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if evaly.ValidatePort(b.Sink) != nil {
		return evaly.ErrInvalid
	}
	if b.Sampled || !complete {
		b.Sink.MarkIncomplete()
	}
	if event.Sequence <= 0 || event.CorrelationID == "" {
		b.Sink.MarkIncomplete()
		return evaly.ErrInvalid
	}
	if err := b.Sink.Record(ctx, event); err != nil {
		b.Sink.MarkIncomplete()
		return err
	}
	return nil
}

// Finish uses an explicit source report. Nil is unknown, never complete.
func (b *CaptureBridge) Finish(report *prompty.CaptureReport) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if evaly.ValidatePort(b.Sink) == nil &&
		(b.Sampled || report == nil || report.Validate() != nil || !report.Closed || !report.Complete || len(report.Failures) != 0) {
		b.Sink.MarkIncomplete()
	}
}
