package evaly

import (
	"math"
	"testing"
)

func TestMeanStepExtremeFiniteValues(t *testing.T) {
	// Arrange.
	extremes := []float64{1.7e308, -1.7e308}
	tiny := math.SmallestNonzeroFloat64
	// Act.
	balanced := meanStep(extremes[0], extremes[1], 2)
	same := meanStep(tiny, tiny, 2)
	interval := bootstrap(extremes, 100, 1)
	// Assert.
	if balanced != 0 || same != tiny || math.IsInf(interval.Lower, 0) || math.IsInf(interval.Upper, 0) ||
		math.IsNaN(interval.Lower) ||
		math.IsNaN(interval.Upper) {
		t.Fatal(balanced, same, interval)
	}
}
