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

func Schema(t reflect.Type) map[string]any { return typeSchema(t) }

func typeSchema(t reflect.Type) schema {
	if t == reflect.TypeFor[json.RawMessage]() {
		return schema{}
	}
	if t == reflect.TypeFor[time.Time]() {
		return schema{
			schemaType: schemaString,
			"format":   "date-time",
			"pattern":  `^[0-9]{4}-(0[1-9]|1[0-2])-(0[1-9]|[12][0-9]|3[01])T([01][0-9]|2[0-3]):[0-5][0-9]:[0-5][0-9](\.[0-9]+)?(Z|[+-]([01][0-9]|2[0-3]):[0-5][0-9])$`,
		}
	}
	switch t.Kind() {
	case reflect.Pointer:
		base := typeSchema(t.Elem())
		return schema{"anyOf": []any{base, schema{schemaType: schemaNull}}}
	case reflect.String:
		return schema{schemaType: schemaString}
	case reflect.Bool:
		return schema{schemaType: "boolean"}
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
		return integerSchema(t)
	case reflect.Float32:
		return schema{schemaType: schemaNumber, schemaMinimum: -math.MaxFloat32, schemaMaximum: math.MaxFloat32}
	case reflect.Float64:
		return schema{schemaType: schemaNumber, schemaMinimum: -math.MaxFloat64, schemaMaximum: math.MaxFloat64}
	case reflect.Slice, reflect.Array:
		return schema{schemaType: []string{schemaArray, schemaNull}, "items": typeSchema(t.Elem())}
	case reflect.Map:
		return schema{schemaType: []string{schemaObject, schemaNull}, "additionalProperties": typeSchema(t.Elem())}
	case reflect.Struct:
		return structSchema(t)
	case reflect.Invalid,
		reflect.Uintptr,
		reflect.Complex64,
		reflect.Complex128,
		reflect.Chan,
		reflect.Func,
		reflect.Interface,
		reflect.UnsafePointer:
		fallthrough
	default:
		panic(fmt.Sprintf("unsupported wire type %v", t))
	}
}
func constrain(pkg, parent, name string, s schema) {
	if name == "Version" && s[schemaType] == schemaInteger {
		s["const"] = 1
		s["x-evaly-version"] = true
		if parent == "ScenarioRecord" || parent == "Comparison" || parent == "Request" || parent == "Response" {
			s["const"] = 2
		}
		if parent == "ExperimentManifest" || parent == "Assessment" || parent == schemaResult {
			s["const"] = 3
		}
		if parent == schemaResult && pkg == optimizerPackagePath {
			s["const"] = 4
		}
	}
	if parent == "CalibrationCounts" {
		s[schemaMinimum] = 0
	}
	constrainEnvelope(parent, name, s)
	constrainStates(parent, name, s)
	constrainExecution(parent, name, s)
	constrainMeasurement(parent, name, s)
	constrainOptimizer(pkg, parent, name, s)
}

func constrainEnvelope(parent, name string, s schema) {
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
		s[schemaType] = schemaObject
	}
}

