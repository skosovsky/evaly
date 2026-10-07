package recipes

import (
	"context"
	"errors"

	"github.com/skosovsky/prompty"

	"github.com/skosovsky/evaly"
)

// StreamingTarget consumes and closes a source before returning a typed output.
// Open must bind ctx to its stream. All callbacks are immutable and cooperative.
type StreamingTarget[I, O, E any] struct {
	Identity   string
	Conversion Conversion
	Open       func(context.Context, I, evaly.TrialContext[E]) (*prompty.Stream, error)
	Observe    func(context.Context, *prompty.ResponseChunk, evaly.TrialContext[E]) error
	Project    func(context.Context, *prompty.Response, evaly.TrialContext[E]) (O, error)
	// Complete declares source capture completeness, not provider success.
	Complete func(prompty.StreamStatus) bool
	// CloseTransport releases a host transport after consumption, including failures.
	CloseTransport func(*prompty.Stream) error
}

func (t StreamingTarget[I, O, E]) Validate() error {
	if t.Identity == "" || t.Open == nil || t.Observe == nil || t.Project == nil || t.Complete == nil {
		return evaly.ErrInvalid
	}
	return t.Conversion.Validate()
}

func (t StreamingTarget[I, O, E]) Run(
	ctx context.Context,
	in I,
	trial evaly.TrialContext[E],
) (evaly.TargetResult[O], error) {
	var out evaly.TargetResult[O]
	if err := t.Validate(); err != nil {
		return out, err
	}
	if evaly.ValidatePort(trial.Evidence) != nil {
		return out, evaly.ErrInvalid
	}
	if err := ctx.Err(); err != nil {
		trial.Evidence.MarkIncomplete()
		return out, err
	}
	stream, openErr := t.Open(ctx, in, trial)
	if stream == nil {
		trial.Evidence.MarkIncomplete()
		return out, errors.Join(openErr, evaly.ErrUnsupported)
	}
	var consumeErr error
	if openErr == nil {
		consumeErr = t.consume(ctx, stream, trial)
	}
	closeErr := stream.Close()
	if t.CloseTransport != nil {
		closeErr = errors.Join(closeErr, t.CloseTransport(stream))
	}
	status := stream.Status()
	var usageErr error
	if status.Result != nil {
		out.Usage, usageErr = t.Conversion.Convert(t.Conversion.SourceUnit, status.Result.Usage)
	}
	err := errors.Join(openErr, consumeErr, closeErr, status.Err, ctx.Err(), usageErr)
	if status.State != prompty.StreamCompleted || status.Result == nil {
		err = errors.Join(err, evaly.ErrUnsupported)
	}
	if status.Result != nil && status.Result.Outcome != prompty.OutcomeCompleted {
		err = errors.Join(err, prompty.ErrResponseOutcome)
	}
	if !t.Complete(status) {
		trial.Evidence.MarkIncomplete()
	}
	if err == nil {
		out.Output, err = t.Project(ctx, status.Result, trial)
	}
	err = errors.Join(err, ctx.Err())
	if err != nil {
		trial.Evidence.MarkIncomplete()
	}
	return out, err
}

func (t StreamingTarget[I, O, E]) consume(
	ctx context.Context,
	stream *prompty.Stream,
	trial evaly.TrialContext[E],
) error {
	for frame, err := range stream.Events() {
		if err != nil {
			return err
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if frame == nil {
			return evaly.ErrInvalid
		}
		if err = t.Observe(ctx, frame, trial); err != nil {
			return err
		}
	}
	return nil
}

// JudgeAdapter keeps trusted rubric separate from projected data. Invoke must
// finish its execution and return final cumulative usage even on failure.
type JudgeAdapter[I, O, R any] struct {
	Conversion Conversion
	Invoke     func(context.Context, string, evaly.View[I, O, R]) (*prompty.Response, error)
	Decode     func(context.Context, *prompty.Response) (evaly.Grade, error)
}

func (j JudgeAdapter[I, O, R]) Validate() error {
	if j.Invoke == nil || j.Decode == nil {
		return evaly.ErrInvalid
	}
	return j.Conversion.Validate()
}

func (j JudgeAdapter[I, O, R]) Judge(ctx context.Context, r evaly.JudgeRequest[I, O, R]) (evaly.Grade, error) {
	var grade evaly.Grade
	if err := j.Validate(); err != nil {
		return grade, err
	}
	if r.Instructions == "" {
		return grade, evaly.ErrInvalid
	}
	if err := ctx.Err(); err != nil {
		return grade, err
	}
	response, err := j.Invoke(ctx, r.Instructions, r.Data)
	if response == nil {
		return grade, errors.Join(err, evaly.ErrUnsupported)
	}
	usage, usageErr := j.Conversion.Convert(j.Conversion.SourceUnit, response.Usage)
	err = errors.Join(err, usageErr, ctx.Err())
	if response.Outcome != prompty.OutcomeCompleted {
		err = errors.Join(err, prompty.ErrResponseOutcome)
	}
	if err == nil {
		grade, err = j.Decode(ctx, response)
	}
	grade.Usage = usage
	return grade, errors.Join(err, ctx.Err())
}
