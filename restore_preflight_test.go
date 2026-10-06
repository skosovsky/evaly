package evaly_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/skosovsky/evaly"
	"github.com/skosovsky/evaly/optimizer"
)

type preflightCodec struct {
	codec       evaly.JSONCodec[int]
	validateErr error
	calls       *atomic.Int64
}

func (p *preflightCodec) Validate() error               { return p.validateErr }
func (p *preflightCodec) Identity() evaly.CodecIdentity { p.calls.Add(1); return p.codec.Identity() }
func (p *preflightCodec) Encode(v int) ([]byte, error)  { p.calls.Add(1); return p.codec.Encode(v) }
func (p *preflightCodec) Decode(b []byte) (int, error)  { p.calls.Add(1); return p.codec.Decode(b) }

type preflightStep struct{}

func (preflightStep) Revision() string { return "step-v1" }
func (preflightStep) Step(_ context.Context, state int, _ evaly.ScenarioContext) (int, int, bool, error) {
	return state + 1, state + 1, true, nil
}

func preflightRecords(t *testing.T) (evaly.Dataset[int, int], evaly.ScenarioRecord) {
	t.Helper()
	codec := evaly.JSONCodec[int]{ID: "int", Version: "1"}
	dataset, err := (evaly.DatasetDraft[int, int]{Selection: "all", Cases: []evaly.Case[int, int]{{ID: "case", Input: 1}}}).Seal(
		codec,
		codec,
	)
	if err != nil {
		t.Fatal(err)
	}
	scenario, err := evaly.RunScenario(t.Context(), preflightStep{}, 0,
		evaly.ScenarioPlan{Mode: "replay", MaxSteps: 1, Timeout: time.Second}, codec, codec)
	if err != nil {
		t.Fatal(err)
	}
	return dataset, scenario
}

func TestRestoreStructuralPreflightBeforeCodecDispatch(t *testing.T) {
	dataset, scenario := preflightRecords(t)
	for _, position := range []int{0, 1} {
		for _, variant := range []string{"nil", "typed_nil", "validator"} {
			t.Run(variant+string(rune('0'+position)), func(t *testing.T) {
				// Arrange: valid sealed records and instrumented codecs.
				var calls atomic.Int64
				contractErr := errors.New("local configuration rejected")
				var missing *preflightCodec
				bad := &preflightCodec{validateErr: contractErr, calls: &calls}
				ports := []evaly.Codec[int]{
					&preflightCodec{codec: evaly.JSONCodec[int]{ID: "int", Version: "1"}, calls: &calls},
					&preflightCodec{codec: evaly.JSONCodec[int]{ID: "int", Version: "1"}, calls: &calls},
				}
				expected := evaly.ErrInvalid
				switch variant {
				case "nil":
					ports[position] = nil
				case "typed_nil":
					ports[position] = missing
				case "validator":
					ports[position] = bad
					expected = contractErr
				}
				// Act
				_, datasetErr := evaly.RestoreDataset(dataset.Record(), ports[0], ports[1])
				_, scenarioErr := evaly.RestoreScenario(scenario, ports[0], ports[1])
				// Assert: no Identity/Encode/Decode method was invoked on either peer.
				if !errors.Is(datasetErr, expected) || !errors.Is(scenarioErr, expected) || calls.Load() != 0 {
					t.Fatal(datasetErr, scenarioErr, calls.Load())
				}
			})
		}
	}
}

func TestRestoreCodecIdentityAndValidRoundtrip(t *testing.T) {
	// Arrange
	dataset, scenario := preflightRecords(t)
	valid := evaly.JSONCodec[int]{ID: "int", Version: "1"}
	wrong := evaly.JSONCodec[int]{ID: "other", Version: "1"}
	// Act
	_, datasetErr := evaly.RestoreDataset(dataset.Record(), wrong, valid)
	_, scenarioErr := evaly.RestoreScenario(scenario, valid, wrong)
	restored, validDatasetErr := evaly.RestoreDataset(dataset.Record(), valid, valid)
	output, validScenarioErr := evaly.RestoreScenario(scenario, valid, valid)
	_, malformedIdentity := evaly.RestoreDataset(dataset.Record(), evaly.JSONCodec[int]{}, valid)
	// Assert
	if !errors.Is(datasetErr, evaly.ErrUnsupported) || !errors.Is(scenarioErr, evaly.ErrUnsupported) ||
		validDatasetErr != nil || validScenarioErr != nil || restored.Revision() != dataset.Revision() ||
		output.State != 1 || len(output.Outputs) != 1 || !errors.Is(malformedIdentity, evaly.ErrInvalid) {
		t.Fatal(datasetErr, scenarioErr, validDatasetErr, validScenarioErr, malformedIdentity, output)
	}
}

type preflightStore struct {
	validateErr error
	calls       *atomic.Int64
}

func (s *preflightStore) Validate() error { return s.validateErr }
func (s *preflightStore) Put(context.Context, evaly.Envelope) error {
	s.calls.Add(1)
	return evaly.ErrClosed
}
func (s *preflightStore) Get(context.Context, string) (evaly.Envelope, error) {
	s.calls.Add(1)
	return evaly.Envelope{}, evaly.ErrClosed
}

