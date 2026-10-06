package httpjson_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/skosovsky/evaly"
	"github.com/skosovsky/evaly/adapters/httpjson"
	"github.com/skosovsky/evaly/internal/fixtures"
)

func newCapture(t *testing.T) *evaly.Capture {
	t.Helper()
	capture, err := evaly.NewCapture(
		evaly.CaptureConfig{
			Policy:        evaly.FieldPolicy{ID: "http-test", Allowed: map[string][]string{"tool": {"name"}}},
			KnownKinds:    []string{"tool"},
			RequiredKinds: []string{"tool"},
			MaxEvents:     10,
			MaxBytes:      4096,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	return capture
}
func target(server *httptest.Server) httpjson.Target[fixtures.Calculation, fixtures.CalculationOutput, struct{}] {
	return httpjson.Target[fixtures.Calculation, fixtures.CalculationOutput, struct{}]{
		URL:      server.URL,
		Client:   server.Client(),
		Input:    fixtures.InputCodec(),
		Output:   fixtures.OutputCodec(),
		MaxBytes: 4096,
		Fixture:  "calculation-v1",
		Reset:    "empty-v1",
	}
}
func trial(capture *evaly.Capture) evaly.TrialContext[struct{}] {
	return evaly.TrialContext[struct{}]{ID: "http-trial", CaseRevision: "case-v1", Evidence: capture}
}
func toolEvent() evaly.Event {
	return evaly.Event{
		Version:       1,
		Sequence:      1,
		Kind:          "tool",
		CorrelationID: "effect-v1",
		Payload:       json.RawMessage(`{"name":"write","secret":"omitted"}`),
	}
}

func TestTargetFailurePreservesUsageAndPolicyControlledEffects(t *testing.T) {
	for _, complete := range []bool{true, false} {
		t.Run(map[bool]string{true: "complete", false: "incomplete"}[complete], func(t *testing.T) {
			checkTargetFailurePreservesUsageAndPolicyControlledEffects(t, &complete)
		},
		)
	}
}

func TestHandlerRequiresExplicitSeedBeforeInvoke(t *testing.T) {
	for _, test := range []struct {
		name, seed string
		wantCalls  int
	}{{"missing", "", 0}, {"null", `,"seed":null`, 0}, {"zero", `,"seed":0`, 1}, {"duplicate", `,"seed":0,"seed":1`, 0}} {
		t.Run(test.name, func(t *testing.T) {
			// Arrange.
			calls := 0
			handler, handlerErr := httpjson.NewHandler(
				fixtures.InputCodec(),
				fixtures.OutputCodec(),
				4096,
				func(context.Context, fixtures.Calculation, httpjson.Trial) (httpjson.Invocation[fixtures.CalculationOutput], error) {
					calls++
					return httpjson.Invocation[fixtures.CalculationOutput]{
						Output:   fixtures.CalculationOutput{Sum: 3},
						Evidence: httpjson.EvidenceDelivery{Complete: true},
					}, nil
				},
			)
			if handlerErr != nil {
				t.Fatal(handlerErr)
			}
			body := `{"version":3,"input_codec":{"id":"` + fixtures.InputCodec().
				Identity().
				ID + `","version":"` + fixtures.InputCodec().
				Identity().
				Version + `"},"output_codec":{"id":"` + fixtures.OutputCodec().
				Identity().
				ID + `","version":"` + fixtures.OutputCodec().
				Identity().
				Version + `"},"trial":{"id":"trial","case_revision":"case","fixture":"calculation-v1","reset":"empty-v1"` + test.seed + `},"input":{"left":1,"right":2}}`
			request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
			request.Header.Set("Content-Type", "application/json")
			recorder := httptest.NewRecorder()
			// Act.
			handler.ServeHTTP(recorder, request)
			// Assert.
			if calls != test.wantCalls {
				t.Fatalf("invoke calls=%d want=%d status=%d", calls, test.wantCalls, recorder.Code)
			}
			if test.wantCalls == 1 && recorder.Code != 200 || test.wantCalls == 0 && recorder.Code == 200 {
				t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
			}
		})
	}
}

func TestUndeliverableResponseNeverProvesAbsence(t *testing.T) {
	valid := httpjson.Response{
		Version:  3,
		Status:   "completed",
		Output:   json.RawMessage(`{"sum":3}`),
		Usage:    evaly.Usage{Known: true, Units: 7},
		Events:   []evaly.Event{toolEvent()},
		Evidence: httpjson.EvidenceDelivery{Complete: true},
	}
	encoded, _ := json.Marshal(valid)
	for _, test := range []struct {
		name, body string
		status     int
	}{{"truncated", string(encoded[:len(encoded)-3]), 200}, {"oversized", strings.Repeat(" ", 4097), 200}, {"malformed", `{"version":3,}`, 200}, {"old_version", strings.Replace(string(encoded), `"version":3`, `"version":1`, 1), 200}, {"unknown_status", strings.Replace(string(encoded), `"completed"`, `"unknown"`, 1), 200}, {"server_error", string(encoded), 500}, {"missing_evidence", strings.Replace(string(encoded), `,"evidence":{"complete":true,"reason":""}`, "", 1), 200}} {
		t.Run(test.name, func(t *testing.T) {
			// Arrange.
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(test.status)
				_, _ = io.WriteString(w, test.body)
			}))
			defer server.Close()
			capture := newCapture(t)
			// Act.
			result, err := target(server).Run(context.Background(), fixtures.Calculation{}, trial(capture))
			evidence := capture.Seal()
			// Assert.
			if err == nil || evidence.State != "incomplete" || evidence.Coverage["tool"] || len(evidence.Events) != 0 ||
				result.Usage.Known {
				t.Fatalf("untrusted response became evidence: result=%+v evidence=%+v error=%v", result, evidence, err)
			}
		})
	}
}

