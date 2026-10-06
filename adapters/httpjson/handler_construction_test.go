package httpjson_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/skosovsky/evaly"
	"github.com/skosovsky/evaly/adapters/httpjson"
	"github.com/skosovsky/evaly/internal/fixtures"
)

type handlerCalls struct{ identity, decode, encode, invoke, read int }
type handlerCodec struct {
	base     evaly.JSONCodec[int]
	calls    *handlerCalls
	invalid  error
	onDecode func()
}

func (c *handlerCodec) Validate() error { return c.invalid }
func (c *handlerCodec) Identity() evaly.CodecIdentity {
	c.calls.identity++
	return c.base.Identity()
}
func (c *handlerCodec) Encode(value int) ([]byte, error) {
	c.calls.encode++
	return c.base.Encode(value)
}
func (c *handlerCodec) Decode(bytes []byte) (int, error) {
	c.calls.decode++
	if c.onDecode != nil {
		c.onDecode()
	}
	return c.base.Decode(bytes)
}
func probeHandlerCodec(calls *handlerCalls) *handlerCodec {
	return &handlerCodec{
		base:     evaly.JSONCodec[int]{ID: "integer", Version: "1"},
		calls:    calls,
		invalid:  nil,
		onDecode: nil,
	}
}
func probeInvocation(calls *handlerCalls) func(context.Context, int, httpjson.Trial) (httpjson.Invocation[int], error) {
	return func(context.Context, int, httpjson.Trial) (httpjson.Invocation[int], error) {
		calls.invoke++
		return httpjson.Invocation[int]{Output: 1, Evidence: httpjson.EvidenceDelivery{Complete: true}}, nil
	}
}

func TestNewHandlerRejectsStaticConfigurationWithoutDispatch(t *testing.T) {
	for _, variant := range []string{"nil_input", "nil_output", "typed_nil_input", "typed_nil_output", "validator_input", "validator_output", "identity_input", "identity_output", "nil_callback", "zero_limit", "negative_limit", "overflow_limit"} {
		t.Run(variant, func(t *testing.T) {
			// Arrange: valid peers count any accidental capability dispatch.
			calls := &handlerCalls{}
			left, right := probeHandlerCodec(calls), probeHandlerCodec(calls)
			var input, output evaly.Codec[int] = left, right
			invoke := probeInvocation(calls)
			maximum := int64(4096)
			validatorErr := errors.New("local codec invalid")
			switch variant {
			case "nil_input":
				input = nil
			case "nil_output":
				output = nil
			case "typed_nil_input":
				var absent *handlerCodec
				input = absent
			case "typed_nil_output":
				var absent *handlerCodec
				output = absent
			case "validator_input":
				left.invalid = validatorErr
			case "validator_output":
				right.invalid = validatorErr
			case "identity_input":
				left.base.ID = ""
			case "identity_output":
				right.base.Version = ""
			case "nil_callback":
				invoke = nil
			case "zero_limit":
				maximum = 0
			case "negative_limit":
				maximum = -1
			case "overflow_limit":
				maximum = math.MaxInt64
			}
			// Act.
			handler, err := httpjson.NewHandler(input, output, maximum, invoke)
			// Assert: setup failure yields no usable handler and preserves validator errors.
			want := evaly.ErrInvalid
			if strings.HasPrefix(variant, "validator") {
				want = validatorErr
			}
			if handler != nil || !errors.Is(err, want) || calls.invoke != 0 || calls.encode != 0 || calls.decode != 0 {
				t.Fatal(err, calls, handler)
			}
			if !strings.HasPrefix(variant, "identity") && calls.identity != 0 {
				t.Fatal("identity called before peer structure validated", calls)
			}
		})
	}
}

func intRequestBody(t *testing.T) string {
	t.Helper()
	wire := httpjson.Request{
		Version:     3,
		InputCodec:  evaly.CodecIdentity{ID: "integer", Version: "1"},
		OutputCodec: evaly.CodecIdentity{ID: "integer", Version: "1"},
		Trial:       httpjson.Trial{ID: "trial", CaseRevision: "case", Seed: 0, Fixture: "fixture", Reset: "reset"},
		Input:       json.RawMessage("0"),
	}
	bytes, err := json.Marshal(wire)
	if err != nil {
		t.Fatal(err)
	}
	return string(bytes)
}

type handlerBody struct {
	source io.ReadCloser
	calls  *handlerCalls
	cancel context.CancelFunc
}

