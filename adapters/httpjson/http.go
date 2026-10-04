package httpjson

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"mime"
	"net/http"
	"net/url"

	"github.com/skosovsky/evaly"
)

type Trial struct {
	ID           string `json:"id"`
	CaseRevision string `json:"case_revision"`
	Seed         int64  `json:"seed"`
	Fixture      string `json:"fixture"`
	Reset        string `json:"reset"`
}
type Request struct {
	Version     int                 `json:"version"`
	InputCodec  evaly.CodecIdentity `json:"input_codec"`
	OutputCodec evaly.CodecIdentity `json:"output_codec"`
	Trial       Trial               `json:"trial"`
	Input       json.RawMessage     `json:"input"`
}

// EvidenceDelivery is a host declaration independent of target success.
// Reason is a non-secret classification label, required only when incomplete.
type EvidenceDelivery struct {
	Complete bool   `json:"complete"`
	Reason   string `json:"reason"`
}

func (e EvidenceDelivery) Validate() error {
	if e.Complete && e.Reason != "" || !e.Complete && e.Reason == "" {
		return evaly.ErrInvalid
	}
	return nil
}

// Invocation preserves available effects even when the host returns an error.
// The host must explicitly declare delivery completeness.
type Invocation[O any] struct {
	Output   O
	Usage    evaly.Usage
	Events   []evaly.Event
	Evidence EvidenceDelivery
}
type Response struct {
	Version      int                       `json:"version"`
	Status       string                    `json:"status"`
	Output       json.RawMessage           `json:"output,omitempty"`
	Usage        evaly.Usage               `json:"usage"`
	Events       []evaly.Event             `json:"events"`
	Capabilities evaly.InteropCapabilities `json:"capabilities"`
	Evidence     EvidenceDelivery          `json:"evidence"`
}
type Doer interface {
	Do(*http.Request) (*http.Response, error)
}
type Target[I, O, E any] struct {
	URL            string
	Client         Doer
	Input          evaly.Codec[I]
	Output         evaly.Codec[O]
	MaxBytes       int64
	Fixture, Reset string
}

// Validate checks local configuration without dispatching any HTTP request.
func (t Target[I, O, E]) Validate() error {
	u, e := url.Parse(t.URL)
	if e != nil || u.User != nil || u.Host == "" || u.Fragment != "" || (u.Scheme != "http" && u.Scheme != "https") ||
		t.MaxBytes <= 0 ||
		t.MaxBytes == math.MaxInt64 ||
		t.Fixture == "" ||
		t.Reset == "" {
		return evaly.ErrInvalid
	}
	for _, port := range []any{t.Client, t.Input, t.Output} {
		if e := evaly.ValidatePort(port); e != nil {
			return e
		}
	}
	if e := evaly.ValidateCodecIdentity(t.Input.Identity()); e != nil {
		return e
	}
	return evaly.ValidateCodecIdentity(t.Output.Identity())
}
func (t Target[I, O, E]) Run(ctx context.Context, i I, tc evaly.TrialContext[E]) (evaly.TargetResult[O], error) {
	var result evaly.TargetResult[O]
	if e := t.Validate(); e != nil {
		return result, e
	}
	if e := evaly.ValidatePort(tc.Evidence); e != nil {
		return result, e
	}
	input, e := t.Input.Encode(i)
	if e != nil {
		return result, e
	}
	wire := Request{
		Version:     2,
		InputCodec:  t.Input.Identity(),
		OutputCodec: t.Output.Identity(),
		Trial:       Trial{tc.ID, tc.CaseRevision, tc.Seed, t.Fixture, t.Reset},
		Input:       input,
	}
	body, e := json.Marshal(wire)
	if e != nil || int64(len(body)) > t.MaxBytes {
		return result, evaly.ErrInvalid
	}
	if _, e = evaly.DecodeWire[Request](body); e != nil {
		return result, e
	}
	req, e := http.NewRequestWithContext(ctx, http.MethodPost, t.URL, bytes.NewReader(body))
	if e != nil {
		return result, e
	}
	req.Header.Set("Content-Type", "application/json")
	incomplete := func(reason string) { tc.Evidence.MarkIncomplete(reason) }
	if e := ctx.Err(); e != nil {
		return result, e
	}
	resp, e := t.Client.Do(req)
	if e != nil {
		incomplete("http_transport")
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
		return result, e
	}
	if resp == nil || evaly.ValidatePort(resp.Body) != nil {
		incomplete("http_response")
		return result, evaly.ErrInvalid
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		incomplete("http_status")
		return result, fmt.Errorf("HTTP status %d: %w", resp.StatusCode, evaly.ErrUnsupported)
	}
	media, _, mediaErr := mime.ParseMediaType(resp.Header.Get("Content-Type"))
	if mediaErr != nil || media != "application/json" {
		incomplete("http_content_type")
		return result, evaly.ErrInvalid
	}
	data, e := io.ReadAll(io.LimitReader(resp.Body, t.MaxBytes+1))
	if e != nil {
		incomplete("http_read")
		return result, e
	}
	if int64(len(data)) > t.MaxBytes {
		incomplete("http_size")
		return result, evaly.ErrInvalid
	}
	out, e := evaly.DecodeWire[Response](data)
	if e != nil {
		incomplete("http_wire")
		return result, e
	}
	if out.Version != 2 || !evaly.CheckMapping(out.Capabilities).Supported ||
		out.Status != "completed" && out.Status != "target_error" {
		incomplete("http_protocol")
		return result, evaly.ErrUnsupported
	}
	if e := out.Evidence.Validate(); e != nil {
		incomplete("http_evidence_declaration")
		return result, e
	}
	if out.Status == "completed" && len(out.Output) == 0 || out.Status == "target_error" && len(out.Output) != 0 {
		incomplete("http_output_contract")
		return result, evaly.ErrInvalid
	}
	result.Usage = out.Usage
	for _, event := range out.Events {
		if e := tc.Evidence.Record(ctx, event); e != nil {
			incomplete("http_event_retention")
			return result, e
		}
	}
	if !out.Evidence.Complete {
		incomplete("http_host_incomplete")
	}
	if out.Status == "target_error" {
		return result, evaly.ErrTarget
	}
	result.Output, e = t.Output.Decode(out.Output)
	return result, e
}

