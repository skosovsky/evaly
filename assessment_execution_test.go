package evaly

import (
	"context"
	"errors"
	"testing"
)

type mutationPair struct {
	calls  int
	cancel context.CancelFunc
}

func (j *mutationPair) JudgePair(_ context.Context, r PairRequest[map[string]int]) (PairJudgment, error) {
	j.calls++
	if r.A["v"] != 1 || r.B["v"] != 1 {
		return PairJudgment{}, ErrCorrupt
	}
	r.A["v"] = 99
	r.B["v"] = 99
	if j.cancel != nil {
		j.cancel()
		return PairJudgment{Usage: Usage{Known: true, Units: 3}}, errors.New("paid failure")
	}
	return PairJudgment{Preferred: "A"}, nil
}
func TestPairSnapshotsAndCancellationPreserveUsage(t *testing.T) {
	// Arrange.
	original := map[string]int{"v": 1}
	snapshot, e := SealSnapshot(original, JSONCodec[map[string]int]{ID: "map", Version: "1"})
	if e != nil {
		t.Fatal(e)
	}
	judge := &mutationPair{}
	// Act.
	r := CheckPair(context.Background(), judge, "rubric", snapshot, snapshot)
	// Assert.
	if r.Reviewed != 2 || original["v"] != 1 {
		t.Fatal(r, original)
	}
	// Arrange.
	ctx, cancel := context.WithCancel(context.Background())
	judge = &mutationPair{cancel: cancel}
	// Act.
	r = CheckPair(ctx, judge, "rubric", snapshot, snapshot)
	// Assert.
	if judge.calls != 1 || !r.ForwardDispatched || r.ReverseDispatched || r.Abstention || r.Forward.Usage.Units != 3 ||
		!r.Forward.Usage.Known {
		t.Fatal(r, judge.calls)
	}
	// Act.
	r = CheckPair(ctx, judge, "rubric", snapshot, snapshot)
	// Assert.
	if judge.calls != 1 || r.ForwardDispatched || r.ReverseDispatched {
		t.Fatal(r, judge.calls)
	}
}
func TestAssessPreflightsWholePlanAndFreshFactories(t *testing.T) {
	// Arrange.
	calls := 0
	rev := GraderRevision{ID: "a", Implementation: "1", Rubric: "1"}
	grader := GraderFunc[map[string]int, map[string]int, map[string]int]{
		Identity: rev,
		Evaluate: func(_ context.Context, v View[map[string]int, map[string]int, map[string]int]) (Grade, error) {
			calls++
			if v.Case.Input["v"] != 1 || v.Output["v"] != 1 || (*v.Case.Reference)["v"] != 1 {
				return Grade{}, ErrCorrupt
			}
			v.Case.Input["v"] = 2
			v.Output["v"] = 2
			(*v.Case.Reference)["v"] = 2
			return Grade{Status: Scored, Assertions: []Assertion{{Name: "ok", Pass: true}}}, nil
		},
	}
	factory := func() (View[map[string]int, map[string]int, map[string]int], error) {
		ref := map[string]int{"v": 1}
		return View[map[string]int, map[string]int, map[string]int]{
			Case:   Case[map[string]int, map[string]int]{Input: map[string]int{"v": 1}, Reference: &ref},
			Output: map[string]int{"v": 1},
		}, nil
	}
	duplicate := grader
	// Act.
	result := Assess(
		context.Background(),
		[]Grader[map[string]int, map[string]int, map[string]int]{grader, duplicate},
		factory,
	)
	// Assert.
	if calls != 0 || result[0].Status != GraderError {
		t.Fatal(calls, result)
	}
	// Arrange.
	duplicate.Identity.ID = "b"
	// Act.
	result = Assess(
		context.Background(),
		[]Grader[map[string]int, map[string]int, map[string]int]{grader, duplicate},
		factory,
	)
	// Assert.
	if calls != 2 || result[0].Status != Scored || result[1].Status != Scored {
		t.Fatal(calls, result)
	}
}
func TestAssessmentPartialCanonicalRestore(t *testing.T) {
	// Arrange.
	rev := GraderRevision{ID: "a", Implementation: "1", Rubric: "1"}
	a := Assessment{
		Version:    2,
		Source:     "source",
		View:       "view",
		Mode:       "rescore",
		State:      "partial",
		StopReason: "reconciliation",
		Planned:    []GraderRevision{rev},
		Grades: []Grade{
			{Dispatched: true, Revision: rev, Status: GraderError, Usage: Usage{Known: true, Units: 2}},
		},
	}
	// Act.
	sealed, e := SealAssessment(a)
	if e != nil {
		t.Fatal(e)
	}
	restored, e := RestoreAssessment(sealed)
	// Assert.
	if e != nil || restored.Revision != sealed.Revision || restored.Grades[0].Usage.Units != 2 {
		t.Fatal(restored, e)
	}
	// Act.
	sealed.StopReason = "different"
	_, e = RestoreAssessment(sealed)
	// Assert.
	if !errors.Is(e, ErrCorrupt) {
		t.Fatal(e)
	}
}

