package main

import (
	"context"
	"fmt"
	"log"
	"net/http/httptest"

	"github.com/skosovsky/evaly"
	"github.com/skosovsky/evaly/adapters/httpjson"
	"github.com/skosovsky/evaly/internal/fixtures"
)

func main() {
	c, e := fixtures.CalculationConfig("http-example", "good", "")
	if e != nil {
		log.Fatal(e)
	}
	server := httptest.NewServer(
		httpjson.Handler(
			fixtures.InputCodec(),
			fixtures.OutputCodec(),
			4096,
			func(ctx context.Context, i fixtures.Calculation, t httpjson.Trial) (httpjson.Invocation[fixtures.CalculationOutput], error) {
				if t.Fixture != "calculation-v1" || t.Reset != "empty-v1" {
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
		),
	)
	defer server.Close()
	c.Target = httpjson.Target[fixtures.Calculation, fixtures.CalculationOutput, *fixtures.Environment]{
		URL:      server.URL,
		Client:   server.Client(),
		Input:    fixtures.InputCodec(),
		Output:   fixtures.OutputCodec(),
		MaxBytes: 4096,
		Fixture:  "calculation-v1",
		Reset:    "empty-v1",
	}
	result, e := evaly.Run(context.Background(), c)
	if e != nil {
		log.Fatal(e)
	}
	fmt.Printf("HTTP trials: %d\n", len(result.Record().Trials))
}
