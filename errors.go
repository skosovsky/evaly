package evaly

import "errors"

var (
	ErrInvalid     = errors.New("evaly: invalid contract")
	ErrUnsupported = errors.New("evaly: unsupported capability or schema")
	ErrConflict    = errors.New("evaly: revision or identity conflict")
	ErrUnsealed    = errors.New("evaly: snapshot is not sealed")
	ErrBudget      = errors.New("evaly: budget exhausted")
	ErrIncomplete  = errors.New("evaly: insufficient evidence")
	ErrClosed      = errors.New("evaly: closed")
	ErrCorrupt     = errors.New("evaly: corrupt artifact")
)
