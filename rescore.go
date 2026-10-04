package evaly

import (
	"context"
	"encoding/json"
)

// SavedView stores only an explicitly permitted grading projection, encoded by
// consumer codecs. It has no reference to a target or environment lifecycle.
type SavedView[I, O, R any] struct {
	record    SavedViewRecord
	input     Codec[I]
	output    Codec[O]
	reference Codec[R]
}
type SavedViewRecord struct {
	Version        int             `json:"version"`
	InputCodec     CodecIdentity   `json:"input_codec"`
	OutputCodec    CodecIdentity   `json:"output_codec"`
	ReferenceCodec CodecIdentity   `json:"reference_codec"`
	Case           CaseRecord      `json:"case"`
	Output         json.RawMessage `json:"output"`
	Evidence       EvidenceRecord  `json:"evidence"`
	Projection     string          `json:"projection"`
	Revision       string          `json:"revision"`
}

func SaveView[I, O, R any](
	view View[I, O, R],
	projection string,
	ic Codec[I],
	oc Codec[O],
	rc Codec[R],
) (SavedView[I, O, R], error) {
	var out SavedView[I, O, R]
	if projection == "" || ValidatePort(ic) != nil || ValidatePort(oc) != nil || ValidatePort(rc) != nil ||
		view.Case.ID == "" ||
		view.Case.Revision == "" {
		return out, ErrInvalid
	}
	for _, identity := range []CodecIdentity{ic.Identity(), oc.Identity(), rc.Identity()} {
		if identity.ID == "" || identity.Version == "" {
			return out, ErrInvalid
		}
	}
	if e := ValidateEvidence(view.Evidence); e != nil {
		return out, e
	}
	i, e := ic.Encode(view.Case.Input)
	if e != nil {
		return out, e
	}
	o, e := oc.Encode(view.Output)
	if e != nil {
		return out, e
	}
	i, e = CanonicalJSON(i)
	if e != nil {
		return out, e
	}
	o, e = CanonicalJSON(o)
	if e != nil {
		return out, e
	}
	r := SavedViewRecord{
		Version:        1,
		InputCodec:     ic.Identity(),
		OutputCodec:    oc.Identity(),
		ReferenceCodec: rc.Identity(),
		Case: CaseRecord{
			ID:               view.Case.ID,
			Revision:         view.Case.Revision,
			Input:            i,
			Metadata:         view.Case.Metadata,
			RequiredEvidence: view.Case.RequiredEvidence,
			Generation:       view.Case.Generation,
		},
		Output:     o,
		Evidence:   view.Evidence,
		Projection: projection,
	}
	if view.Case.Reference != nil {
		r.Case.Reference, e = rc.Encode(*view.Case.Reference)
		if e != nil {
			return out, e
		}
		r.Case.Reference, e = CanonicalJSON(r.Case.Reference)
		if e != nil {
			return out, e
		}
	}
	b, e := canonical(r)
	if e != nil {
		return out, e
	}
	r.Revision = digest(b)
	r, e = cloneJSON(r)
	if e != nil {
		return out, e
	}
	return SavedView[I, O, R]{r, ic, oc, rc}, nil
}
func (s SavedView[I, O, R]) Revision() string { return s.record.Revision }
func (s SavedView[I, O, R]) View() (View[I, O, R], error) {
	var out View[I, O, R]
	if s.record.Revision == "" || ValidatePort(s.input) != nil || ValidatePort(s.output) != nil ||
		ValidatePort(s.reference) != nil {
		return out, ErrUnsealed
	}
	r, e := cloneJSON(s.record)
	if e != nil {
		return out, e
	}
	out.Case = Case[I, R]{
		ID:               r.Case.ID,
		Revision:         r.Case.Revision,
		Metadata:         r.Case.Metadata,
		RequiredEvidence: r.Case.RequiredEvidence,
		Generation:       r.Case.Generation,
	}
	out.Case.Input, e = s.input.Decode(r.Case.Input)
	if e != nil {
		return out, e
	}
	out.Output, e = s.output.Decode(r.Output)
	if e != nil {
		return out, e
	}
	if len(r.Case.Reference) > 0 {
		v, e := s.reference.Decode(r.Case.Reference)
		if e != nil {
			return out, e
		}
		out.Case.Reference = &v
	}
	out.Evidence = r.Evidence
	return out, nil
}

