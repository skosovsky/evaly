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
	Output   O                `json:"Output"`
	Usage    evaly.Usage      `json:"Usage"`
	Events   []evaly.Event    `json:"Events"`
	Evidence EvidenceDelivery `json:"Evidence"`
}
type Response struct {
	Version  int              `json:"version"`
	Status   string           `json:"status"`
	Output   json.RawMessage  `json:"output,omitempty"`
	Usage    evaly.Usage      `json:"usage"`
	Events   []evaly.Event    `json:"events"`
	Evidence EvidenceDelivery `json:"evidence"`
}
type Doer interface {
	Do(*http.Request) (*http.Response, error)
}
type Target[I, O, E any] struct {
	URL      string
	Client   Doer
	Input    evaly.Codec[I]
	Output   evaly.Codec[O]
	MaxBytes int64
	Fixture  string
	Reset    string
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
	if err := ctx.Err(); err != nil {
		return result, err
	}
	req, e := t.request(ctx, i, tc)
	if e != nil {
		return result, e
	}
	incomplete := tc.Evidence.MarkIncomplete
	if eLocal := ctx.Err(); eLocal != nil {
		return result, eLocal
	}
	resp, e := t.Client.Do(req)
	if e != nil {
		incomplete()
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
		return result, e
	}
	if resp == nil || evaly.ValidatePort(resp.Body) != nil {
		incomplete()
		return result, evaly.ErrInvalid
	}
	defer resp.Body.Close()
	return t.consume(ctx, tc, resp)
}

// NewHandler constructs the reference v3 server after local configuration checks.
// Codecs must keep their validated identities and behavior stable for its lifetime.
// Invoke owns isolation, fixture/reset compatibility and evidence completeness.
func NewHandler[I, O any](ic evaly.Codec[I], oc evaly.Codec[O], maxBytes int64,
	invoke func(context.Context, I, Trial) (Invocation[O], error)) (http.Handler, error) {
	if invoke == nil || maxBytes <= 0 || maxBytes == math.MaxInt64 {
		return nil, evaly.ErrInvalid
	}
	for _, port := range []any{ic, oc} {
		if err := evaly.ValidatePort(port); err != nil {
			return nil, err
		}
	}
	inputIdentity, outputIdentity := ic.Identity(), oc.Identity()
	for _, identity := range []evaly.CodecIdentity{inputIdentity, outputIdentity} {
		if err := evaly.ValidateCodecIdentity(identity); err != nil {
			return nil, err
		}
	}
	return &referenceHandler[I, O]{
		input:          ic,
		output:         oc,
		maxBytes:       maxBytes,
		inputIdentity:  inputIdentity,
		outputIdentity: outputIdentity,
		invoke:         invoke,
	}, nil
}

type referenceHandler[I, O any] struct {
	input          evaly.Codec[I]
	output         evaly.Codec[O]
	maxBytes       int64
	inputIdentity  evaly.CodecIdentity
	outputIdentity evaly.CodecIdentity
	invoke         func(context.Context, I, Trial) (Invocation[O], error)
}

