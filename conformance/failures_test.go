package conformance_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/skosovsky/evaly"
	"github.com/skosovsky/evaly/conformance"
	"github.com/skosovsky/evaly/optimizer"
)

type partialPairJudge struct{ calls int }

func (j *partialPairJudge) JudgePair(ctx context.Context, _ evaly.PairRequest[int]) (evaly.PairJudgment, error) {
	if err := ctx.Err(); err != nil {
		return evaly.PairJudgment{}, err
	}
	j.calls++
	result := evaly.PairJudgment{Preferred: "A", Usage: evaly.Usage{Known: true, Units: float64(j.calls)}}
	if j.calls == 2 {
		return result, errors.New("provider failed after paid judgment")
	}
	return result, nil
}

type effectExport struct{ delivered int }

func (*effectExport) Capabilities() evaly.ExportCapabilities {
	return evaly.ExportCapabilities{EnvelopeVersion: 1}
}
func (s *effectExport) Deliver(ctx context.Context, r evaly.DeliveryRecord) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.delivered++
	r.Artifact.Data[0] = '!'
	return errors.New("acknowledgement lost after external effect")
}
func TestFailureReferenceAdapters(t *testing.T) {
	t.Run("grader_partial_paid_error", func(t *testing.T) {
		checkPartialPaidGraderError(t)
	})
	t.Run("pair_reverse_paid_error", func(t *testing.T) {
		checkReversePaidPairError(t)
	})
	t.Run("evidence_delivery_loss", func(t *testing.T) {
		conformance.EvidenceFailures(t, func() (*evaly.Capture, evaly.Event, error) {
			capture, err := evaly.NewCapture(
				evaly.CaptureConfig{
					Policy:        evaly.FieldPolicy{ID: "action-v1", Allowed: map[string][]string{"tool": {"action"}}},
					KnownKinds:    []string{"tool"},
					RequiredKinds: []string{"tool"},
					MaxEvents:     4,
					MaxBytes:      4096,
				},
			)
			return capture, evaly.Event{
				Version:  1,
				Sequence: 1,
				Kind:     "tool",
				Payload:  json.RawMessage(`{"action":"apply"}`),
			}, err
		})
	})
	t.Run("export_effect_then_error", func(t *testing.T) {
		conformance.ExportFailures(t, func() conformance.ExportFault {
			envelope, err := evaly.NewEnvelope("evidence", "partial-export", map[string]string{"effect": "retained"})
			if err != nil {
				t.Fatal(err)
			}
			sink := &effectExport{}
			return conformance.ExportFault{
				Sink:   sink,
				Record: evaly.DeliveryRecord{ObservationID: "delivery-1", Artifact: envelope},
				Verify: func(t *testing.T) {
					if sink.delivered != 1 {
						t.Fatal("external effect not observed", sink.delivered)
					}
				},
			}
		})
	})
	t.Run("proposal_partial_paid_error", func(t *testing.T) {
		conformance.ProposalFailures(t, func() conformance.ProposalFault[int, int, int] {
			usage := evaly.Usage{Known: true, Units: 4}
			calls := 0
			proposer := optimizer.ProposalFunc[int, int, int]{
				Identity: "partial-v1",
				Generate: func(ctx context.Context, _ optimizer.ProposalRequest[int, int]) (optimizer.ProposalResult[int], error) {
					if err := ctx.Err(); err != nil {
						return optimizer.ProposalResult[int]{}, err
					}
					calls++
					return optimizer.ProposalResult[int]{
						Candidates: []optimizer.Proposal[int]{{ID: "candidate", Value: 2}},
						Usage:      usage,
					}, errors.New(
						"paid proposal interrupted",
					)
				},
			}
			return conformance.ProposalFault[int, int, int]{
				Proposer: proposer,
				Request: optimizer.ProposalRequest[int, int]{
					Round:      0,
					Maximum:    2,
					DispatchID: "search/proposal/0",
				},
				ExpectedCandidates: 1,
				ExpectedUsage:      usage,
				Verify: func(t *testing.T, result optimizer.ProposalResult[int], err error) {
					if calls != 1 || result.Candidates[0].Value != 2 || err == nil {
						t.Fatal("host partial proposal mismatch")
					}
				},
			}
		})
	})
}

func checkPartialPaidGraderError(t *testing.T) {
	conformance.GraderFailures(t, func() conformance.GraderFault[int, int, int] {
		usage := evaly.Usage{Known: true, Units: 3}
		return conformance.GraderFault[int, int, int]{
			Grader: evaly.GraderFunc[int, int, int]{
				Identity: evaly.GraderRevision{ID: "partial", Implementation: "1", Rubric: "1"},
				Evaluate: func(context.Context, evaly.View[int, int, int]) (evaly.Grade, error) {
					return evaly.Grade{
						Status:     evaly.Scored,
						Assertions: []evaly.Assertion{{Name: "unconfirmed", Pass: true}},
						Usage:      usage,
					}, errors.New(
						"provider failed",
					)
				},
			},
			View:          func() (evaly.View[int, int, int], error) { return evaly.View[int, int, int]{}, nil },
			ExpectedUsage: usage,
		}
	})
}

func checkReversePaidPairError(t *testing.T) {
	conformance.PairJudgeFailures(t, func() conformance.PairFault[int] {
		codec := evaly.JSONCodec[int]{ID: "int", Version: "1"}
		a, err := evaly.SealSnapshot(1, codec)
		if err != nil {
			t.Fatal(err)
		}
		b, err := evaly.SealSnapshot(2, codec)
		if err != nil {
			t.Fatal(err)
		}
		return conformance.PairFault[int]{
			Judge:           &partialPairJudge{},
			A:               a,
			B:               b,
			ExpectedForward: evaly.Usage{Known: true, Units: 1},
			ExpectedReverse: evaly.Usage{Known: true, Units: 2},
		}
	})
}
