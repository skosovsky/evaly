// Package wirecontract describes only evaly service envelopes, never host domain codecs.
package wirecontract

import (
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"strings"
	"time"
)

type schema map[string]any

func Schema(t reflect.Type) schema {
	if t == reflect.TypeFor[json.RawMessage]() {
		return schema{}
	}
	if t == reflect.TypeFor[time.Time]() {
		return schema{
			"type":    "string",
			"format":  "date-time",
			"pattern": `^[0-9]{4}-(0[1-9]|1[0-2])-(0[1-9]|[12][0-9]|3[01])T([01][0-9]|2[0-3]):[0-5][0-9]:[0-5][0-9](\.[0-9]+)?(Z|[+-]([01][0-9]|2[0-3]):[0-5][0-9])$`,
		}
	}
	switch t.Kind() {
	case reflect.Pointer:
		base := Schema(t.Elem())
		return schema{"anyOf": []any{base, schema{"type": "null"}}}
	case reflect.String:
		return schema{"type": "string"}
	case reflect.Bool:
		return schema{"type": "boolean"}
	case reflect.Int,
		reflect.Int8,
		reflect.Int16,
		reflect.Int32,
		reflect.Int64,
		reflect.Uint,
		reflect.Uint8,
		reflect.Uint16,
		reflect.Uint32,
		reflect.Uint64:
		bits := t.Bits()
		if t.Kind() >= reflect.Uint && t.Kind() <= reflect.Uint64 {
			maximum := uint64(^uint64(0))
			if bits < 64 {
				maximum = (uint64(1) << bits) - 1
			}
			return schema{"type": "integer", "minimum": uint64(0), "maximum": maximum}
		}
		maximum := int64(^uint64(0) >> 1)
		if bits < 64 {
			maximum = (int64(1) << (bits - 1)) - 1
		}
		return schema{"type": "integer", "minimum": -maximum - 1, "maximum": maximum}
	case reflect.Float32:
		return schema{"type": "number", "minimum": -math.MaxFloat32, "maximum": math.MaxFloat32}
	case reflect.Float64:
		return schema{"type": "number", "minimum": -math.MaxFloat64, "maximum": math.MaxFloat64}
	case reflect.Slice, reflect.Array:
		return schema{"type": []string{"array", "null"}, "items": Schema(t.Elem())}
	case reflect.Map:
		return schema{"type": []string{"object", "null"}, "additionalProperties": Schema(t.Elem())}
	case reflect.Struct:
		properties := map[string]any{}
		required := []string{}
		for field := range t.Fields() {
			field := field
			if !field.IsExported() {
				continue
			}
			tag := strings.Split(field.Tag.Get("json"), ",")
			name := tag[0]
			if name == "-" {
				continue
			}
			if name == "" {
				name = field.Name
			}
			value := Schema(field.Type)
			constrain(t.PkgPath(), t.Name(), field.Name, value)
			properties[name] = value
			optional := false
			for _, option := range tag[1:] {
				if option == "omitempty" {
					optional = true
				}
			}
			if !optional {
				required = append(required, name)
			}
		}
		document := schema{
			"type":                 "object",
			"properties":           properties,
			"required":             required,
			"additionalProperties": false,
		}
		if t.Name() == "Grade" {
			document["allOf"] = []any{
				schema{
					"if": schema{
						"properties": schema{
							"status": schema{
								"enum": []string{"grader_error", "not_applicable", "insufficient_evidence"},
							},
						},
					},
					"then": schema{
						"not": schema{
							"anyOf": []any{
								schema{"required": []string{"metrics"}},
								schema{"required": []string{"assertions"}},
							},
						},
					},
				},
			}
		}
		return document
	default:
		panic(fmt.Sprintf("unsupported wire type %v", t))
	}
}
func constrain(pkg, parent, name string, s schema) {
	if name == "Version" && s["type"] == "integer" {
		s["const"] = 1
		s["x-evaly-version"] = true
		if parent == "ScenarioRecord" || parent == "Comparison" || parent == "Request" || parent == "Response" {
			s["const"] = 2
		}
		if parent == "ExperimentManifest" || parent == "Assessment" || parent == "Result" {
			s["const"] = 3
		}
		if parent == "Result" && pkg == "github.com/skosovsky/evaly/optimizer" {
			s["const"] = 4
		}
	}
	if parent == "CalibrationCounts" {
		s["minimum"] = 0
	}
	switch parent + "." + name {
	case "Envelope.Kind":
		s["enum"] = []string{
			"dataset",
			"experiment",
			"comparison",
			"evidence",
			"observation",
			"search",
			"view",
			"scenario",
			"assessment",
			"candidate",
			"calibration",
		}
	case "Envelope.ID":
		s["pattern"] = `^[a-zA-Z0-9_-]{1,128}$`
	case "Envelope.Checksum":
		s["pattern"] = `^[0-9a-f]{64}$`
	case "Envelope.Data":
		s["type"] = "object"
	case "DatasetRecord.State":
		s["const"] = "sealed"
	case "EvidenceRecord.State", "ExperimentManifest.State":
		s["enum"] = []string{"sealed", "incomplete"}
	case "ExperimentManifest.Mode":
		s["const"] = "controlled"
	case "Grade.Status":
		s["enum"] = []string{"scored", "not_applicable", "insufficient_evidence", "grader_error"}
	case "TrialRecord.GradingState":
		s["enum"] = []string{"complete", "partial"}
	case "TrialRecord.Status":
		s["enum"] = []string{
			"completed",
			"target_error",
			"setup_error",
			"cancelled",
			"budget_exhausted",
			"infrastructure_stop",
		}
	case "LifecycleIdentity.Isolation":
		s["enum"] = []string{"isolated", "serial/shared"}
	case "Metric.Direction", "ObjectiveIdentity.Direction":
		s["enum"] = []string{"higher", "lower"}
	case "CleanupStatus.State":
		s["enum"] = []string{"not_needed", "completed", "failed"}
	case "ComparisonPolicy.ObjectiveKind":
		s["enum"] = []string{"assertion", "numeric"}
	case "Comparison.Verdict":
		s["enum"] = []string{"pass", "fail", "inconclusive", "invalid_comparison"}
	case "Generation.Mode", "ScenarioPlan.Mode":
		s["enum"] = []string{"search", "replay"}
	case "ScenarioRecord.Stop":
		s["enum"] = []string{"completed", "step_limit", "deadline", "error", "codec_error"}
	case "Response.Status":
		s["enum"] = []string{"completed", "target_error"}
	case "Assessment.State":
		s["enum"] = []string{"complete", "partial"}
	case "Assessment.Mode":
		s["enum"] = []string{"rescore", "observation"}
	case "RunPlan.AssertionPolicy":
		s["enum"] = []string{"all", "any"}
	case "Usage.Units", "Reservation.Units":
		s["minimum"] = 0
	case "RunPlan.Repeats", "RunPlan.Concurrency", "RunPlan.MaxAttempts", "ScenarioPlan.MaxSteps":
		s["minimum"] = 1
	case "RunPlan.Timeout", "RunPlan.CleanupTimeout", "ScenarioPlan.Timeout":
		s["minimum"] = 1
	case "RunPlan.DispatchUnits", "RunPlan.GraderUnits":
		s["minimum"] = 0
	case "GatePolicy.MinimumCoverage", "GatePolicy.MinimumMatchedCoverage":
		s["minimum"] = 0
		s["maximum"] = 1
	case "Event.Sequence":
		s["minimum"] = 1
	case "EvidenceRecord.Errors", "EvidenceRecord.Gaps", "Grade.Reasons", "Grade.EvidenceRefs":
		s["maxItems"] = 32
	case "Comparison.Unit":
		s["const"] = "case"
	case "GatePolicy.MinimumMatchedCases":
		s["minimum"] = 1
	case "GatePolicy.MaximumRegression":
		s["minimum"] = 0
	case "CalibrationRate.Numerator", "CalibrationRate.Denominator":
		s["minimum"] = 0
	case "CalibrationRate.Value":
		branches := s["anyOf"].([]any)
		branches[0].(schema)["minimum"] = 0
		branches[0].(schema)["maximum"] = 1
	case "Result.MaximumRounds", "Result.MaximumCandidates":
		if pkg == "github.com/skosovsky/evaly/optimizer" {
			s["minimum"] = 1
			s["maximum"] = 10000
		}
	case "Result.TimeoutNanoseconds":
		if pkg == "github.com/skosovsky/evaly/optimizer" {
			s["minimum"] = 1
		}
	case "Result.ProposalUnits", "Result.EvaluationUnits":
		if pkg == "github.com/skosovsky/evaly/optimizer" {
			s["minimum"] = 0
		}
	case "Result.State":
		if pkg == "github.com/skosovsky/evaly/optimizer" {
			s["enum"] = []string{"completed", "stopped"}
		}
	case "Result.TieRevision":
		if pkg == "github.com/skosovsky/evaly/optimizer" {
			s["const"] = "candidate-revision-lexical-v1"
		}
	case "Evaluation.Round", "Round.Index":
		s["minimum"] = 0
	case "Evaluation.State":
		s["enum"] = []string{"invalid", "failed", "incomplete", "evaluated"}
	case "Round.State":
		s["enum"] = []string{"completed", "stopped", "failed"}
	case "CalibrationReport.Groups":
		s["type"] = "array"
	case "PairSchedule.Slots":
		s["type"] = "array"
	case "PairSchedule.Revision":
		s["const"] = "case-repeat-v1"
	case "PairSlot.Repeat":
		s["minimum"] = 0
	case "PairSchedule.Concurrency":
		s["minimum"] = 1
	case "GatePolicy.BootstrapSamples":
		s["minimum"] = 100
		s["maximum"] = 100000
	}
}
