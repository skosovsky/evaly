package evaly

import "reflect"

// StructuralValidator optionally declares local configuration invariants for a
// port. Validate must be side-effect-free and must not probe paid capabilities.
// Port identities and validation results must remain stable during execution.
type StructuralValidator interface{ Validate() error }

// ValidatePort rejects absent capabilities and invokes an optional structural
// validation hook. Reflection only detects typed nil capabilities; caller-owned
// domain values are never copied or inspected.
func ValidatePort(port any) error {
	if port == nil {
		return ErrInvalid
	}
	value := reflect.ValueOf(port)
	switch value.Kind() {
	case reflect.Pointer, reflect.Func, reflect.Interface, reflect.Map, reflect.Slice, reflect.Chan:
		if value.IsNil() {
			return ErrInvalid
		}
	}
	if validator, ok := port.(StructuralValidator); ok {
		return validator.Validate()
	}
	return nil
}

// ValidateCodecIdentity validates an identity without executing a codec.
func ValidateCodecIdentity(identity CodecIdentity) error {
	if identity.ID == "" || identity.Version == "" {
		return ErrInvalid
	}
	return nil
}

// ValidateGraderRevisions rejects missing or duplicate planned grader identities.
func ValidateGraderRevisions(revisions []GraderRevision) error {
	if len(revisions) == 0 {
		return ErrInvalid
	}
	seen := make(map[string]bool, len(revisions))
	for _, revision := range revisions {
		if revision.ID == "" || revision.Implementation == "" || revision.Rubric == "" {
			return ErrInvalid
		}
		if seen[revision.ID] {
			return ErrConflict
		}
		seen[revision.ID] = true
	}
	return nil
}

// ValidateRunConfig checks structural invariants without lifecycle or target dispatch.
func ValidateRunConfig[I, O, R, E any](c RunConfig[I, O, R, E]) error {
	if c.Budget != nil {
		if err := ValidatePort(c.Budget); err != nil {
			return err
		}
	}
	if err := validatePlan(c.Plan); err != nil {
		return err
	}
	if !validArtifactID(c.ID) || c.Target == nil || c.OutputCodec == nil || c.Lifecycle == nil || c.Project == nil ||
		c.ProjectionRevision == "" ||
		c.Dataset.Len() == 0 ||
		c.Dataset.Len()*c.Plan.Repeats > 1000000 {
		return ErrInvalid
	}
	for _, port := range []any{c.Target, c.Lifecycle, c.OutputCodec, c.Capture.Policy, c.Dataset.input, c.Dataset.reference} {
		if err := ValidatePort(port); err != nil {
			return err
		}
	}
	if c.Dataset.record.State != "sealed" {
		return ErrUnsealed
	}
	if c.Dataset.input == nil || c.Dataset.reference == nil ||
		c.Dataset.input.Identity() != c.Dataset.record.InputCodec ||
		c.Dataset.reference.Identity() != c.Dataset.record.ReferenceCodec {
		return ErrInvalid
	}
	if err := ValidateCodecIdentity(c.OutputCodec.Identity()); err != nil {
		return err
	}
	if err := validateProvenance(c.Provenance); err != nil {
		return err
	}
	life := c.Lifecycle.Identity()
	if life.Fixture == "" || life.Reset == "" || (life.Isolation != Isolated && life.Isolation != SerialShared) {
		return ErrInvalid
	}
	revisions := make([]GraderRevision, 0, len(c.Graders))
	for _, g := range c.Graders {
		if err := ValidatePort(g); err != nil {
			return err
		}
		if g == nil {
			return ErrInvalid
		}
		revisions = append(revisions, g.Revision())
	}
	if err := ValidateGraderRevisions(revisions); err != nil {
		return err
	}
	if _, err := NewCapture(c.Capture); err != nil {
		return err
	}
	for _, cs := range c.Dataset.record.Cases {
		cfg := c.Capture
		cfg.RequiredKinds = append(append([]string(nil), cfg.RequiredKinds...), cs.RequiredEvidence...)
		if _, err := NewCapture(cfg); err != nil {
			return err
		}
	}
	return nil
}
