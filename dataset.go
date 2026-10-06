package evaly

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

type Generation struct {
	DriverRevision     string `json:"driver_revision,omitempty"`
	TrajectoryRevision string `json:"trajectory_revision,omitempty"`
	ParentCase         string `json:"parent_case"`
	Generator          string `json:"generator"`
	Model              string `json:"model"`
	Seed               int64  `json:"seed"`
	Mode               string `json:"mode"` // replay or search
	LabelValidated     bool   `json:"label_validated"`
}
type Case[I, R any] struct {
	ID               string            `json:"ID"`
	Revision         string            `json:"Revision"`
	Input            I                 `json:"Input"`
	Reference        *R                `json:"Reference"`
	Metadata         map[string]string `json:"Metadata"`
	RequiredEvidence []string          `json:"RequiredEvidence"`
	Generation       *Generation       `json:"Generation"`
}
type DatasetDraft[I, R any] struct {
	Cases          []Case[I, R] `json:"Cases"`
	Selection      string       `json:"Selection"`
	ParentRevision string       `json:"ParentRevision"`
}
type CaseRecord struct {
	ID               string            `json:"id"`
	Revision         string            `json:"revision"`
	Input            json.RawMessage   `json:"input"`
	Reference        json.RawMessage   `json:"reference,omitempty"`
	Metadata         map[string]string `json:"metadata,omitempty"`
	RequiredEvidence []string          `json:"required_evidence,omitempty"`
	Generation       *Generation       `json:"generation,omitempty"`
}
type DatasetRecord struct {
	Version        int           `json:"version"`
	State          string        `json:"state"`
	Revision       string        `json:"revision"`
	Selection      string        `json:"selection"`
	ParentRevision string        `json:"parent_revision,omitempty"`
	InputCodec     CodecIdentity `json:"input_codec"`
	ReferenceCodec CodecIdentity `json:"reference_codec"`
	Cases          []CaseRecord  `json:"cases"`
}

// Dataset holds only private serialized domain values. Access decodes fresh copies.
type Dataset[I, R any] struct {
	record    DatasetRecord
	input     Codec[I]
	reference Codec[R]
}

func (d Dataset[I, R]) Revision() string      { return d.record.Revision }
func (d Dataset[I, R]) Len() int              { return len(d.record.Cases) }
func (d Dataset[I, R]) Record() DatasetRecord { r, _ := cloneJSON(d.record); return r }