// Handler is the reference v2 server. Invoke owns isolation, fixture/reset
// compatibility and the declaration of evidence completeness.
func Handler[I, O any](
	ic evaly.Codec[I],
	oc evaly.Codec[O],
	maxBytes int64,
	invoke func(context.Context, I, Trial) (Invocation[O], error),
) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fail := func(code int) { http.Error(w, "invalid evaly request", code) }
		media, _, mediaErr := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if evaly.ValidatePort(ic) != nil || evaly.ValidatePort(oc) != nil || invoke == nil || maxBytes <= 0 ||
			maxBytes == math.MaxInt64 ||
			r.Method != http.MethodPost ||
			mediaErr != nil ||
			media != "application/json" {
			fail(http.StatusBadRequest)
			return
		}
		if evaly.ValidateCodecIdentity(ic.Identity()) != nil || evaly.ValidateCodecIdentity(oc.Identity()) != nil {
			fail(http.StatusBadRequest)
			return
		}
		b, e := io.ReadAll(io.LimitReader(r.Body, maxBytes+1))
		if e != nil || int64(len(b)) > maxBytes {
			fail(http.StatusRequestEntityTooLarge)
			return
		}
		req, e := evaly.DecodeWire[Request](b)
		if e != nil {
			fail(http.StatusBadRequest)
			return
		}
		if req.Version != 2 || req.InputCodec != ic.Identity() || req.OutputCodec != oc.Identity() ||
			req.Trial.ID == "" ||
			req.Trial.CaseRevision == "" ||
			req.Trial.Fixture == "" ||
			req.Trial.Reset == "" {
			fail(http.StatusUnprocessableEntity)
			return
		}
		i, e := ic.Decode(req.Input)
		if e != nil {
			fail(http.StatusBadRequest)
			return
		}
		invocation, invokeErr := invoke(r.Context(), i, req.Trial)
		if invocation.Evidence.Validate() != nil {
			fail(http.StatusInternalServerError)
			return
		}
		resp := Response{
			Version:  2,
			Status:   "completed",
			Usage:    invocation.Usage,
			Events:   invocation.Events,
			Evidence: invocation.Evidence,
			Capabilities: evaly.InteropCapabilities{
				Version:       1,
				Outcome:       true,
				ResetIdentity: true,
				Evidence:      true,
				RichStatus:    true,
				MetricScales:  true,
			},
		}
		if invokeErr != nil {
			resp.Status = "target_error"
		} else {
			resp.Output, e = oc.Encode(invocation.Output)
			if e != nil {
				resp.Status = "target_error"
				resp.Output = nil
			}
		}
		data, e := json.Marshal(resp)
		if e != nil || int64(len(data)) > maxBytes {
			fail(http.StatusInternalServerError)
			return
		}
		if _, e := evaly.DecodeWire[Response](data); e != nil {
			fail(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(data)
	})
}
