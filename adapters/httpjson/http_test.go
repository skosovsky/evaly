package httpjson_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/skosovsky/evaly"
	"github.com/skosovsky/evaly/adapters/httpjson"
	"github.com/skosovsky/evaly/conformance"
	"github.com/skosovsky/evaly/internal/fixtures"
)

func TestHTTPReferenceConformance(t *testing.T) {
	// Arrange.
	handler, handlerErr := httpjson.NewHandler(
		fixtures.InputCodec(),
		fixtures.OutputCodec(),
		4096,
		func(ctx context.Context, i fixtures.Calculation, trial httpjson.Trial) (httpjson.Invocation[fixtures.CalculationOutput], error) {
			if trial.Fixture != "calculation-v1" || trial.Reset != "empty-v1" {
				return httpjson.Invocation[fixtures.CalculationOutput]{
					Evidence: httpjson.EvidenceDelivery{Complete: true},
				}, evaly.ErrUnsupported
			}
			return httpjson.Invocation[fixtures.CalculationOutput]{
				Output:   fixtures.CalculationOutput{Sum: i.Left + i.Right},
				Usage:    evaly.Usage{Known: true, Units: 1},
				Evidence: httpjson.EvidenceDelivery{Complete: true},
			}, ctx.Err()
		},
	)
	if handlerErr != nil {
		t.Fatal(handlerErr)
	}
	server := httptest.NewServer(handler)
	defer server.Close()
	c, e := fixtures.CalculationConfig("http-conformance", "good", "")
	if e != nil {
		t.Fatal(e)
	}
	c.Target = httpjson.Target[fixtures.Calculation, fixtures.CalculationOutput, *fixtures.Environment]{
		URL:      server.URL,
		Client:   server.Client(),
		Input:    fixtures.InputCodec(),
		Output:   fixtures.OutputCodec(),
		MaxBytes: 4096,
		Fixture:  "calculation-v1",
		Reset:    "empty-v1",
	}
	// Act / Assert using the same suite as the local target.
	conformance.Target(t, c)
}
func TestHTTPRejectsOldAndUnknownVersion(t *testing.T) {
	for _, version := range []int{1, 2, 99} {
		t.Run(string(rune('0'+version)), func(t *testing.T) {
			// Arrange.
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).
					Encode(httpjson.Response{Version: version, Status: "completed", Output: json.RawMessage(`{"sum":3}`), Evidence: httpjson.EvidenceDelivery{Complete: true}})
			}))
			defer server.Close()
			c, e := fixtures.CalculationConfig("http-loss", "good", "")
			if e != nil {
				t.Fatal(e)
			}
			c.Target = httpjson.Target[fixtures.Calculation, fixtures.CalculationOutput, *fixtures.Environment]{
				URL:      server.URL,
				Client:   server.Client(),
				Input:    fixtures.InputCodec(),
				Output:   fixtures.OutputCodec(),
				MaxBytes: 4096,
				Fixture:  "calculation-v1",
				Reset:    "empty-v1",
			}
			// Act.
			result, e := evaly.Run(context.Background(), c)
			// Assert.
			if e != nil {
				t.Fatal(e)
			}
			for _, trial := range result.Record().Trials {
				if trial.Status != evaly.TargetError {
					t.Fatal("loss became success", trial)
				}
			}
		})
	}
}
