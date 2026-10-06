package optimizer

import (
	"math"
	"math/big"

	"github.com/skosovsky/evaly"
)

// proposalUsage sums finite binary receipts exactly, rounding upward only once.
func proposalUsage(rounds []Round) (evaly.Usage, error) {
	total := new(big.Rat)
	known := len(rounds) > 0
	for _, round := range rounds {
		if math.IsNaN(round.Usage.Units) || math.IsInf(round.Usage.Units, 0) || round.Usage.Units < 0 {
			return evaly.Usage{}, evaly.ErrInvalid
		}
		total.Add(total, new(big.Rat).SetFloat64(round.Usage.Units))
		known = known && round.Usage.Known
	}
	if total.Cmp(new(big.Rat).SetFloat64(math.MaxFloat64)) > 0 {
		return evaly.Usage{}, evaly.ErrInvalid
	}
	units, exact := total.Float64()
	if !exact && new(big.Rat).SetFloat64(units).Cmp(total) < 0 {
		units = math.Nextafter(units, math.Inf(1))
	}
	return evaly.Usage{Known: known, Units: units}, nil
}