type Assessment struct {
	Version    int              `json:"version"`
	Revision   string           `json:"revision"`
	Parent     string           `json:"parent"`
	Source     string           `json:"source"`
	View       string           `json:"view"`
	Mode       string           `json:"mode"`
	Planned    []GraderRevision `json:"planned"`
	Grades     []Grade          `json:"grades"`
	State      string           `json:"state"`
	StopReason string           `json:"stop_reason"`
	Skipped    []SkippedGrader  `json:"skipped"`
}
type SkippedGrader struct {
	Revision GraderRevision `json:"revision"`
	Reason   string         `json:"reason"`
}

func validateAssessmentContent(a Assessment) error {
	if a.Version != 2 {
		return ErrUnsupported
	}
	if a.Source == "" || a.View == "" || (a.Mode != "rescore" && a.Mode != "observation") || len(a.Planned) == 0 {
		return ErrInvalid
	}
	if e := ValidateGraderRevisions(a.Planned); e != nil {
		return e
	}
	planned := map[string]GraderRevision{}
	seen := map[string]bool{}
	for _, r := range a.Planned {
		planned[r.ID] = r
	}
	for _, g := range a.Grades {
		if !g.Dispatched || planned[g.Revision.ID] != g.Revision || seen[g.Revision.ID] {
			return ErrInvalid
		}
		if e := ValidateGrade(g); e != nil {
			return e
		}
		seen[g.Revision.ID] = true
	}
	for _, g := range a.Skipped {
		if planned[g.Revision.ID] != g.Revision || seen[g.Revision.ID] || g.Reason == "" {
			return ErrInvalid
		}
		seen[g.Revision.ID] = true
	}
	if len(seen) != len(planned) {
		return ErrInvalid
	}
	switch a.State {
	case "complete":
		if a.StopReason != "" || len(a.Skipped) > 0 || len(a.Grades) != len(a.Planned) {
			return ErrInvalid
		}
	case "partial":
		if a.StopReason == "" {
			return ErrInvalid
		}
	default:
		return ErrUnsupported
	}
	return nil
}

// SealAssessment supplies canonical identity after all partial results are collected.
func SealAssessment(a Assessment) (Assessment, error) {
	a.Revision = ""
	if a.Grades == nil {
		a.Grades = []Grade{}
	}
	if a.Skipped == nil {
		a.Skipped = []SkippedGrader{}
	}
	if e := validateAssessmentContent(a); e != nil {
		return Assessment{}, e
	}
	b, e := canonical(a)
	if e != nil {
		return Assessment{}, e
	}
	a.Revision = digest(b)
	return cloneJSON(a)
}
func ValidateAssessment(a Assessment) error {
	revision := a.Revision
	sealed, e := SealAssessment(a)
	if e != nil {
		return e
	}
	if revision == "" || sealed.Revision != revision {
		return ErrCorrupt
	}
	return nil
}
func RestoreAssessment(a Assessment) (Assessment, error) {
	if e := ValidateAssessment(a); e != nil {
		return Assessment{}, e
	}
	return cloneJSON(a)
}

