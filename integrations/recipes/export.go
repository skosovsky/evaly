package recipes

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"reflect"
	"sync"

	"github.com/skosovsky/metry/genai"

	"github.com/skosovsky/evaly"
)

// EvaluationRecord retains metric semantics the telemetry DTO cannot represent.
// Artifact remains authoritative; this sidecar must be retained by the host.
type EvaluationRecord struct {
	Evaluation  genai.Evaluation
	Kind        string
	GradeStatus evaly.GradeStatus
	Metric      *evaly.Metric
}

func identity(value any) string {
	data, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// ProjectEvaluations requires a frozen experiment and explicit export policy identity.
func ProjectEvaluations(e evaly.Experiment, policy string) ([]EvaluationRecord, error) {
	if policy == "" {
		return nil, evaly.ErrInvalid
	}
	record := e.Record()
	if _, err := evaly.RestoreExperiment(record); err != nil {
		return nil, err
	}
	records := make([]EvaluationRecord, 0)
	appendResult := func(trial evaly.TrialRecord, g evaly.Grade, kind, name string, score *float64, outcome string, metric *evaly.Metric) {
		status := genai.EvaluationSkipped
		switch g.Status {
		case evaly.Scored:
			status = genai.EvaluationSucceeded
		case evaly.GraderError:
			status = genai.EvaluationFailed
		case evaly.NotApplicable, evaly.InsufficientEvidence:
			status = genai.EvaluationSkipped
		}
		records = append(records, EvaluationRecord{
			Kind: kind, GradeStatus: g.Status, Metric: metric,
			Evaluation: genai.Evaluation{
				ObservationID: identity(
					[]any{record.Manifest.ID, record.Manifest.Revision, trial.ID, g.Revision, kind, name, policy},
				),
				Metric:  genai.EvaluationMetric(name),
				Status:  status,
				Score:   score,
				Outcome: outcome,
				Association: genai.EvaluationAssociation{
					ExperimentID:     record.Manifest.ID,
					DatasetReference: record.Manifest.Dataset,
					CaseID:           trial.CaseID,
					TrialID:          trial.ID,
				},
				Provenance: genai.EvaluationProvenance{
					TargetReference: record.Manifest.Provenance.Target,
					GraderReference: identity(g.Revision),
				},
				ArtifactReference: record.Manifest.Revision,
			},
		})
	}
	for _, trial := range record.Trials {
		for _, g := range trial.Grades {
			for _, a := range g.Assertions {
				outcome := "fail"
				if a.Pass {
					outcome = "pass"
				}
				appendResult(trial, g, "assertion", a.Name, nil, outcome, nil)
			}
			for _, m := range g.Metrics {
				value := m.Value
				appendResult(trial, g, "metric", m.Name, &value, "", &m)
			}
			if g.Status != evaly.Scored {
				appendResult(trial, g, "status", "measurement", nil, "", nil)
			}
		}
		for _, g := range trial.SkippedGraders {
			appendResult(
				trial,
				evaly.Grade{Revision: g.Revision, Status: evaly.InsufficientEvidence},
				"status",
				"measurement",
				nil,
				"",
				nil,
			)
		}
	}
	return records, nil
}

// EvaluationSink requires both a privacy projection and a host atomic delivery
// callback. DeliverOne owns persistence/dedup and must honor ctx.
type EvaluationSink struct {
	Policy       string
	Project      func(context.Context, EvaluationRecord) (EvaluationRecord, error)
	DeliverOne   func(context.Context, EvaluationRecord) error
	Deduplicates bool
}

func (s EvaluationSink) Validate() error {
	if s.Policy == "" || s.Project == nil || s.DeliverOne == nil {
		return evaly.ErrInvalid
	}
	return nil
}
func (s EvaluationSink) Capabilities() evaly.ExportCapabilities {
	return evaly.ExportCapabilities{Deduplication: s.Deduplicates, EnvelopeVersion: 1, BooleanOnly: false}
}
func (s EvaluationSink) Deliver(ctx context.Context, d evaly.DeliveryRecord) error {
	if err := s.Validate(); err != nil {
		return err
	}
	if err := evaly.ValidateEnvelope(d.Artifact); err != nil {
		return err
	}
	if d.Artifact.Kind != "experiment" {
		return evaly.ErrUnsupported
	}
	var record evaly.ExperimentRecord
	if err := json.Unmarshal(d.Artifact.Data, &record); err != nil {
		return evaly.ErrInvalid
	}
	if d.Artifact.ID != record.Manifest.ID {
		return evaly.ErrConflict
	}
	experiment, err := evaly.RestoreExperiment(record)
	if err != nil {
		return err
	}
	results, err := ProjectEvaluations(experiment, s.Policy)
	if err != nil {
		return err
	}
	for i, original := range results {
		if err = ctx.Err(); err != nil {
			return err
		}
		// JSON copy prevents the projection changing source score/metric by pointer.
		raw, marshalErr := json.Marshal(original)
		if marshalErr != nil {
			return evaly.ErrInvalid
		}
		var detached EvaluationRecord
		if json.Unmarshal(raw, &detached) != nil {
			return evaly.ErrInvalid
		}
		projected, projectErr := s.Project(ctx, detached)
		if projectErr != nil {
			return projectErr
		}
		if !sameMeasurement(original, projected) {
			return evaly.ErrConflict
		}
		if err = validateRecord(projected); err != nil {
			return evaly.ErrInvalid
		}
		results[i] = projected
	}
	for _, r := range results {
		if err = ctx.Err(); err != nil {
			return err
		}
		if err = s.DeliverOne(ctx, r); err != nil {
			return err
		}
	}
	return nil
}

func sameMeasurement(a, b EvaluationRecord) bool {
	return a.Kind == b.Kind && a.GradeStatus == b.GradeStatus && reflect.DeepEqual(a.Metric, b.Metric) &&
		a.Evaluation.ObservationID == b.Evaluation.ObservationID && a.Evaluation.Metric == b.Evaluation.Metric &&
		a.Evaluation.Status == b.Evaluation.Status && reflect.DeepEqual(a.Evaluation.Score, b.Evaluation.Score) && a.Evaluation.Outcome == b.Evaluation.Outcome &&
		redacted(a.Evaluation.Association.ExperimentID, b.Evaluation.Association.ExperimentID) &&
		redacted(a.Evaluation.Association.DatasetReference, b.Evaluation.Association.DatasetReference) &&
		redacted(a.Evaluation.Association.CaseID, b.Evaluation.Association.CaseID) &&
		redacted(a.Evaluation.Association.TrialID, b.Evaluation.Association.TrialID) &&
		redacted(a.Evaluation.Provenance.TargetReference, b.Evaluation.Provenance.TargetReference) &&
		redacted(a.Evaluation.Provenance.GraderReference, b.Evaluation.Provenance.GraderReference) &&
		redacted(a.Evaluation.ArtifactReference, b.Evaluation.ArtifactReference) &&
		reflect.DeepEqual(a.Evaluation.TraceReference, b.Evaluation.TraceReference)
}

func redacted(before, after string) bool { return after == "" || before == after }

func validateRecord(r EvaluationRecord) error {
	if err := r.Evaluation.Validate(); err != nil {
		return evaly.ErrInvalid
	}
	switch r.Kind {
	case "metric":
		if r.GradeStatus != evaly.Scored || r.Metric == nil || r.Evaluation.Score == nil ||
			r.Evaluation.Status != genai.EvaluationSucceeded ||
			r.Metric.Value != *r.Evaluation.Score ||
			string(r.Evaluation.Metric) != r.Metric.Name {
			return evaly.ErrInvalid
		}
		return evaly.ValidateGrade(
			evaly.Grade{
				Revision: evaly.GraderRevision{ID: "export", Implementation: "export", Rubric: "export"},
				Status:   evaly.Scored,
				Metrics:  []evaly.Metric{*r.Metric},
			},
		)
	case "assertion":
		if r.GradeStatus != evaly.Scored || r.Metric != nil || r.Evaluation.Score != nil ||
			r.Evaluation.Status != genai.EvaluationSucceeded ||
			(r.Evaluation.Outcome != "pass" && r.Evaluation.Outcome != "fail") {
			return evaly.ErrInvalid
		}
	case "status":
		if r.Metric != nil || r.Evaluation.Score != nil || r.Evaluation.Outcome != "" {
			return evaly.ErrInvalid
		}
		switch r.GradeStatus {
		case evaly.GraderError:
			if r.Evaluation.Status != genai.EvaluationFailed {
				return evaly.ErrInvalid
			}
		case evaly.InsufficientEvidence, evaly.NotApplicable:
			if r.Evaluation.Status != genai.EvaluationSkipped {
				return evaly.ErrInvalid
			}
		default:
			return evaly.ErrInvalid
		}
	default:
		return evaly.ErrUnsupported
	}
	return nil
}

// LocalRegistry is a process-local host reference. Configure Recorder before use.
// It retains projected content to reject ID reuse; it is not a durable outbox.
type LocalRegistry struct {
	mu       sync.Mutex
	Recorder *genai.EvaluationRecorder
	accepted map[string]string
}

func (r *LocalRegistry) Deliver(ctx context.Context, e EvaluationRecord) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.Recorder == nil {
		return evaly.ErrInvalid
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := validateRecord(e); err != nil {
		return evaly.ErrInvalid
	}
	key := e.Evaluation.ObservationID
	digest := identity(e)
	if digest == "" {
		return evaly.ErrInvalid
	}
	if old, ok := r.accepted[key]; ok {
		if old != digest {
			return evaly.ErrConflict
		}
		return nil
	}
	if err := r.Recorder.Record(ctx, e.Evaluation); err != nil {
		return err
	}
	if r.accepted == nil {
		r.accepted = make(map[string]string)
	}
	r.accepted[key] = digest
	return nil
}
