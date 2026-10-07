package recipes

import (
	"math"
	"math/big"
	"slices"

	"github.com/skosovsky/prompty"

	"github.com/skosovsky/evaly"
)

// Conversion is immutable host policy. Rates convert source units to whole budget
// units, rounded upward. Bound is both an admission and a receipt bound.
type Conversion struct {
	Identity   string
	SourceUnit string
	Unit       string
	Mode       string // total or components
	TotalRate  float64
	InputRate  float64
	OutputRate float64
	Bound      float64
}

func (c Conversion) Validate() error {
	if c.SourceUnit != "token" {
		return evaly.ErrUnsupported
	}
	if c.Identity == "" || c.Unit == "" || !finite(c.Bound) {
		return evaly.ErrInvalid
	}
	switch c.Mode {
	case "total":
		if !positive(c.TotalRate) || c.InputRate != 0 || c.OutputRate != 0 {
			return evaly.ErrInvalid
		}
	case "components":
		if !positive(c.InputRate) || !positive(c.OutputRate) || c.TotalRate != 0 {
			return evaly.ErrInvalid
		}
	default:
		return evaly.ErrUnsupported
	}
	return nil
}

// Convert reads explicit presence only. Source updates must already be accumulated.
func (c Conversion) Convert(unit string, u prompty.Usage) (evaly.Usage, error) {
	if err := c.Validate(); err != nil {
		return evaly.Usage{}, err
	}
	if unit != c.SourceUnit || u.Validate() != nil || u.Mode == prompty.UsageDelta {
		return evaly.Usage{}, evaly.ErrInvalid
	}
	var amount big.Rat
	switch c.Mode {
	case "total":
		if !slices.Contains(u.Known, prompty.UsageTotal) {
			return evaly.Usage{}, nil
		}
		amount.Mul(counter(u.TotalTokens), new(big.Rat).SetFloat64(c.TotalRate))
	case "components":
		if !slices.Contains(u.Known, prompty.UsagePrompt) || !slices.Contains(u.Known, prompty.UsageCompletion) {
			return evaly.Usage{}, nil
		}
		amount.Mul(counter(u.PromptTokens), new(big.Rat).SetFloat64(c.InputRate))
		output := new(big.Rat).Mul(counter(u.CompletionTokens), new(big.Rat).SetFloat64(c.OutputRate))
		amount.Add(&amount, output)
	default:
		return evaly.Usage{}, evaly.ErrUnsupported
	}
	// Ceil the exact binary-rational policy result, then round the float receipt up.
	integer, remainder := new(big.Int), new(big.Int)
	integer.QuoRem(amount.Num(), amount.Denom(), remainder)
	if remainder.Sign() != 0 {
		integer.Add(integer, big.NewInt(1))
	}
	rounded := new(big.Rat).SetInt(integer)
	units, _ := rounded.Float64()
	if finite(units) && new(big.Rat).SetFloat64(units).Cmp(rounded) < 0 {
		units = math.Nextafter(units, math.Inf(1))
	}
	if !finite(units) || units > c.Bound {
		return evaly.Usage{}, evaly.ErrBudget
	}
	return evaly.Usage{Known: true, Units: units}, nil
}

// ReservationUnits uses precisely the receipt conversion; an unknown bound is invalid.
func (c Conversion) ReservationUnits(unit string, upper prompty.Usage) (float64, error) {
	receipt, err := c.Convert(unit, upper)
	if err != nil {
		return 0, err
	}
	if !receipt.Known {
		return 0, evaly.ErrInvalid
	}
	return receipt.Units, nil
}

func finite(v float64) bool   { return v >= 0 && !math.IsNaN(v) && !math.IsInf(v, 0) }
func positive(v float64) bool { return v > 0 && finite(v) }

func counter(value int) *big.Rat { return new(big.Rat).SetInt64(int64(value)) }