func (h *referenceHandler[I, O]) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fail := func(code int) { http.Error(w, "invalid evaly request", code) }
	if r.Context().Err() != nil {
		fail(http.StatusRequestTimeout)
		return
	}
	if r.Method != http.MethodPost || !jsonContentType(r.Header.Get("Content-Type")) ||
		evaly.ValidatePort(r.Body) != nil {
		fail(http.StatusBadRequest)
		return
	}
	b, err := io.ReadAll(io.LimitReader(r.Body, h.maxBytes+1))
	if r.Context().Err() != nil {
		fail(http.StatusRequestTimeout)
		return
	}
	if err != nil {
		fail(http.StatusBadRequest)
		return
	}
	if int64(len(b)) > h.maxBytes {
		fail(http.StatusRequestEntityTooLarge)
		return
	}
	req, err := evaly.DecodeWire[Request](b)
	if err != nil {
		fail(http.StatusBadRequest)
		return
	}
	if req.InputCodec != h.inputIdentity || req.OutputCodec != h.outputIdentity || req.Trial.ID == "" ||
		req.Trial.CaseRevision == "" || req.Trial.Fixture == "" || req.Trial.Reset == "" {
		fail(http.StatusUnprocessableEntity)
		return
	}
	if r.Context().Err() != nil {
		fail(http.StatusRequestTimeout)
		return
	}
	input, err := h.input.Decode(req.Input)
	if err != nil {
		fail(http.StatusBadRequest)
		return
	}
	if r.Context().Err() != nil {
		fail(http.StatusRequestTimeout)
		return
	}
	data, err := encodeInvocation(r.Context(), input, req.Trial, h.output, h.maxBytes, h.invoke)
	if err != nil {
		fail(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func (t Target[I, O, E]) request(ctx context.Context, i I, tc evaly.TrialContext[E]) (*http.Request, error) {
	body, e := t.encodeRequest(i, tc)
	if e != nil {
		return nil, e
	}
	req, e := http.NewRequestWithContext(ctx, http.MethodPost, t.URL, bytes.NewReader(body))
	if e != nil {
		return nil, e
	}
	req.Header.Set("Content-Type", "application/json")
	return req, nil
}

func (t Target[I, O, E]) consume(
	ctx context.Context,
	tc evaly.TrialContext[E],
	resp *http.Response,
) (evaly.TargetResult[O], error) {
	var result evaly.TargetResult[O]
	incomplete := tc.Evidence.MarkIncomplete
	out, e := t.readResponse(resp, incomplete)
	if e != nil {
		return result, e
	}
	if e = t.validateResponse(out, incomplete); e != nil {
		return result, e
	}
	result.Usage = out.Usage
	for _, event := range out.Events {
		if eLocal := tc.Evidence.Record(ctx, event); eLocal != nil {
			incomplete()
			return result, eLocal
		}
	}
	if !out.Evidence.Complete {
		incomplete()
	}
	if out.Status == valueTargetError {
		return result, evaly.ErrTarget
	}
	result.Output, e = t.Output.Decode(out.Output)
	return result, e
}

func encodeInvocation[I, O any](
	ctx context.Context,
	i I,
	trial Trial,
	oc evaly.Codec[O],
	maxBytes int64,
	invoke func(context.Context, I, Trial) (Invocation[O], error),
) ([]byte, error) {
	var e error
	invocation, invokeErr := invoke(ctx, i, trial)
	if invocation.Evidence.Validate() != nil {
		return nil, evaly.ErrInvalid
	}
	resp := Response{
		Version:  wireRevision,
		Status:   completedState,
		Usage:    invocation.Usage,
		Events:   invocation.Events,
		Evidence: invocation.Evidence,
		Output:   nil,
	}
	if invokeErr != nil {
		resp.Status = valueTargetError
	} else {
		resp.Output, e = oc.Encode(invocation.Output)
		if e != nil {
			resp.Status = valueTargetError
			resp.Output = nil
		}
	}
	data, e := json.Marshal(resp)
	if e != nil || int64(len(data)) > maxBytes {
		return nil, evaly.ErrInvalid
	}
	if _, e := evaly.DecodeWire[Response](data); e != nil {
		return nil, evaly.ErrInvalid
	}
	return data, nil
}

func (t Target[I, O, E]) readResponse(resp *http.Response, incomplete func()) (Response, error) {
	if resp.StatusCode != http.StatusOK {
		incomplete()
		return Response{}, fmt.Errorf("HTTP status %d: %w", resp.StatusCode, evaly.ErrUnsupported)
	}
	if !jsonContentType(resp.Header.Get("Content-Type")) {
		incomplete()
		return Response{}, evaly.ErrInvalid
	}
	data, e := io.ReadAll(io.LimitReader(resp.Body, t.MaxBytes+1))
	if e != nil {
		incomplete()
		return Response{}, e
	}
	if int64(len(data)) > t.MaxBytes {
		incomplete()
		return Response{}, evaly.ErrInvalid
	}
	out, e := evaly.DecodeWire[Response](data)
	if e != nil {
		incomplete()
		return Response{}, e
	}
	return out, nil
}

func (t Target[I, O, E]) validateResponse(out Response, incomplete func()) error {
	if out.Version != wireRevision ||
		out.Status != completedState && out.Status != valueTargetError {
		incomplete()
		return evaly.ErrUnsupported
	}
	if eLocal := out.Evidence.Validate(); eLocal != nil {
		incomplete()
		return eLocal
	}
	if out.Status == completedState && len(out.Output) == 0 || out.Status == valueTargetError && len(out.Output) != 0 {
		incomplete()
		return evaly.ErrInvalid
	}
	return nil
}

func (t Target[I, O, E]) encodeRequest(i I, tc evaly.TrialContext[E]) ([]byte, error) {
	input, e := t.Input.Encode(i)
	if e != nil {
		return nil, e
	}
	wire := Request{
		Version:     wireRevision,
		InputCodec:  t.Input.Identity(),
		OutputCodec: t.Output.Identity(),
		Trial:       Trial{tc.ID, tc.CaseRevision, tc.Seed, t.Fixture, t.Reset},
		Input:       input,
	}
	body, e := json.Marshal(wire)
	if e != nil || int64(len(body)) > t.MaxBytes {
		return nil, evaly.ErrInvalid
	}
	if _, e = evaly.DecodeWire[Request](body); e != nil {
		return nil, e
	}
	return body, nil
}

func jsonContentType(header string) bool {
	media, _, err := mime.ParseMediaType(header)
	return err == nil && media == "application/json"
}
