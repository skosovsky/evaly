// Command schemagen emits JSON wire schemas from explicit envelope types.
// This is development tooling, never a domain codec or artifact store path.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/skosovsky/evaly"
	"github.com/skosovsky/evaly/adapters/httpjson"
	"github.com/skosovsky/evaly/observation"
	"github.com/skosovsky/evaly/optimizer"
)

type schema map[string]any

func main() {
	directory := "schemas"
	if len(os.Args) == 2 {
		directory = os.Args[1]
	}
	types := map[string]any{
		"envelope":           evaly.Envelope{},
		"dataset":            evaly.DatasetRecord{},
		"experiment":         evaly.ExperimentRecord{},
		"evidence":           evaly.EvidenceRecord{},
		"comparison":         evaly.Comparison{},
		"view":               evaly.SavedViewRecord{},
		"scenario":           evaly.ScenarioRecord{},
		"assessment":         evaly.Assessment{},
		"observation":        observation.Record{},
		"observation-result": observation.Result{},
		"search":             optimizer.Result{},
		"candidate":          optimizer.CandidateRecord{},
		"http-request":       httpjson.Request{},
		"http-response":      httpjson.Response{},
	}
	for name, value := range types {
		document := fromType(reflect.TypeOf(value))
		document["$schema"] = "https://json-schema.org/draft/2020-12/schema"
		version := 1
		switch name {
		case "experiment", "scenario", "assessment", "observation-result", "search":
			version = 2
		}
		document["$id"] = "urn:evaly:" + name + ":" + strconv.Itoa(version)
		b, e := json.MarshalIndent(document, "", "  ")
		if e != nil {
			panic(e)
		}
		if e = os.WriteFile(
			filepath.Join(directory, name+"-v"+strconv.Itoa(version)+".json"),
			append(b, '\n'),
			0600,
		); e != nil {
			panic(e)
		}
	}
}
func fromType(t reflect.Type) schema {
	if t == reflect.TypeFor[json.RawMessage]() {
		return schema{}
	}
	if t == reflect.TypeFor[time.Time]() {
		return schema{"type": "string", "format": "date-time"}
	}
	switch t.Kind() {
	case reflect.Pointer:
		base := fromType(t.Elem())
		return schema{"anyOf": []any{base, schema{"type": "null"}}}
	case reflect.String:
		return schema{"type": "string"}
	case reflect.Bool:
		return schema{"type": "boolean"}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint64:
		return schema{"type": "integer"}
	case reflect.Float32, reflect.Float64:
		return schema{"type": "number"}
	case reflect.Slice, reflect.Array:
		return schema{"type": []string{"array", "null"}, "items": fromType(t.Elem())}
	case reflect.Map:
		return schema{"type": []string{"object", "null"}, "additionalProperties": fromType(t.Elem())}
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
			value := fromType(field.Type)
			constrain(t.Name(), field.Name, value)
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
func constrain(parent, name string, s schema) {
	if name == "Version" && s["type"] == "integer" {
		s["const"] = 1
		if parent == "ExperimentManifest" || parent == "ScenarioRecord" || parent == "Assessment" ||
			parent == "Result" {
			s["const"] = 2
		}
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
	case "Metric.Direction":
		s["enum"] = []string{"higher", "lower"}
	case "CleanupStatus.State":
		s["enum"] = []string{"not_needed", "completed", "failed"}
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
	case "GatePolicy.MinimumCoverage", "GatePolicy.MinimumQuality", "GatePolicy.MaximumRegression":
		s["minimum"] = 0
		s["maximum"] = 1
	case "Event.Sequence":
		s["minimum"] = 1
	case "EvidenceRecord.Errors", "EvidenceRecord.Gaps", "Grade.Reasons", "Grade.EvidenceRefs":
		s["maxItems"] = 32
	case "Comparison.Unit":
		s["const"] = "case"
	case "Comparison.Metric":
		s["const"] = "assertion_pass_rate"
	}
}