func preflightSavedView(t *testing.T) evaly.SavedView[int, int, int] {
	t.Helper()
	capture, err := evaly.NewCapture(
		evaly.CaptureConfig{
			Policy:     evaly.FieldPolicy{ID: "none"},
			KnownKinds: []string{"tool"},
			MaxEvents:  1,
			MaxBytes:   1024,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	codec := evaly.JSONCodec[int]{ID: "int", Version: "1"}
	view, err := evaly.SaveView(
		evaly.View[int, int, int]{
			Case:     evaly.Case[int, int]{ID: "case", Revision: "case-v1", Input: 1},
			Output:   2,
			Evidence: capture.Seal(),
		},
		"safe",
		codec,
		codec,
		codec,
	)
	if err != nil {
		t.Fatal(err)
	}
	return view
}

func TestArtifactHelpersRejectInvalidStoresBeforeEffects(t *testing.T) {
	// Arrange: supported sealed artifacts, then absent/invalid store capabilities.
	experiment, err := evaly.Run(t.Context(), config(t, 1))
	if err != nil {
		t.Fatal(err)
	}
	view := preflightSavedView(t)
	codec := evaly.JSONCodec[int]{ID: "int", Version: "1"}
	var calls atomic.Int64
	contractErr := errors.New("store configuration rejected")
	var missing *preflightStore
	variants := []struct {
		name     string
		port     evaly.ArtifactStore
		expected error
	}{
		{"nil", nil, evaly.ErrInvalid}, {"typed_nil", missing, evaly.ErrInvalid},
		{"validator", &preflightStore{validateErr: contractErr, calls: &calls}, contractErr},
	}
	for _, variant := range variants {
		t.Run(variant.name, func(t *testing.T) {
			// Act
			saveExperiment := evaly.SaveExperiment(t.Context(), variant.port, experiment)
			_, loadExperiment := evaly.LoadExperiment(t.Context(), variant.port, experiment.ID())
			saveView := evaly.SaveSavedView(t.Context(), variant.port, "view", view)
			_, loadView := evaly.LoadSavedView(t.Context(), variant.port, "view", codec, codec, codec)
			// Assert
			for _, failure := range []error{saveExperiment, loadExperiment, saveView, loadView} {
				if !errors.Is(failure, variant.expected) {
					t.Fatal(failure, variant.expected)
				}
			}
			if calls.Load() != 0 {
				t.Fatal(calls.Load())
			}
		})
	}
}

func TestSavedViewCodecPreflightBeforeStoreGet(t *testing.T) {
	view := preflightSavedView(t)
	for position := range 3 {
		for _, variant := range []string{"nil", "typed_nil", "validator"} {
			t.Run(variant+string(rune('0'+position)), func(t *testing.T) {
				// Arrange
				var calls atomic.Int64
				store := &preflightStore{calls: &calls}
				codec := evaly.JSONCodec[int]{ID: "int", Version: "1"}
				ports := []evaly.Codec[int]{codec, codec, codec}
				var missing *preflightCodec
				expected := evaly.ErrInvalid
				switch variant {
				case "nil":
					ports[position] = nil
				case "typed_nil":
					ports[position] = missing
				case "validator":
					expected = errors.New("codec configuration rejected")
					ports[position] = &preflightCodec{calls: &calls, validateErr: expected}
				}
				// Act
				_, loadErr := evaly.LoadSavedView(t.Context(), store, "view", ports[0], ports[1], ports[2])
				_, restoreErr := evaly.RestoreSavedView(view.Record(), ports[0], ports[1], ports[2])
				// Assert
				if !errors.Is(loadErr, expected) || !errors.Is(restoreErr, expected) || calls.Load() != 0 {
					t.Fatal(loadErr, restoreErr, calls.Load())
				}
			})
		}
	}
}

func TestCandidateCodecStructuralPreflight(t *testing.T) {
	// Arrange
	codec := evaly.JSONCodec[int]{ID: "int", Version: "1"}
	candidate, err := optimizer.Seal("candidate", "", "algorithm", 1, codec)
	if err != nil {
		t.Fatal(err)
	}
	var calls atomic.Int64
	var missing *preflightCodec
	for _, port := range []evaly.Codec[int]{nil, missing, &preflightCodec{calls: &calls, validateErr: evaly.ErrInvalid}, evaly.JSONCodec[int]{}} {
		// Act
		_, restoreErr := optimizer.RestoreCandidate(candidate.Record(), port)
		// Assert
		if !errors.Is(restoreErr, evaly.ErrInvalid) || calls.Load() != 0 {
			t.Fatal(restoreErr, calls.Load())
		}
	}
	_, unsupported := optimizer.RestoreCandidate(candidate.Record(), evaly.JSONCodec[int]{ID: "other", Version: "1"})
	if !errors.Is(unsupported, evaly.ErrUnsupported) {
		t.Fatal(unsupported)
	}
}