func Rescore[I, O, R any](
	ctx context.Context,
	s SavedView[I, O, R],
	graders []Grader[I, O, R],
	source, parent, mode string,
) (Assessment, error) {
	a := Assessment{Version: 2, Source: source, Parent: parent, View: s.Revision(), Mode: mode, State: "complete"}
	if source == "" || (mode != "rescore" && mode != "observation") || len(graders) == 0 {
		return Assessment{}, ErrInvalid
	}
	for _, g := range graders {
		if e := ValidatePort(g); e != nil {
			return Assessment{}, e
		}
		a.Planned = append(a.Planned, g.Revision())
	}
	if e := ValidateGraderRevisions(a.Planned); e != nil {
		return Assessment{}, e
	}
	// Validate the entire saved projection before the first judge dispatch.
	if _, e := RestoreSavedView(s.Record(), s.input, s.output, s.reference); e != nil {
		return Assessment{}, e
	}
	for i, g := range graders {
		if ctx.Err() != nil {
			a.State = "partial"
			a.StopReason = "context_cancelled"
			for _, rev := range a.Planned[i:] {
				a.Skipped = append(a.Skipped, SkippedGrader{rev, a.StopReason})
			}
			break
		}
		grade := Assess(ctx, []Grader[I, O, R]{g}, s.View)[0]
		if !grade.Dispatched {
			a.State = "partial"
			a.StopReason = "grading_projection"
			if ctx.Err() != nil {
				a.StopReason = "context_cancelled"
			}
			for _, rev := range a.Planned[i:] {
				a.Skipped = append(a.Skipped, SkippedGrader{rev, a.StopReason})
			}
			break
		}
		a.Grades = append(a.Grades, grade)
		if ctx.Err() != nil {
			a.State = "partial"
			a.StopReason = "context_cancelled"
			for _, rev := range a.Planned[i+1:] {
				a.Skipped = append(a.Skipped, SkippedGrader{rev, a.StopReason})
			}
			break
		}
	}
	return SealAssessment(a)
}

// Record returns a portable permitted projection with explicit consumer codecs.
func (s SavedView[I, O, R]) Record() SavedViewRecord { r, _ := cloneJSON(s.record); return r }

func RestoreSavedView[I, O, R any](
	r SavedViewRecord,
	ic Codec[I],
	oc Codec[O],
	rc Codec[R],
) (SavedView[I, O, R], error) {
	var out SavedView[I, O, R]
	if r.Version != 1 {
		return out, ErrUnsupported
	}
	if ValidatePort(ic) != nil || ValidatePort(oc) != nil || ValidatePort(rc) != nil || r.InputCodec != ic.Identity() ||
		r.OutputCodec != oc.Identity() ||
		r.ReferenceCodec != rc.Identity() {
		return out, ErrUnsupported
	}
	if r.Projection == "" || r.Case.ID == "" || r.Case.Revision == "" {
		return out, ErrInvalid
	}
	if err := ValidateEvidence(r.Evidence); err != nil {
		return out, err
	}
	revision := r.Revision
	r.Revision = ""
	b, e := canonical(r)
	if e != nil || digest(b) != revision {
		return out, ErrCorrupt
	}
	r.Revision = revision
	r, e = cloneJSON(r)
	if e != nil {
		return out, e
	}
	out = SavedView[I, O, R]{r, ic, oc, rc}
	v, e := out.View()
	if e != nil {
		return SavedView[I, O, R]{}, e
	}
	rebuilt, e := SaveView(v, r.Projection, ic, oc, rc)
	if e != nil {
		return SavedView[I, O, R]{}, e
	}
	if rebuilt.Revision() != revision {
		return SavedView[I, O, R]{}, ErrCorrupt
	}
	return rebuilt, nil
}
func SaveSavedView[I, O, R any](ctx context.Context, s ArtifactStore, id string, v SavedView[I, O, R]) error {
	if v.Revision() == "" {
		return ErrUnsealed
	}
	if _, err := RestoreSavedView(v.Record(), v.input, v.output, v.reference); err != nil {
		return err
	}
	env, e := NewEnvelope("view", id, v.Record())
	if e != nil {
		return e
	}
	return s.Put(ctx, env)
}

func LoadSavedView[I, O, R any](
	ctx context.Context,
	s ArtifactStore,
	id string,
	ic Codec[I],
	oc Codec[O],
	rc Codec[R],
) (SavedView[I, O, R], error) {
	env, e := s.Get(ctx, id)
	if e != nil {
		return SavedView[I, O, R]{}, e
	}
	if env.Kind != "view" {
		return SavedView[I, O, R]{}, ErrUnsupported
	}
	r, e := DecodeWire[SavedViewRecord](env.Data)
	if e != nil {
		return SavedView[I, O, R]{}, ErrCorrupt
	}
	return RestoreSavedView(r, ic, oc, rc)
}