func TestAssessFactoryFailureNeverDispatches(t *testing.T) {
	// Arrange.
	calls := 0
	grader := GraderFunc[int, int, int]{
		Identity: GraderRevision{ID: "g", Implementation: "1", Rubric: "1"},
		Evaluate: func(context.Context, View[int, int, int]) (Grade, error) { calls++; return Grade{}, nil },
	}
	factory := func() (View[int, int, int], error) { return View[int, int, int]{}, ErrCorrupt }
	// Act.
	result := Assess(context.Background(), []Grader[int, int, int]{grader}, factory)
	// Assert.
	if calls != 0 || result[0].Dispatched || result[0].Status != GraderError || result[0].Usage.Known {
		t.Fatal(calls, result)
	}
}

func TestSavedViewIsolatesMutatingGraders(t *testing.T) {
	// Arrange.
	capture, e := NewCapture(
		CaptureConfig{Policy: FieldPolicy{ID: "safe"}, KnownKinds: []string{"tool"}, MaxEvents: 2, MaxBytes: 1024},
	)
	if e != nil {
		t.Fatal(e)
	}
	ref := map[string]int{"v": 1}
	original := View[map[string]int, map[string]int, map[string]int]{
		Case: Case[map[string]int, map[string]int]{
			ID:        "case",
			Revision:  "1",
			Input:     map[string]int{"v": 1},
			Reference: &ref,
			Metadata:  map[string]string{"label": "safe"},
		},
		Output:   map[string]int{"v": 1},
		Evidence: capture.Seal(),
	}
	codec := JSONCodec[map[string]int]{ID: "map", Version: "1"}
	saved, e := SaveView(original, "permitted", codec, codec, codec)
	if e != nil {
		t.Fatal(e)
	}
	calls := 0
	makeGrader := func(id string) Grader[map[string]int, map[string]int, map[string]int] {
		return GraderFunc[map[string]int, map[string]int, map[string]int]{
			Identity: GraderRevision{ID: id, Implementation: "1", Rubric: "1"},
			Evaluate: func(_ context.Context, v View[map[string]int, map[string]int, map[string]int]) (Grade, error) {
				calls++
				if v.Case.Input["v"] != 1 || v.Output["v"] != 1 || (*v.Case.Reference)["v"] != 1 ||
					v.Case.Metadata["label"] != "safe" ||
					v.Evidence.State != "sealed" {
					return Grade{}, ErrCorrupt
				}
				v.Case.Input["v"] = 9
				v.Output["v"] = 9
				(*v.Case.Reference)["v"] = 9
				v.Case.Metadata["label"] = "mutated"
				v.Evidence.Coverage["tool"] = false
				return Grade{Status: Scored, Assertions: []Assertion{{Name: "ok", Pass: true}}}, nil
			},
		}
	}
	// Act.
	assessment, e := Rescore(
		context.Background(),
		saved,
		[]Grader[map[string]int, map[string]int, map[string]int]{makeGrader("a"), makeGrader("b")},
		"source",
		"",
		"rescore",
	)
	fresh, viewErr := saved.View()
	// Assert.
	if e != nil || viewErr != nil || calls != 2 || assessment.State != "complete" ||
		assessment.Grades[1].Status != Scored ||
		original.Case.Input["v"] != 1 ||
		ref["v"] != 1 ||
		fresh.Case.Metadata["label"] != "safe" ||
		!fresh.Evidence.Coverage["tool"] {
		t.Fatal(assessment, e, viewErr, original, fresh)
	}
}

