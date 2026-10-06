package optimizer

import (
	"errors"
	"math"
	"testing"

	"github.com/skosovsky/evaly"
)

func TestProposalUsageConservativeSumAndKnownConjunction(t *testing.T) {
	for _, receipts := range [][]float64{{1e16, 1, 1}, {1, 1, 1e16}, {math.SmallestNonzeroFloat64, 1}} {
		// Arrange: finite receipts at different scales.
		rounds := make([]Round, 0, len(receipts)+1)
		for _, units := range receipts {
			rounds = append(rounds, Round{Usage: evaly.Usage{Known: true, Units: units}})
		}
		// Act.
		usage, err := proposalUsage(rounds)
		// Assert: small costs survive the aggregate, including subnormals.
		want := 1e16 + 2
		if len(receipts) == 2 {
			want = math.Nextafter(1, math.Inf(1))
		}
		if err != nil || !usage.Known || usage.Units != want {
			t.Fatal(err, usage, want)
		}
		rounds = append(rounds, Round{Usage: evaly.Usage{Known: false, Units: 0}})
		// Act / Assert: a no-dispatch round participates in Known without altering units.
		usage, err = proposalUsage(rounds)
		if err != nil || usage.Known || usage.Units != want {
			t.Fatal(err, usage)
		}
	}
	// Arrange / Act / Assert: no rounds and representability boundaries.
	empty, err := proposalUsage(nil)
	if err != nil || empty != (evaly.Usage{}) {
		t.Fatal(err, empty)
	}
	_, err = proposalUsage(
		[]Round{{Usage: evaly.Usage{Units: math.MaxFloat64}}, {Usage: evaly.Usage{Units: math.SmallestNonzeroFloat64}}},
	)
	if !errors.Is(err, evaly.ErrInvalid) {
		t.Fatal(err)
	}
}

func FuzzProposalUsageRetainsSmallLiability(f *testing.F) {
	f.Add(uint16(1), uint16(1), true)
	f.Add(uint16(100), uint16(7), false)
	f.Fuzz(func(t *testing.T, scale, amount uint16, known bool) {
		// Arrange: positive exact binary costs separated by at least 54 exponents.
		exponent := int(scale%900) - 450
		bigCost := math.Ldexp(1, exponent)
		smallCost := math.Ldexp(float64(amount)+1, exponent-70)
		// Act: order must not alter the exact sum or reported Known bit.
		first, err1 := proposalUsage(
			[]Round{
				{Usage: evaly.Usage{Known: true, Units: bigCost}},
				{Usage: evaly.Usage{Known: known, Units: smallCost}},
			},
		)
		second, err2 := proposalUsage(
			[]Round{
				{Usage: evaly.Usage{Known: known, Units: smallCost}},
				{Usage: evaly.Usage{Known: true, Units: bigCost}},
			},
		)
		// Assert: any positive liability below one ulp still rounds conservatively upward.
		if err1 != nil || err2 != nil || first != second || first.Known != known ||
			first.Units != math.Nextafter(bigCost, math.Inf(1)) {
			t.Fatal(err1, err2, first, second)
		}
	})
}