// CaseAt returns a fresh decoded copy of only the requested case.
func (d Dataset[I, R]) CaseAt(index int) (Case[I, R], error) {
	if d.record.State != sealedState {
		return Case[I, R]{}, ErrUnsealed
	}
	if index < 0 || index >= len(d.record.Cases) {
		return Case[I, R]{}, ErrInvalid
	}
	r, err := cloneJSON(d.record.Cases[index])
	if err != nil {
		return Case[I, R]{}, err
	}
	i, err := d.input.Decode(r.Input)
	if err != nil {
		return Case[I, R]{}, err
	}
	c := Case[I, R]{
		ID:               r.ID,
		Revision:         r.Revision,
		Input:            i,
		Metadata:         r.Metadata,
		RequiredEvidence: r.RequiredEvidence,
		Generation:       r.Generation, Reference: nil,
	}
	if len(r.Reference) > 0 {
		ref, err := d.reference.Decode(r.Reference)
		if err != nil {
			return Case[I, R]{}, err
		}
		c.Reference = &ref
	}
	return c, nil
}
func (d Dataset[I, R]) Cases() ([]Case[I, R], error) {
	if d.record.State != sealedState {
		return nil, ErrUnsealed
	}
	out := make([]Case[I, R], 0, d.Len())
	for index := range d.Len() {
		c, err := d.CaseAt(index)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, nil
}

// Validate performs full encoding checks without publishing a runnable snapshot.
func (d DatasetDraft[I, R]) Validate(ic Codec[I], rc Codec[R]) error {
	_, e := d.build(ic, rc)
	return e
}
func (d DatasetDraft[I, R]) Seal(ic Codec[I], rc Codec[R]) (Dataset[I, R], error) {
	r, e := d.build(ic, rc)
	if e != nil {
		return Dataset[I, R]{}, e
	}
	return Dataset[I, R]{r, ic, rc}, nil
}
func (d DatasetDraft[I, R]) build(ic Codec[I], rc Codec[R]) (DatasetRecord, error) {
	var zeroCodecIdentity CodecIdentity
	r := DatasetRecord{
		Version:        1,
		State:          sealedState,
		Selection:      d.Selection,
		ParentRevision: d.ParentRevision,
		Revision:       "",
		InputCodec:     zeroCodecIdentity,
		ReferenceCodec: zeroCodecIdentity,
		Cases:          nil,
	}
	if ic == nil || rc == nil || len(d.Cases) == 0 || d.Selection == "" {
		return r, ErrInvalid
	}
	if err := ValidatePort(ic); err != nil {
		return r, err
	}
	if err := ValidatePort(rc); err != nil {
		return r, err
	}
	r.InputCodec = ic.Identity()
	r.ReferenceCodec = rc.Identity()
	if r.InputCodec.ID == "" || r.InputCodec.Version == "" || r.ReferenceCodec.ID == "" ||
		r.ReferenceCodec.Version == "" {
		return r, ErrInvalid
	}
	seen := map[string]bool{}
	for _, c := range d.Cases {
		if c.ID == "" {
			return r, ErrInvalid
		}
		if seen[c.ID] {
			return r, fmt.Errorf("%w: case %s", ErrConflict, c.ID)
		}
		seen[c.ID] = true
		rec, e := sealCaseRecord(c, ic, rc, r.InputCodec, r.ReferenceCodec)
		if e != nil {
			return r, e
		}
		r.Cases = append(r.Cases, rec)
	}
	b, e := canonical(r)
	if e != nil {
		return r, e
	}
	r.Revision = digest(b)
	return cloneJSON(r)
}
func RestoreDataset[I, R any](r DatasetRecord, ic Codec[I], rc Codec[R]) (Dataset[I, R], error) {
	if r.Version != 1 {
		return Dataset[I, R]{}, ErrUnsupported
	}
	if r.State != sealedState {
		return Dataset[I, R]{}, ErrUnsealed
	}
	if err := validateCodecPorts(ic, rc); err != nil {
		return Dataset[I, R]{}, err
	}
	if r.InputCodec != ic.Identity() || r.ReferenceCodec != rc.Identity() {
		return Dataset[I, R]{}, ErrUnsupported
	}
	rcopy, e := cloneJSON(r)
	if e != nil {
		return Dataset[I, R]{}, e
	}
	d := Dataset[I, R]{rcopy, ic, rc}
	cs, e := d.Cases()
	if e != nil {
		return Dataset[I, R]{}, e
	}
	rebuilt, e := (DatasetDraft[I, R]{Cases: cs, Selection: r.Selection, ParentRevision: r.ParentRevision}).Seal(ic, rc)
	if e != nil {
		return Dataset[I, R]{}, e
	}
	storedBytes, _ := canonical(r)
	rebuiltBytes, _ := canonical(rebuilt.Record())
	if rebuilt.Revision() != r.Revision || string(storedBytes) != string(rebuiltBytes) {
		return Dataset[I, R]{}, ErrCorrupt
	}
	return rebuilt, nil
}

type Generator[I, R any] interface {
	Generate(context.Context, []Case[I, R]) ([]Case[I, R], error)
	Provenance() Generation
}

// GenerateDraft forcibly marks self-assigned labels unvalidated.
func GenerateDraft[I, R any](
	ctx context.Context,
	g Generator[I, R],
	parents []Case[I, R],
	selection string,
) (DatasetDraft[I, R], error) {
	if g == nil || selection == "" {
		return DatasetDraft[I, R]{}, ErrInvalid
	}
	if err := ValidatePort(g); err != nil {
		return DatasetDraft[I, R]{}, err
	}
	p := g.Provenance()
	if p.Generator == "" || (p.Mode != searchMode && p.Mode != replayMode) {
		return DatasetDraft[I, R]{}, ErrInvalid
	}
	if err := ctx.Err(); err != nil {
		return DatasetDraft[I, R]{}, err
	}
	cs, e := g.Generate(ctx, parents)
	if e != nil {
		return DatasetDraft[I, R]{}, e
	}
	p.LabelValidated = false
	for i := range cs {
		cp := p
		cs[i].Revision = ""
		cs[i].Generation = &cp
	}
	return DatasetDraft[I, R]{Cases: cs, Selection: selection, ParentRevision: ""}, nil
}

type ScenarioContext struct {
	Seed int64  `json:"Seed"`
	Mode string `json:"Mode"`
	Step int    `json:"Step"`
}

type ScenarioStep[S, O any] interface {
	Revision() string
	Step(context.Context, S, ScenarioContext) (S, O, bool, error)
}
type ScenarioResult[S, O any] struct {
	DriverRevision string      `json:"DriverRevision"`
	Mode           string      `json:"Mode"`
	Generation     *Generation `json:"Generation"`
	State          S           `json:"State"`
	Outputs        []O         `json:"Outputs"`
	Steps          int         `json:"Steps"`
	StateStep      int         `json:"StateStep"`
	Stop           string      `json:"Stop"`
}

// Drive bounds host-owned multistep scenarios; Step must honor context cancellation.
func Drive[S, O any](
	ctx context.Context,
	driver ScenarioStep[S, O],
	state S,
	maxSteps int,
	timeout time.Duration,
) (ScenarioResult[S, O], error) {
	r := ScenarioResult[S, O]{
		State:          state,
		DriverRevision: "",
		Mode:           "",
		Generation:     nil,
		Outputs:        nil,
		Steps:          0,
		StateStep:      0,
		Stop:           "",
	}
	if err := ValidatePort(driver); err != nil {
		return r, err
	}
	if driver == nil || driver.Revision() == "" || maxSteps <= 0 || maxSteps > 10000 || timeout <= 0 {
		return r, ErrInvalid
	}
	r.DriverRevision = driver.Revision()
	r.Mode = replayMode
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	for step := range maxSteps {
		if e := ctx.Err(); e != nil {
			r.Stop = scenarioDeadline
			return r, e
		}
		s, o, done, e := driver.Step(ctx, r.State, ScenarioContext{Mode: replayMode, Step: step, Seed: 0})
		r.State = s
		r.Outputs = append(r.Outputs, o)
		r.Steps++
		r.StateStep = r.Steps
		if ctx.Err() != nil {
			r.Stop = scenarioDeadline
			return r, ctx.Err()
		}
		if e != nil {
			r.Stop = scenarioError
			return r, e
		}
		if done {
			r.Stop = completedState
			return r, nil
		}
	}
	r.Stop = scenarioStepLimit
	return r, nil
}

func sealCaseRecord[I, R any](
	c Case[I, R],
	ic Codec[I],
	rc Codec[R],
	inputIdentity, referenceIdentity CodecIdentity,
) (CaseRecord, error) {
	if c.Generation != nil && c.Generation.Mode != replayMode && c.Generation.Mode != searchMode {
		return CaseRecord{}, ErrUnsupported
	}
	if c.Generation != nil &&
		(!c.Generation.LabelValidated || c.Generation.Generator == "" || c.Generation.Mode == "") {
		return CaseRecord{}, fmt.Errorf("%w: generated draft %s needs host validation", ErrUnsealed, c.ID)
	}
	b, e := ic.Encode(c.Input)
	if e != nil {
		return CaseRecord{}, e
	}
	b, e = CanonicalJSON(b)
	if e != nil {
		return CaseRecord{}, e
	}
	rec := CaseRecord{
		ID:               c.ID,
		Input:            b,
		Metadata:         c.Metadata,
		RequiredEvidence: c.RequiredEvidence,
		Generation:       c.Generation, Revision: "", Reference: nil,
	}
	if c.Reference != nil {
		b, e = rc.Encode(*c.Reference)
		if e != nil {
			return CaseRecord{}, e
		}
		rec.Reference, e = CanonicalJSON(b)
		if e != nil {
			return CaseRecord{}, e
		}
	}
	bytes, e := canonical(struct {
		Case      CaseRecord    `json:"Case"`
		Input     CodecIdentity `json:"Input"`
		Reference CodecIdentity `json:"Reference"`
	}{rec, inputIdentity, referenceIdentity})
	if e != nil {
		return CaseRecord{}, e
	}
	rec.Revision = digest(bytes)
	return rec, nil
}