func TestGradingPreflightsInvalidBuiltInPorts(t *testing.T) {
	for _, kind := range []string{"function", "llm_port", "instructions", "scripted"} {
		t.Run(kind, func(t *testing.T) {
			// Arrange: a valid first grader must not execute before the later invalid port is discovered.
			calls, factories := 0, 0
			rev := GraderRevision{ID: "invalid", Implementation: "1", Rubric: "1"}
			first := GraderFunc[int, int, int]{
				Identity: GraderRevision{ID: "first", Implementation: "1", Rubric: "1"},
				Evaluate: func(context.Context, View[int, int, int]) (Grade, error) {
					calls++
					return Grade{Status: Scored, Assertions: []Assertion{{Name: "ok", Pass: true}}}, nil
				},
			}
			var invalid Grader[int, int, int]
			switch kind {
			case "function":
				invalid = GraderFunc[int, int, int]{Identity: rev}
			case "llm_port":
				invalid = LLMGrader[int, int, int]{Identity: rev, Instructions: "trusted"}
			case "instructions":
				invalid = LLMGrader[int, int, int]{
					Identity: rev,
					Port: ScriptedJudge[int, int, int]{
						Evaluate: func(context.Context, JudgeRequest[int, int, int]) (Grade, error) { calls++; return Grade{}, nil },
					},
				}
			case "scripted":
				invalid = LLMGrader[int, int, int]{
					Identity:     rev,
					Instructions: "trusted",
					Port:         ScriptedJudge[int, int, int]{},
				}
			}
			capture, e := NewCapture(
				CaptureConfig{
					Policy:     FieldPolicy{ID: "safe"},
					KnownKinds: []string{"tool"},
					MaxEvents:  2,
					MaxBytes:   1024,
				},
			)
			if e != nil {
				t.Fatal(e)
			}
			view := View[int, int, int]{
				Case:     Case[int, int]{ID: "case", Revision: "1", Input: 1},
				Output:   1,
				Evidence: capture.Seal(),
			}
			codec := JSONCodec[int]{ID: "int", Version: "1"}
			saved, e := SaveView(view, "permitted", codec, codec, codec)
			if e != nil {
				t.Fatal(e)
			}
			// Act.
			grades := Assess(
				context.Background(),
				[]Grader[int, int, int]{first, invalid},
				func() (View[int, int, int], error) { factories++; return view, nil },
			)
			_, rescoreErr := Rescore(
				context.Background(),
				saved,
				[]Grader[int, int, int]{first, invalid},
				"source",
				"",
				"rescore",
			)
			// Assert.
			if calls != 0 || factories != 0 || rescoreErr == nil || grades[0].Dispatched || grades[1].Dispatched {
				t.Fatal(calls, factories, rescoreErr, grades)
			}
		})
	}
}

func TestTypedNilCodecPreflightDoesNotCallIdentity(t *testing.T) {
	// Arrange: calling a value-receiver Identity method through this typed nil pointer would panic.
	var nilCodec *JSONCodec[int]
	codec := JSONCodec[int]{ID: "int", Version: "1"}
	capture, e := NewCapture(
		CaptureConfig{Policy: FieldPolicy{ID: "safe"}, KnownKinds: []string{"tool"}, MaxEvents: 2, MaxBytes: 1024},
	)
	if e != nil {
		t.Fatal(e)
	}
	view := View[int, int, int]{
		Case:     Case[int, int]{ID: "case", Revision: "1", Input: 1},
		Output:   1,
		Evidence: capture.Seal(),
	}
	valid, e := SaveView(view, "permitted", codec, codec, codec)
	if e != nil {
		t.Fatal(e)
	}
	// Act.
	_, snapshotErr := SealSnapshot(1, nilCodec)
	_, saveErr := SaveView(view, "permitted", nilCodec, codec, codec)
	_, restoreErr := RestoreSavedView(valid.Record(), codec, nilCodec, codec)
	// Assert.
	if snapshotErr == nil || saveErr == nil || restoreErr == nil {
		t.Fatal(snapshotErr, saveErr, restoreErr)
	}
}