type failedDoer struct{ err error }

func (f failedDoer) Do(*http.Request) (*http.Response, error) { return nil, f.err }
func TestTransportCancellationMarksEvidenceIncomplete(t *testing.T) {
	// Arrange.
	capture := newCapture(t)
	remote := httpjson.Target[fixtures.Calculation, fixtures.CalculationOutput, struct{}]{
		URL:      "http://example.invalid",
		Client:   failedDoer{context.DeadlineExceeded},
		Input:    fixtures.InputCodec(),
		Output:   fixtures.OutputCodec(),
		MaxBytes: 4096,
		Fixture:  "fixture",
		Reset:    "reset",
	}
	// Act.
	_, err := remote.Run(context.Background(), fixtures.Calculation{}, trial(capture))
	evidence := capture.Seal()
	// Assert.
	if !errors.Is(err, context.DeadlineExceeded) || evidence.State != "incomplete" || evidence.Coverage["tool"] {
		t.Fatalf("timeout became complete: %+v %v", evidence, err)
	}
}

func TestOutputDecodeFailureRetainsProvenCompleteEvidence(t *testing.T) {
	// Arrange.
	response := httpjson.Response{
		Version:  3,
		Status:   "completed",
		Output:   json.RawMessage(`"invalid-domain-output"`),
		Usage:    evaly.Usage{Known: true, Units: 4},
		Events:   []evaly.Event{toolEvent()},
		Evidence: httpjson.EvidenceDelivery{Complete: true},
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()
	capture := newCapture(t)
	// Act.
	result, err := target(server).Run(context.Background(), fixtures.Calculation{}, trial(capture))
	evidence := capture.Seal()
	// Assert.
	if err == nil || evidence.State != "sealed" || !evidence.Coverage["tool"] || len(evidence.Events) != 1 ||
		result.Usage.Units != 4 {
		t.Fatalf("output decode erased proven delivery: %+v %+v %v", result, evidence, err)
	}
}

type countingDoer struct{ calls int }

func (d *countingDoer) Do(*http.Request) (*http.Response, error) {
	d.calls++
	return nil, errors.New("must not dispatch")
}
func TestTargetPreflightAndCancellationAvoidHTTPDispatch(t *testing.T) {
	// Arrange.
	client := &countingDoer{}
	remote := httpjson.Target[fixtures.Calculation, fixtures.CalculationOutput, struct{}]{
		URL:      "http://example.invalid",
		Client:   client,
		Input:    fixtures.InputCodec(),
		Output:   fixtures.OutputCodec(),
		MaxBytes: 4096,
		Fixture:  "fixture",
		Reset:    "reset",
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	// Act.
	_, err := remote.Run(ctx, fixtures.Calculation{}, trial(newCapture(t)))
	// Assert.
	if !errors.Is(err, context.Canceled) || client.calls != 0 {
		t.Fatalf("cancelled target dispatched: calls=%d err=%v", client.calls, err)
	}
	remote.MaxBytes = int64(^uint64(0) >> 1)
	if remote.Validate() == nil || client.calls != 0 {
		t.Fatal("overflowing limit accepted or validation dispatched")
	}
	var absent *countingDoer
	remote.MaxBytes = 4096
	remote.Client = absent
	if remote.Validate() == nil {
		t.Fatal("typed nil HTTP capability accepted")
	}
}

func TestResponseContentTypeAndInvalidDeliveryAreUntrusted(t *testing.T) {
	for _, test := range []struct {
		name, contentType string
		delivery          httpjson.EvidenceDelivery
	}{{"missing_type", "", httpjson.EvidenceDelivery{Complete: true}}, {"wrong_type", "text/plain", httpjson.EvidenceDelivery{Complete: true}}, {"contradictory_delivery", "application/json", httpjson.EvidenceDelivery{Complete: true, Reason: "gap"}}, {"unspecified_delivery", "application/json", httpjson.EvidenceDelivery{}}} {
		t.Run(test.name, func(t *testing.T) {
			// Arrange.
			response := httpjson.Response{
				Version:  3,
				Status:   "target_error",
				Usage:    evaly.Usage{Known: true, Units: 4},
				Events:   []evaly.Event{toolEvent()},
				Evidence: test.delivery,
			}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header()["Content-Type"] = []string{test.contentType}
				_ = json.NewEncoder(w).Encode(response)
			}))
			defer server.Close()
			capture := newCapture(t)
			// Act.
			result, err := target(server).Run(context.Background(), fixtures.Calculation{}, trial(capture))
			evidence := capture.Seal()
			// Assert.
			if err == nil || evidence.State != "incomplete" || len(evidence.Events) != 0 || result.Usage.Known {
				t.Fatalf("untrusted protocol retained: %+v %+v %v", result, evidence, err)
			}
		})
	}
}

