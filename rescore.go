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
	if projection == "" || ic == nil || oc == nil || rc == nil || view.Case.ID == "" || view.Case.Revision == "" {
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
	if s.record.Revision == "" || s.input == nil || s.output == nil || s.reference == nil {
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
	Version  int     `json:"version"`
	Revision string  `json:"revision"`
	Parent   string  `json:"parent"`
	Source   string  `json:"source"`
	View     string  `json:"view"`
	Mode     string  `json:"mode"`
	Grades   []Grade `json:"grades"`
}

func Rescore[I, O, R any](
	ctx context.Context,
	s SavedView[I, O, R],
	graders []Grader[I, O, R],
	source, parent, mode string,
) (Assessment, error) {
	a := Assessment{Version: 1, Source: source, Parent: parent, View: s.Revision(), Mode: mode}
	if source == "" || (mode != "rescore" && mode != "observation") || len(graders) == 0 {
		return a, ErrInvalid
	}
	for _, g := range graders {
		if g == nil {
			return a, ErrInvalid
		}
		view, e := s.View()
		if e != nil {
			return a, e
		}
		a.Grades = append(a.Grades, Assess(ctx, []Grader[I, O, R]{g}, view)[0])
	}
	b, e := canonical(a)
	if e != nil {
		return a, e
	}
	a.Revision = digest(b)
	return a, nil
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
	if ic == nil || oc == nil || rc == nil || r.InputCodec != ic.Identity() || r.OutputCodec != oc.Identity() ||
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
	var r SavedViewRecord
	if e = json.Unmarshal(env.Data, &r); e != nil {
		return SavedView[I, O, R]{}, ErrCorrupt
	}
	return RestoreSavedView(r, ic, oc, rc)
}
