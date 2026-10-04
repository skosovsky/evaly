package httpjson

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

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
type Response struct {
	Version      int                       `json:"version"`
	Status       string                    `json:"status"`
	Output       json.RawMessage           `json:"output,omitempty"`
	Usage        evaly.Usage               `json:"usage"`
	Events       []evaly.Event             `json:"events"`
	Capabilities evaly.InteropCapabilities `json:"capabilities"`
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

func (t Target[I, O, E]) Run(ctx context.Context, i I, tc evaly.TrialContext[E]) (evaly.TargetResult[O], error) {
	var result evaly.TargetResult[O]
	u, e := url.Parse(t.URL)
	if e != nil || u.User != nil || (u.Scheme != "http" && u.Scheme != "https") || t.Client == nil || t.Input == nil ||
		t.Output == nil ||
		t.MaxBytes <= 0 ||
		t.Fixture == "" ||
		t.Reset == "" {
		return result, evaly.ErrInvalid
	}
	input, e := t.Input.Encode(i)
	if e != nil {
		return result, e
	}
	wire := Request{
		Version:     1,
		InputCodec:  t.Input.Identity(),
		OutputCodec: t.Output.Identity(),
		Trial:       Trial{tc.ID, tc.CaseRevision, tc.Seed, t.Fixture, t.Reset},
		Input:       input,
	}
	body, e := json.Marshal(wire)
	if e != nil || int64(len(body)) > t.MaxBytes {
		return result, evaly.ErrInvalid
	}
	req, e := http.NewRequestWithContext(ctx, http.MethodPost, t.URL, bytes.NewReader(body))
	if e != nil {
		return result, e
	}
	req.Header.Set("Content-Type", "application/json")
	resp, e := t.Client.Do(req)
	if e != nil {
		return result, e
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return result, fmt.Errorf("HTTP status %d: %w", resp.StatusCode, evaly.ErrUnsupported)
	}
	data, e := io.ReadAll(io.LimitReader(resp.Body, t.MaxBytes+1))
	if e != nil {
		return result, e
	}
	if int64(len(data)) > t.MaxBytes {
		return result, evaly.ErrInvalid
	}
	if _, e = evaly.CanonicalJSON(data); e != nil {
		return result, e
	}
	var out Response
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if e = d.Decode(&out); e != nil {
		return result, evaly.ErrInvalid
	}
	result.Usage = out.Usage
	if out.Version != 1 || !evaly.CheckMapping(out.Capabilities).Supported {
		return result, evaly.ErrUnsupported
	}
	if out.Status != "completed" {
		return result, evaly.ErrUnsupported
	}
	if tc.Evidence == nil {
		return result, evaly.ErrInvalid
	}
	for _, event := range out.Events {
		if e = tc.Evidence.Record(ctx, event); e != nil {
			return result, e
		}
	}
	result.Output, e = t.Output.Decode(out.Output)
	return result, e
}

// Handler provides a runnable server-side reference for the JSON protocol.
// Invoke owns environment isolation and must check the declared fixture/reset.
func Handler[I, O any](
	ic evaly.Codec[I],
	oc evaly.Codec[O],
	maxBytes int64,
	invoke func(context.Context, I, Trial) (O, evaly.Usage, []evaly.Event, error),
) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fail := func(code int) { http.Error(w, "invalid evaly request", code) }
		if ic == nil || oc == nil || invoke == nil || maxBytes <= 0 || r.Method != http.MethodPost ||
			!strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
			fail(http.StatusBadRequest)
			return
		}
		b, e := io.ReadAll(io.LimitReader(r.Body, maxBytes+1))
		if e != nil || int64(len(b)) > maxBytes {
			fail(http.StatusRequestEntityTooLarge)
			return
		}
		if _, e = evaly.CanonicalJSON(b); e != nil {
			fail(http.StatusBadRequest)
			return
		}
		var req Request
		decoder := json.NewDecoder(bytes.NewReader(b))
		decoder.DisallowUnknownFields()
		if e = decoder.Decode(&req); e != nil {
			fail(http.StatusBadRequest)
			return
		}
		if req.Version != 1 || req.InputCodec != ic.Identity() || req.OutputCodec != oc.Identity() ||
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
		o, usage, events, e := invoke(r.Context(), i, req.Trial)
		resp := Response{
			Version: 1,
			Status:  "completed",
			Usage:   usage,
			Events:  events,
			Capabilities: evaly.InteropCapabilities{
				Version:       1,
				Outcome:       true,
				ResetIdentity: true,
				Evidence:      true,
				RichStatus:    true,
				MetricScales:  true,
			},
		}
		if e != nil {
			resp.Status = "target_error"
		} else {
			resp.Output, e = oc.Encode(o)
			if e != nil {
				resp.Status = "target_error"
			}
		}
		data, e := json.Marshal(resp)
		if e != nil || int64(len(data)) > maxBytes {
			fail(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(data)
	})
}
