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
	handler := httpjson.Handler(
		fixtures.InputCodec(),
		fixtures.OutputCodec(),
		4096,
		func(ctx context.Context, i fixtures.Calculation, trial httpjson.Trial) (fixtures.CalculationOutput, evaly.Usage, []evaly.Event, error) {
			if trial.Fixture != "calculation-v1" || trial.Reset != "empty-v1" {
				return fixtures.CalculationOutput{}, evaly.Usage{}, nil, evaly.ErrUnsupported
			}
			return fixtures.CalculationOutput{Sum: i.Left + i.Right}, evaly.Usage{Known: true, Units: 1}, nil, ctx.Err()
		},
	)
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
func TestHTTPRejectsLossAndUnknownVersion(t *testing.T) {
	for _, version := range []int{1, 2} {
		t.Run(string(rune('0'+version)), func(t *testing.T) {
			// Arrange.
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_ = json.NewEncoder(w).
					Encode(httpjson.Response{Version: version, Status: "completed", Output: json.RawMessage(`{"sum":3}`), Capabilities: evaly.InteropCapabilities{Version: 1, Outcome: false, ResetIdentity: true, Evidence: true, RichStatus: true, MetricScales: true}})
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
