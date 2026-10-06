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
	StateStep   int               `json:"state_step"`
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
	var zeroCodecIdentity CodecIdentity
	r := ScenarioRecord{
		Version:     2,
		Plan:        p,
		Outputs:     []json.RawMessage{},
		Revision:    "",
		Driver:      "",
		StateCodec:  zeroCodecIdentity,
		OutputCodec: zeroCodecIdentity,
		State:       nil,
		Steps:       0,
		StateStep:   0,
		Stop:        "",
	}
	for _, port := range []any{driver, sc, oc} {
		if err := ValidatePort(port); err != nil {
			return r, err
		}
	}
	if driver == nil || driver.Revision() == "" || sc == nil || oc == nil || p.MaxSteps <= 0 || p.MaxSteps > 10000 ||
		p.Timeout <= 0 ||
		(p.Mode != searchMode && p.Mode != replayMode) {
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
	initial, err := sc.Encode(state)
	if err != nil {
		return r, err
	}
	r.State, err = CanonicalJSON(initial)
	if err != nil {
		return r, err
	}
	ctx, cancel := context.WithTimeout(ctx, p.Timeout)
	defer cancel()
	r.Stop = scenarioStepLimit
	var executionErr error
	for step := range p.MaxSteps {
		if e := ctx.Err(); e != nil {
			r.Stop = scenarioDeadline
			executionErr = e
			break
		}
		done, stepErr := executeScenarioStep(ctx, driver, &state, p, sc, oc, step, &r)
		if done {
			executionErr = stepErr
			break
		}
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
	if err := validateCodecPorts(sc, oc); err != nil {
		return out, err
	}
	if r.StateCodec != sc.Identity() || r.OutputCodec != oc.Identity() {
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
	out.StateStep = r.StateStep
	out.Stop = r.Stop
	return out, nil
}

// DraftFromScenario attaches the discovered trajectory to generator lineage.
func DraftFromScenario[I, R any](c Case[I, R], p Generation, trajectory ScenarioRecord) (DatasetDraft[I, R], error) {
	if err := validateScenario(trajectory); err != nil {
		return DatasetDraft[I, R]{}, err
	}
	if trajectory.Plan.Mode != searchMode || p.Generator == "" {
		return DatasetDraft[I, R]{}, ErrInvalid
	}
	p.LabelValidated = false
	p.Mode = searchMode
	p.DriverRevision = trajectory.Driver
	p.TrajectoryRevision = trajectory.Revision
	c.Generation = &p
	c.Revision = ""
	return DatasetDraft[I, R]{Cases: []Case[I, R]{c}, Selection: assertionAll, ParentRevision:

	// Integrity hashes are not substitutes for trajectory semantics.
	""}, nil
}

func validateScenario(r ScenarioRecord) error {
	if r.Version != 2 {
		return ErrUnsupported
	}
	if r.Driver == "" || r.Revision == "" || r.Plan.MaxSteps <= 0 || r.Plan.MaxSteps > 10000 || r.Plan.Timeout <= 0 ||
		r.Steps < 0 ||
		r.Steps > r.Plan.MaxSteps || r.StateStep < 0 || r.StateStep > r.Steps ||
		r.StateCodec.ID == "" ||
		r.StateCodec.Version == "" ||
		r.OutputCodec.ID == "" ||
		r.OutputCodec.Version == "" ||
		(r.Plan.Mode != searchMode && r.Plan.Mode != replayMode) {
		return ErrInvalid
	}
	if r.Plan.Generation != nil && (r.Plan.Generation.Mode != r.Plan.Mode || r.Plan.Generation.Generator == "") {
		return ErrInvalid
	}
	if r.Stop != scenarioCodecError && r.StateStep != r.Steps {
		return ErrInvalid
	}
	if err := validateScenarioStop(r); err != nil {
		return err
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

func executeScenarioStep[S, O any](
	ctx context.Context,
	driver ScenarioStep[S, O],
	state *S,
	p ScenarioPlan,
	sc Codec[S],
	oc Codec[O],
	step int,
	r *ScenarioRecord,
) (bool, error) {
	var executionErr error
	next, o, done, e := driver.Step(ctx, *state, ScenarioContext{Seed: p.Seed, Mode: p.Mode, Step: step})
	*state = next
	r.Steps++
	stateBytes, stateErr := sc.Encode(*state)
	if stateErr == nil {
		stateBytes, stateErr = CanonicalJSON(stateBytes)
	}
	if stateErr == nil {
		r.State = stateBytes
		r.StateStep = r.Steps
	}
	b, encodeErr := oc.Encode(o)
	if encodeErr != nil {
		r.Stop = scenarioCodecError
		executionErr = encodeErr
		return true, executionErr
	}
	b, encodeErr = CanonicalJSON(b)
	if encodeErr != nil {
		r.Stop = scenarioCodecError
		executionErr = encodeErr
		return true, executionErr
	}
	r.Outputs = append(r.Outputs, append(json.RawMessage(nil), b...))
	if stateErr != nil {
		r.Stop = scenarioCodecError
		executionErr = stateErr
		return true, executionErr
	}
	if ctx.Err() != nil {
		r.Stop = scenarioDeadline
		executionErr = ctx.Err()
		return true, executionErr
	}
	if e != nil {
		r.Stop = scenarioError
		executionErr = e
		return true, executionErr
	}
	if done {
		r.Stop = completedState
		return true, executionErr
	}
	return false, nil
}

func validateScenarioStop(r ScenarioRecord) error {
	switch r.Stop {
	case completedState, scenarioError:
		if r.Steps == 0 || len(r.Outputs) != r.Steps {
			return ErrInvalid
		}
	case scenarioStepLimit:
		if r.Steps != r.Plan.MaxSteps || len(r.Outputs) != r.Steps {
			return ErrInvalid
		}
	case scenarioDeadline:
		if len(r.Outputs) != r.Steps {
			return ErrInvalid
		}
	case scenarioCodecError:
		if len(r.Outputs) == r.Steps && r.StateStep == r.Steps {
			return ErrInvalid
		}
		if r.Steps == 0 || (len(r.Outputs) != r.Steps-1 && len(r.Outputs) != r.Steps) ||
			(r.StateStep != r.Steps && r.StateStep != r.Steps-1) {
			return ErrInvalid
		}
	default:
		return ErrUnsupported
	}
	return nil
}