func (b handlerBody) Close() error { return b.source.Close() }
func (b handlerBody) Read(bytes []byte) (int, error) {
	b.calls.read++
	count, err := b.source.Read(bytes)
	if b.cancel != nil {
		b.cancel()
	}
	return count, err
}

func TestHandlerCancellationGuardsAllDispatchBoundaries(t *testing.T) {
	for _, stage := range []string{"before_request", "after_read", "after_decode"} {
		t.Run(stage, func(t *testing.T) {
			checkHandlerCancellation(t, stage)
		})
	}
}

func TestHandlerRuntimeRequestErrorsRemainClassified(t *testing.T) {
	for _, variant := range []string{"malformed", "foreign_codec", "oversized", "old_version", "legacy_capabilities"} {
		t.Run(variant, func(t *testing.T) {
			// Arrange: static configuration is valid; only the incoming wire is invalid.
			calls := &handlerCalls{}
			handler, err := httpjson.NewHandler(
				probeHandlerCodec(calls),
				probeHandlerCodec(calls),
				4096,
				probeInvocation(calls),
			)
			if err != nil {
				t.Fatal(err)
			}
			body := intRequestBody(t)
			status := http.StatusBadRequest
			switch variant {
			case "malformed":
				body = "{"
			case "foreign_codec":
				body = strings.Replace(body, "integer", "foreign", 1)
				status = http.StatusUnprocessableEntity
			case "oversized":
				body = strings.Repeat(" ", 4097)
				status = http.StatusRequestEntityTooLarge
			case "old_version":
				body = strings.Replace(body, `"version":3`, `"version":2`, 1)
			case "legacy_capabilities":
				body = strings.TrimSuffix(body, "}") + `,"capabilities":{}}`
			}
			request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
			request.Header.Set("Content-Type", "application/json")
			recorder := httptest.NewRecorder()
			// Act.
			handler.ServeHTTP(recorder, request)
			// Assert: request failures remain distinct from constructor failures.
			if recorder.Code != status || calls.invoke != 0 {
				t.Fatal(recorder.Code, status, calls, recorder.Body.String())
			}
		})
	}
}

func checkHandlerCancellation(t *testing.T, stage string) {
	t.Helper()
	// Arrange: cancel at a deterministic boundary before Invoke.
	calls := &handlerCalls{}
	input, output := probeHandlerCodec(calls), probeHandlerCodec(calls)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var cancelRead context.CancelFunc
	if stage == "before_request" {
		cancel()
	}
	if stage == "after_read" {
		cancelRead = cancel
	}
	if stage == "after_decode" {
		input.onDecode = cancel
	}
	handler, err := httpjson.NewHandler(input, output, 4096, probeInvocation(calls))
	if err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequestWithContext(ctx, http.MethodPost, "/", strings.NewReader(intRequestBody(t)))
	request.Body = handlerBody{source: request.Body, calls: calls, cancel: cancelRead}
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	// Act.
	handler.ServeHTTP(recorder, request)
	// Assert: known cancellation never starts Invoke/output encoding.
	verifyCancellationBoundary(t, stage, calls, recorder.Code)
}

func verifyCancellationBoundary(t *testing.T, stage string, calls *handlerCalls, status int) {
	t.Helper()
	wantDecode := 0
	if stage == "after_decode" {
		wantDecode = 1
	}
	if status != http.StatusRequestTimeout || calls.invoke != 0 || calls.encode != 0 || calls.decode != wantDecode {
		t.Fatal(status, calls)
	}
	if stage == "before_request" && calls.read != 0 {
		t.Fatal(calls)
	}
}

func TestClientRejectsLegacyCapabilitiesInV3Response(t *testing.T) {
	// Arrange: a valid v3 response shape with a legacy capability field added.
	response := `{"version":3,"status":"completed","output":{"sum":3},"usage":{"known":true,"units":1},"events":[],"evidence":{"complete":true,"reason":""},"capabilities":{}}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, response)
	}))
	defer server.Close()
	capture := newCapture(t)
	// Act.
	result, err := target(server).Run(context.Background(), fixtures.Calculation{}, trial(capture))
	evidence := capture.Seal()
	// Assert: obsolete certification flags do not enter the trusted v3 envelope.
	if !errors.Is(err, evaly.ErrInvalid) || result.Usage.Known || evidence.State != "incomplete" {
		t.Fatal(err, result, evidence)
	}
}