func TestEventRetentionFailurePreservesDeliveredPrefixAndUsage(t *testing.T) {
	// Arrange.
	second := toolEvent()
	second.Sequence = 2
	second.Kind = "unknown"
	response := httpjson.Response{
		Version:  3,
		Status:   "target_error",
		Usage:    evaly.Usage{Known: true, Units: 4},
		Events:   []evaly.Event{toolEvent(), second},
		Evidence: httpjson.EvidenceDelivery{Complete: true},
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()
	capture := newCapture(t)
	// Act.
	result, err := target(server).Run(context.Background(), fixtures.Calculation{}, trial(capture))
	evidence := capture.Seal()
	// Assert.
	if err == nil || evidence.State != "incomplete" || len(evidence.Events) != 1 || result.Usage.Units != 4 ||
		evidence.Coverage["tool"] {
		t.Fatalf("lost delivered prefix or completeness: %+v %+v %v", result, evidence, err)
	}
}

func checkTargetFailurePreservesUsageAndPolicyControlledEffects(
	t *testing.T,
	complete *bool,
) {
	t.Helper()
	// Arrange.
	delivery := httpjson.EvidenceDelivery{Complete: (*complete)}
	if !(*complete) {
		delivery.Reason = "upstream_gap"
	}
	handler, handlerErr := httpjson.NewHandler(
		fixtures.InputCodec(),
		fixtures.OutputCodec(),
		4096,
		func(context.Context, fixtures.Calculation, httpjson.Trial) (httpjson.Invocation[fixtures.CalculationOutput], error) {
			return httpjson.Invocation[fixtures.CalculationOutput]{
				Usage:    evaly.Usage{Known: true, Units: 7},
				Events:   []evaly.Event{toolEvent()},
				Evidence: delivery,
			}, errors.New(
				"domain failed after write",
			)
		},
	)
	if handlerErr != nil {
		t.Fatal(handlerErr)
	}
	server := httptest.NewServer(handler)
	defer server.Close()
	capture := newCapture(t)

	result, err := target(
		server,
	).Run(context.Background(), fixtures.Calculation{Left: 1, Right: 2}, trial(capture))
	evidence := capture.Seal()

	if !errors.Is(err, evaly.ErrTarget) || errors.Is(err, evaly.ErrUnsupported) ||
		result.Usage != (evaly.Usage{Known: true, Units: 7}) {
		t.Fatalf("lost target failure/usage: %+v %v", result, err)
	}
	if len(evidence.Events) != 1 || string(evidence.Events[0].Payload) != `{"name":"write"}` {
		t.Fatalf("lost or unfiltered effect: %+v", evidence)
	}
	wantState := "sealed"
	if !(*complete) {
		wantState = "incomplete"
	}
	if evidence.State != wantState || evidence.Coverage["tool"] != (*complete) {
		t.Fatalf("execution failure changed delivery: %+v", evidence)
	}
}
