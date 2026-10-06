package optimizer

import (
	"context"

	"github.com/skosovsky/evaly"
)

// SplitValidator is a host's revisioned declaration of cross-split independence.
// Search still checks technical case IDs independently of this port. Revision
// and callbacks must remain stable after preflight, honor context and support
// concurrent use if shared. Successful validation is a host declaration, not a
// statistical proof. Config may omit this optional port; a provided typed-nil
// or invalid adapter fails preflight.
type SplitValidator[I, R any] interface {
	ValidateSplit(context.Context, Split[I, R]) error
	Revision() string
}

type SplitValidatorFunc[I, R any] struct {
	Identity string
	Check    func(context.Context, Split[I, R]) error
}

func (v SplitValidatorFunc[I, R]) Revision() string { return v.Identity }
func (v SplitValidatorFunc[I, R]) Validate() error {
	if v.Identity == "" || v.Check == nil {
		return evaly.ErrInvalid
	}
	return nil
}
func (v SplitValidatorFunc[I, R]) ValidateSplit(ctx context.Context, s Split[I, R]) error {
	if err := v.Validate(); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return v.Check(ctx, s)
}

// KeySplitValidator rejects overlap according to host supplied group or content
// keys. It does not infer semantic similarity. Repeated keys within one split
// are permitted; the same key across two splits is a conflict.
type KeySplitValidator[I, R any] struct {
	Identity   string
	GroupKey   func(evaly.Case[I, R]) (string, error)
	ContentKey func(evaly.Case[I, R]) (string, error)
}

func (v KeySplitValidator[I, R]) Revision() string { return v.Identity }
func (v KeySplitValidator[I, R]) Validate() error {
	if v.Identity == "" || (v.GroupKey == nil && v.ContentKey == nil) {
		return evaly.ErrInvalid
	}
	return nil
}
func (v KeySplitValidator[I, R]) ValidateSplit(ctx context.Context, s Split[I, R]) error {
	if err := v.Validate(); err != nil {
		return err
	}
	for _, key := range []func(evaly.Case[I, R]) (string, error){v.GroupKey, v.ContentKey} {
		if key == nil {
			continue
		}
		if err := validateSplitKey(ctx, s, key); err != nil {
			return err
		}
	}
	return nil
}

func validateSplitKey[I, R any](ctx context.Context, s Split[I, R], key func(evaly.Case[I, R]) (string, error)) error {
	seen := map[string]int{}
	for phase, d := range []evaly.Dataset[I, R]{s.Training, s.Calibration, s.Holdout} {
		for index := range d.Len() {
			if err := ctx.Err(); err != nil {
				return err
			}
			c, err := d.CaseAt(index)
			if err != nil {
				return err
			}
			k, err := key(c)
			if err != nil {
				return err
			}
			if k == "" {
				return evaly.ErrInvalid
			}
			if prior, ok := seen[k]; ok && prior != phase {
				return evaly.ErrConflict
			}
			seen[k] = phase
		}
	}
	return nil
}
