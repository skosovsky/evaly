package evaly

import (
	"math"
	"reflect"
	"testing"
)

func TestBootstrapPCGRevisionAndSignedSeeds(t *testing.T) {
	// Arrange: exercise the entire signed seed domain and representable value boundaries.
	const samples = 128
	cases := map[string][]float64{
		"native":            {-5, 0, 7, 11},
		"opposite_extremes": {-math.MaxFloat64, math.MaxFloat64},
		"subnormal":         {math.SmallestNonzeroFloat64, math.SmallestNonzeroFloat64},
		"one_case":          {7},
	}
	for name, values := range cases {
		t.Run(name, func(t *testing.T) {
			for _, seed := range []int64{0, -1, math.MinInt64, math.MaxInt64} {
				// Act: sampling is local and repeatable even across intervening different seeds.
				first := bootstrap(values, samples, seed)
				bootstrap(values, samples, seed^1)
				second := bootstrap(values, samples, seed)
				// Assert: the version, independent unit and numerical bounds are explicit.
				if !reflect.DeepEqual(first, second) || first.Method != "paired_case_bootstrap_percentile_pcg_v2" ||
					first.Unit != "case" {
					t.Fatalf("seed %d: first=%+v second=%+v", seed, first, second)
				}
				assertBootstrapBounds(t, first, values)
			}
		})
	}
}

func TestPairedScheduleSignedSeedVectorsRemainV1(t *testing.T) {
	// Arrange: reference vectors were computed from the prior modular SplitMix64 implementation.
	vectors := []struct {
		seed  int64
		first string
	}{
		{0, "CBCBCBCB"},
		{-1, "BCCBBCCB"},
		{math.MinInt64, "CBBCCCCC"},
		{math.MaxInt64, "CCBCCCCB"},
	}
	for _, vector := range vectors {
		for ordinal, want := range vector.first {
			// Act.
			first, second := pairFirst(vector.seed, ordinal, "B", "C")
			// Assert: both order entries and every seed boundary retain the previous result.
			if first != string(want) || first == second || (second != "B" && second != "C") {
				t.Fatalf("seed=%d ordinal=%d: %s/%s", vector.seed, ordinal, first, second)
			}
		}
	}
}

func assertBootstrapBounds(t *testing.T, first *Interval, values []float64) {
	t.Helper()
	if math.IsNaN(first.Lower) || math.IsInf(first.Lower, 0) || math.IsNaN(first.Upper) ||
		math.IsInf(first.Upper, 0) ||
		first.Lower > first.Upper {
		t.Fatalf("non-finite or reversed interval: %+v", first)
	}
	minimum, maximum := values[0], values[0]
	for _, value := range values {
		minimum = min(minimum, value)
		maximum = max(maximum, value)
	}
	if first.Lower < minimum || first.Upper > maximum {
		t.Fatalf("interval outside sample range: %+v", first)
	}
}