func constrainStates(parent, name string, s schema) {
	switch parent + "." + name {
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
		s["enum"] = []string{schemaCompleted, "target_error",
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
		s["enum"] = []string{"not_needed", schemaCompleted, schemaFailed}
	case "ComparisonPolicy.ObjectiveKind":
		s["enum"] = []string{"assertion", "numeric"}
	case "Comparison.Verdict":
		s["enum"] = []string{"pass", "fail", "inconclusive", "invalid_comparison"}
	case "Generation.Mode", "ScenarioPlan.Mode":
		s["enum"] = []string{"search", "replay"}
	case "ScenarioRecord.Stop":
		s["enum"] = []string{schemaCompleted, "step_limit", "deadline", "error", "codec_error"}
	case "Response.Status":
		s["enum"] = []string{schemaCompleted, "target_error"}
	case "Assessment.State":
		s["enum"] = []string{"complete", "partial"}
	case "Assessment.Mode":
		s["enum"] = []string{"rescore", "observation"}
	case "Evaluation.Round", "Round.Index":
		s[schemaMinimum] = 0
	case "Evaluation.State":
		s["enum"] = []string{"invalid", schemaFailed, "incomplete", "evaluated"}
	case "Round.State":
		s["enum"] = []string{schemaCompleted, "stopped", schemaFailed}
	case "CalibrationReport.Groups":
		s[schemaType] = schemaArray
	case "PairSchedule.Slots":
		s[schemaType] = schemaArray
	case "PairSchedule.Revision":
		s["const"] = "case-repeat-v1"
	case "PairSlot.Repeat":
		s[schemaMinimum] = 0
	case "PairSchedule.Concurrency":
		s[schemaMinimum] = 1
	}
}

func constrainExecution(parent, name string, s schema) {
	switch parent + "." + name {
	case "RunPlan.AssertionPolicy":
		s["enum"] = []string{"all", "any"}
	case "Usage.Units", "Reservation.Units":
		s[schemaMinimum] = 0
	case "RunPlan.Repeats", "RunPlan.Concurrency", "RunPlan.MaxAttempts", "ScenarioPlan.MaxSteps":
		s[schemaMinimum] = 1
	case "RunPlan.Timeout", "RunPlan.CleanupTimeout", "ScenarioPlan.Timeout":
		s[schemaMinimum] = 1
	case "RunPlan.DispatchUnits", "RunPlan.GraderUnits":
		s[schemaMinimum] = 0
	case "Event.Sequence":
		s[schemaMinimum] = 1
	case "EvidenceRecord.Errors", "EvidenceRecord.Gaps", "Grade.Reasons", "Grade.EvidenceRefs":
		s["maxItems"] = 32
	}
}

func constrainMeasurement(parent, name string, s schema) {
	switch parent + "." + name {
	case "GatePolicy.MinimumCoverage", "GatePolicy.MinimumMatchedCoverage":
		s[schemaMinimum] = 0
		s[schemaMaximum] = 1
	case "Comparison.Unit":
		s["const"] = "case"
	case "GatePolicy.MinimumMatchedCases":
		s[schemaMinimum] = 1
	case "GatePolicy.MaximumRegression":
		s[schemaMinimum] = 0
	case "CalibrationRate.Numerator", "CalibrationRate.Denominator":
		s[schemaMinimum] = 0
	case "CalibrationRate.Value":
		branches, ok := s["anyOf"].([]any)
		if !ok || len(branches) == 0 {
			return
		}
		branch, ok := branches[0].(schema)
		if !ok {
			return
		}
		branch[schemaMinimum] = 0
		branch[schemaMaximum] = 1
	case "GatePolicy.BootstrapSamples":
		s[schemaMinimum] = 100
		s[schemaMaximum] = 100000
	}
}

func constrainOptimizer(pkg, parent, name string, s schema) {
	switch parent + "." + name {
	case "Result.MaximumRounds", "Result.MaximumCandidates":
		if pkg == optimizerPackagePath {
			s[schemaMinimum] = 1
			s[schemaMaximum] = 10000
		}
	case "Result.TimeoutNanoseconds":
		if pkg == optimizerPackagePath {
			s[schemaMinimum] = 1
		}
	case "Result.ProposalUnits", "Result.EvaluationUnits":
		if pkg == optimizerPackagePath {
			s[schemaMinimum] = 0
		}
	case "Result.State":
		if pkg == optimizerPackagePath {
			s["enum"] = []string{schemaCompleted, "stopped"}
		}
	case "Result.TieRevision":
		if pkg == optimizerPackagePath {
			s["const"] = "candidate-revision-lexical-v1"
		}
	}
}

func integerSchema(t reflect.Type) schema {
	bits := t.Bits()
	if t.Kind() >= reflect.Uint && t.Kind() <= reflect.Uint64 {
		maximum := ^uint64(0)
		if bits < integerBits {
			maximum = (uint64(1) << bits) - 1
		}
		return schema{schemaType: schemaInteger, schemaMinimum: uint64(0), schemaMaximum: maximum}
	}
	maximum := int64(^uint64(0) >> 1)
	if bits < integerBits {
		maximum = (int64(1) << (bits - 1)) - 1
	}
	return schema{schemaType: schemaInteger, schemaMinimum: -maximum - 1, schemaMaximum: maximum}
}

func structSchema(t reflect.Type) schema {
	properties := map[string]any{}
	required := []string{}
	for field := range t.Fields() {
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
		value := typeSchema(field.Type)
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
		schemaType:             schemaObject,
		"properties":           properties,
		schemaRequired:         required,
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
							schema{schemaRequired: []string{"metrics"}},
							schema{schemaRequired: []string{"assertions"}},
						},
					},
				},
			},
		}
	}
	return document
}
