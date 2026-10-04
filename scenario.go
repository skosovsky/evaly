package evaly

import (
	"context"
	"encoding/json"
	"time"
)

type ScenarioPlan struct {
	Mode       string        `json:"mode"`
	Seed       int64         `json:"seed"`
	MaxSteps   int           `json:"max_steps"`
	Timeout    time.Duration `json:"timeout"`
	Generation *Generation   `json:"generation,omitempty"`
}
type ScenarioRecord struct {
	Version     int               `json:"version"`
	Revision    string            `json:"revision"`
	Driver      string            `json:"driver"`
	Plan        ScenarioPlan      `json:"plan"`
	StateCodec  CodecIdentity     `json:"state_codec"`
	OutputCodec CodecIdentity     `json:"output_codec"`
	State       json.RawMessage   `json:"state"`
	Outputs     []json.RawMessage `json:"outputs"`
	Steps       int               `json:"steps"`
	Stop        string            `json:"stop"`
}

// RunScenario snapshots each output before the next host step. It distinguishes
// replaying a fixed scenario from searching for a new adversarial trajectory.
func RunScenario[S, O any](
	ctx context.Context,
	driver ScenarioStep[S, O],
	state S,
	p ScenarioPlan,
	sc Codec[S],
	oc Codec[O],
) (ScenarioRecord, error) {
	r := ScenarioRecord{Version: 1, Plan: p, Outputs: []json.RawMessage{}}
	if driver == nil || driver.Revision() == "" || sc == nil || oc == nil || p.MaxSteps <= 0 || p.MaxSteps > 10000 ||
		p.Timeout <= 0 ||
		(p.Mode != "search" && p.Mode != "replay") {
		return r, ErrInvalid
	}
	if p.Generation != nil && (p.Generation.Mode != p.Mode || p.Generation.Generator == "") {
		return r, ErrInvalid
	}
	if sc.Identity().ID == "" || sc.Identity().Version == "" || oc.Identity().ID == "" || oc.Identity().Version == "" {
		return r, ErrInvalid
	}
	r.Driver = driver.Revision()
	r.StateCodec = sc.Identity()
	r.OutputCodec = oc.Identity()
	ctx, cancel := context.WithTimeout(ctx, p.Timeout)
	defer cancel()
	r.Stop = "step_limit"
	var executionErr error
	for range p.MaxSteps {
		if e := ctx.Err(); e != nil {
			r.Stop = "deadline"
			executionErr = e
			break
		}
		next, o, done, e := driver.Step(ctx, state)
		state = next
		r.Steps++
		b, encodeErr := oc.Encode(o)
		if encodeErr != nil {
			r.Stop = "codec_error"
			executionErr = encodeErr
			break
		}
		b, encodeErr = CanonicalJSON(b)
		if encodeErr != nil {
			r.Stop = "codec_error"
			executionErr = encodeErr
			break
		}
		r.Outputs = append(r.Outputs, append(json.RawMessage(nil), b...))
		if ctx.Err() != nil {
			r.Stop = "deadline"
			executionErr = ctx.Err()
			break
		}
		if e != nil {
			r.Stop = "error"
			executionErr = e
			break
		}
		if done {
			r.Stop = "completed"
			break
		}
	}
	b, e := sc.Encode(state)
	if e != nil {
		return r, e
	}
	r.State, e = CanonicalJSON(b)
	if e != nil {
		return r, e
	}
	bytes, e := canonical(r)
	if e != nil {
		return r, e
	}
	r.Revision = digest(bytes)
	r, e = cloneJSON(r)
	if e != nil {
		return r, e
	}
	return r, executionErr
}

func RestoreScenario[S, O any](r ScenarioRecord, sc Codec[S], oc Codec[O]) (ScenarioResult[S, O], error) {
	var out ScenarioResult[S, O]
	if err := validateScenario(r); err != nil {
		return out, err
	}
	if sc == nil || oc == nil || r.StateCodec != sc.Identity() || r.OutputCodec != oc.Identity() {
		return out, ErrUnsupported
	}
	var e error
	out.State, e = sc.Decode(r.State)
	if e != nil {
		return out, e
	}
	for _, b := range r.Outputs {
		v, e := oc.Decode(b)
		if e != nil {
			return out, e
		}
		out.Outputs = append(out.Outputs, v)
	}
	out.DriverRevision = r.Driver
	out.Mode = r.Plan.Mode
	if r.Plan.Generation != nil {
		generation := *r.Plan.Generation
		out.Generation = &generation
	}
	out.Steps = r.Steps
	out.Stop = r.Stop
	return out, nil
}

// DraftFromScenario attaches the discovered trajectory to generator lineage.
func DraftFromScenario[I, R any](c Case[I, R], p Generation, trajectory ScenarioRecord) (DatasetDraft[I, R], error) {
	if err := validateScenario(trajectory); err != nil {
		return DatasetDraft[I, R]{}, err
	}
	if trajectory.Plan.Mode != "search" || p.Generator == "" {
		return DatasetDraft[I, R]{}, ErrInvalid
	}
	p.LabelValidated = false
	p.Mode = "search"
	p.DriverRevision = trajectory.Driver
	p.TrajectoryRevision = trajectory.Revision
	c.Generation = &p
	c.Revision = ""
	return DatasetDraft[I, R]{Cases: []Case[I, R]{c}, Selection: "all"}, nil
}

// Integrity hashes are not substitutes for trajectory semantics.
func validateScenario(r ScenarioRecord) error {
	if r.Version != 1 {
		return ErrUnsupported
	}
	if r.Driver == "" || r.Revision == "" || r.Plan.MaxSteps <= 0 || r.Plan.MaxSteps > 10000 || r.Plan.Timeout <= 0 ||
		r.Steps < 0 ||
		r.Steps > r.Plan.MaxSteps ||
		r.StateCodec.ID == "" ||
		r.StateCodec.Version == "" ||
		r.OutputCodec.ID == "" ||
		r.OutputCodec.Version == "" ||
		(r.Plan.Mode != "search" && r.Plan.Mode != "replay") {
		return ErrInvalid
	}
	if r.Plan.Generation != nil && (r.Plan.Generation.Mode != r.Plan.Mode || r.Plan.Generation.Generator == "") {
		return ErrInvalid
	}
	switch r.Stop {
	case "completed", "error":
		if r.Steps == 0 || len(r.Outputs) != r.Steps {
			return ErrInvalid
		}
	case "step_limit":
		if r.Steps != r.Plan.MaxSteps || len(r.Outputs) != r.Steps {
			return ErrInvalid
		}
	case "deadline":
		if len(r.Outputs) != r.Steps {
			return ErrInvalid
		}
	case "codec_error":
		if r.Steps == 0 || len(r.Outputs) != r.Steps-1 {
			return ErrInvalid
		}
	default:
		return ErrUnsupported
	}
	if _, err := CanonicalJSON(r.State); err != nil {
		return err
	}
	for _, output := range r.Outputs {
		if _, err := CanonicalJSON(output); err != nil {
			return err
		}
	}
	rev := r.Revision
	r.Revision = ""
	b, err := canonical(r)
	if err != nil || digest(b) != rev {
		return ErrCorrupt
	}
	return nil
}
