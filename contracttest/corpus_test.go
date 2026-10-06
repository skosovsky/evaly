package contracttest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/skosovsky/evaly"
	"github.com/skosovsky/evaly/adapters/httpjson"
	"github.com/skosovsky/evaly/internal/fixtures"
	"github.com/skosovsky/evaly/observation"
	"github.com/skosovsky/evaly/optimizer"
)

// The independent engine consumes the same bytes as the runtime. Checksum,
// revisions and state-machine relations belong to Restore, after DecodeWire.
func runtimeDecode(name string, raw []byte) error {
	switch name {
	case "envelope":
		_, e := evaly.DecodeWire[evaly.Envelope](raw)
		return e
	case "dataset":
		_, e := evaly.DecodeWire[evaly.DatasetRecord](raw)
		return e
	case "experiment":
		_, e := evaly.DecodeWire[evaly.ExperimentRecord](raw)
		return e
	case "evidence":
		_, e := evaly.DecodeWire[evaly.EvidenceRecord](raw)
		return e
	case "calibration":
		_, e := evaly.DecodeWire[evaly.CalibrationReport](raw)
		return e
	case "comparison-policy":
		_, e := evaly.DecodeWire[evaly.ComparisonPolicy](raw)
		return e
	case "comparison":
		_, e := evaly.DecodeWire[evaly.Comparison](raw)
		return e
	case "view":
		_, e := evaly.DecodeWire[evaly.SavedViewRecord](raw)
		return e
	case "assessment":
		_, e := evaly.DecodeWire[evaly.Assessment](raw)
		return e
	case "scenario":
		_, e := evaly.DecodeWire[evaly.ScenarioRecord](raw)
		return e
	case "candidate":
		_, e := evaly.DecodeWire[optimizer.CandidateRecord](raw)
		return e
	case "observation":
		_, e := evaly.DecodeWire[observation.Record](raw)
		return e
	case "observation-result":
		_, e := evaly.DecodeWire[observation.Result](raw)
		return e
	case "search":
		_, e := evaly.DecodeWire[optimizer.Result](raw)
		return e
	case "http-request":
		_, e := evaly.DecodeWire[httpjson.Request](raw)
		return e
	case "http-response":
		_, e := evaly.DecodeWire[httpjson.Response](raw)
		return e
	default:
		return fmt.Errorf("unregistered corpus kind %s", name)
	}
}

func decodeDocument(t *testing.T, raw []byte) any {
	t.Helper()
	var doc any
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	if e := d.Decode(&doc); e != nil {
		t.Fatal(e)
	}
	return doc
}
func wireSchemaPath(name string) string {
	version := 1
	switch name {
	case "scenario", "comparison-policy":
		version = 2
	case "experiment", "assessment", "observation-result", "http-request", "http-response", "comparison":
		version = 3
	case "search":
		version = 6
	case "candidate":
		version = 2
	}
	return filepath.Join("..", "schemas", fmt.Sprintf("%s-v%d.json", name, version))
}

type mutation struct {
	label  string
	path   []any
	value  any
	remove bool
}

func appendPath(path []any, key any) []any { return append(append([]any(nil), path...), key) }
func schemaMutations(doc any, schema map[string]any, path []any) []mutation {
	var out []mutation
	if branches, ok := schema["anyOf"].([]any); ok {
		// Nullable pointer branches preserve the non-null structural contract.
		for _, b := range branches {
			s := b.(map[string]any)
			if s["type"] != "null" {
				return schemaMutations(doc, s, path)
			}
		}
	}
	out = append(out, schemaBoundMutations(schema, path)...)
	switch v := doc.(type) {
	case map[string]any:
		out = append(out, objectMutations(v, schema, path)...)
	case []any:
		item, ok := schema["items"].(map[string]any)
		if ok && len(v) > 0 {
			out = append(out, schemaMutations(v[0], item, appendPath(path, 0))...)
		}
	case string:
		out = append(out, mutation{"zero", path, "", false})
		if schema["format"] == "date-time" {
			out = append(out, mutation{"date_time", path, "2026-99-99T00:00:00Z", false})
			for _, value := range []string{"2023-02-29T00:00:00Z", "2024-02-29T00:00:00Z", "2026-01-01t00:00:00z", "2026-01-01T00:00:60Z", "2026-01-01T00:00:00+24:00", "2026-01-01T00:00:00+23:59", "2026-01-01T00:00:00+00:60"} {
				out = append(out, mutation{"date_time_edge", path, value, false})
			}
		}
	case json.Number:
		out = append(
			out,
			mutation{"zero", path, json.Number("0"), false},
			mutation{"wrong_type", path, "1", false},
			mutation{"overflow", path, json.Number("1e400"), false},
			mutation{"negative_overflow", path, json.Number("-1e400"), false},
			mutation{"fractional", path, json.Number("1.5"), false},
			mutation{"decimal_integer", path, json.Number("1.0"), false},
			mutation{"negative_underflow", path, json.Number("-1e-10000"), false},
			mutation{"zero_exponent", path, json.Number("0e10000"), false},
		)
	case bool:
		out = append(out, mutation{"zero", path, false, false})
	}
	return out
}
func applyMutation(doc any, m mutation) any {
	if len(m.path) == 0 {
		return m.value
	}
	parent := doc
	for _, key := range m.path[:len(m.path)-1] {
		switch k := key.(type) {
		case string:
			parent = parent.(map[string]any)[k]
		case int:
			parent = parent.([]any)[k]
		}
	}
	switch key := m.path[len(m.path)-1].(type) {
	case string:
		if m.remove {
			delete(parent.(map[string]any), key)
		} else {
			parent.(map[string]any)[key] = m.value
		}
	case int:
		parent.([]any)[key] = m.value
	}
	return doc
}

// Restore APIs deliberately add semantic invariants beyond JSON Schema. Roots
// without a public restore API are covered by typed structural DecodeWire.
func restorePositive(name string, raw []byte) error {
	switch name {
	case "comparison-policy", "search", "candidate":
		return restoreSearchPositive(name, raw)
	case "calibration":
		v, e := evaly.DecodeWire[evaly.CalibrationReport](raw)
		if e != nil {
			return e
		}
		return evaly.ValidateCalibrationReport(v)
	case "envelope":
		v, e := evaly.DecodeWire[evaly.Envelope](raw)
		if e != nil {
			return e
		}
		return evaly.ValidateEnvelope(v)
	case "dataset":
		v, e := evaly.DecodeWire[evaly.DatasetRecord](raw)
		if e != nil {
			return e
		}
		_, e = evaly.RestoreDataset(v, fixtures.InputCodec(), fixtures.ReferenceCodec())
		return e
	case "experiment":
		v, e := evaly.DecodeWire[evaly.ExperimentRecord](raw)
		if e != nil {
			return e
		}
		_, e = evaly.RestoreExperiment(v)
		return e
	case "evidence":
		v, e := evaly.DecodeWire[evaly.EvidenceRecord](raw)
		if e != nil {
			return e
		}
		return evaly.ValidateEvidence(v)
	case "view":
		v, e := evaly.DecodeWire[evaly.SavedViewRecord](raw)
		if e != nil {
			return e
		}
		_, e = evaly.RestoreSavedView(v, fixtures.InputCodec(), fixtures.OutputCodec(), fixtures.ReferenceCodec())
		return e
	case "assessment":
		v, e := evaly.DecodeWire[evaly.Assessment](raw)
		if e != nil {
			return e
		}
		_, e = evaly.RestoreAssessment(v)
		return e
	case "scenario":
		v, e := evaly.DecodeWire[evaly.ScenarioRecord](raw)
		if e != nil {
			return e
		}
		codec := evaly.JSONCodec[int]{ID: "integer", Version: "1"}
		_, e = evaly.RestoreScenario(v, codec, codec)
		return e
	default:
		return nil
	}
}
func TestSharedStructuralCorpus(t *testing.T) {
	// Arrange: genuine emitted values exercise every registered root and nested service record.
	values := generatedWireValues(t)
	if len(values) != 16 {
		t.Fatalf("expected all 16 registered wire kinds, got %d", len(values))
	}
	for name, value := range values {
		t.Run(name, func(t *testing.T) {
			checkStructuralCorpus(t, name, value)
		})
	}
}
func TestSchemaGenerationDoesNotDrift(t *testing.T) {
	// Arrange: generation runs into a disposable directory, never the checked-in schemas.
	directory := t.TempDir()
	cmd := exec.Command("go", "run", "../internal/schemagen", directory)
	// Act.
	if output, e := cmd.CombinedOutput(); e != nil {
		t.Fatalf("generate: %v\n%s", e, output)
	}
	generated, e := filepath.Glob(filepath.Join(directory, "*.json"))
	if e != nil {
		t.Fatal(e)
	}
	checked, e := filepath.Glob("../schemas/*.json")
	if e != nil {
		t.Fatal(e)
	}
	// Assert: both the filenames and bytes are reproduced; stale versions also fail.
	names := func(paths []string) []string {
		v := make([]string, 0, len(paths))
		for _, p := range paths {
			v = append(v, filepath.Base(p))
		}
		sort.Strings(v)
		return v
	}
	if !reflect.DeepEqual(names(generated), names(checked)) {
		t.Fatalf("schema inventory drift: generated %v; checked %v", names(generated), names(checked))
	}
	for _, p := range generated {
		actual, e := os.ReadFile(p)
		if e != nil {
			t.Fatal(e)
		}
		expected, e := os.ReadFile(filepath.Join("../schemas", filepath.Base(p)))
		if e != nil {
			t.Fatal(e)
		}
		if !bytes.Equal(actual, expected) {
			t.Errorf("schema byte drift: %s", filepath.Base(p))
		}
	}
}

func TestSemanticChecksumRemainsSeparateFromStructuralContract(t *testing.T) {
	// Arrange: retain a well-formed checksum token but change its semantic value.
	values := generatedWireValues(t)
	for _, name := range []string{"envelope", "evidence"} {
		t.Run(name, func(t *testing.T) {
			raw, e := json.Marshal(values[name])
			if e != nil {
				t.Fatal(e)
			}
			doc := decodeDocument(t, raw).(map[string]any)
			field := "checksum"
			if name == "evidence" {
				field = "revision"
			}
			doc[field] = strings.Repeat("f", 64)
			corrupted, e := json.Marshal(doc)
			if e != nil {
				t.Fatal(e)
			}
			// Act / Assert: the independent shape contract and DecodeWire agree;
			// semantic restoration then rejects the changed checksum.
			if e = compile(t, name).Validate(decodeDocument(t, corrupted)); e != nil {
				t.Fatalf("structural schema: %v", e)
			}
			if e = runtimeDecode(name, corrupted); e != nil {
				t.Fatalf("structural runtime: %v", e)
			}
			if e = restorePositive(name, corrupted); e == nil {
				t.Fatal("semantic restore accepted changed checksum")
			}
		})
	}
}

func schemaBoundMutations(schema map[string]any, path []any) []mutation {
	var out []mutation
	if enum, ok := schema["enum"].([]any); ok && len(enum) > 0 {
		out = append(out, mutation{"enum", path, "unsupported-corpus-value", false})
	}
	if _, ok := schema["const"]; ok {
		out = append(out, mutation{"version", path, json.Number("999"), false})
	}
	if length, ok := schema["maxLength"].(json.Number); ok {
		n, _ := length.Int64()
		if n >= 0 && n < 10000 {
			out = append(out, mutation{"max_length", path, strings.Repeat("x", int(n)+1), false})
		}
	}
	for _, bound := range []string{"minimum", "maximum"} {
		if n, ok := schema[bound].(json.Number); ok {
			delta := int64(1)
			if bound == "minimum" {
				delta = -1
			}
			out = append(out, mutation{bound, path, exactBoundaryNeighbour(n, delta), false})
		}
	}
	return out
}

func objectMutations(v map[string]any, schema map[string]any, path []any) []mutation {
	var out []mutation

	props, owned := schema["properties"].(map[string]any)
	if !owned {
		return out
	} // Domain raw JSON and host extension maps are opaque.
	out = append(out, mutation{"unknown", appendPath(path, "corpus_unknown"), true, false})
	if required, ok := schema["required"].([]any); ok {
		for _, key := range required {
			out = append(out, mutation{"missing", appendPath(path, key.(string)), nil, true})
		}
	}
	keys := make([]string, 0, len(v))
	for key := range v {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		field, ok := props[key].(map[string]any)
		if !ok {
			continue
		}
		p := appendPath(path, key)
		out = append(out, mutation{"null", p, nil, false})
		out = append(out, schemaMutations(v[key], field, p)...)
	}
	return out
}

func restoreSearchPositive(name string, raw []byte) error {
	switch name {
	case "comparison-policy":
		v, e := evaly.DecodeWire[evaly.ComparisonPolicy](raw)
		if e != nil {
			return e
		}
		_, e = v.Resolve()
		return e

	case "search":
		v, e := evaly.DecodeWire[optimizer.Result](raw)
		if e != nil {
			return e
		}
		_, e = optimizer.RestoreResult(v, evaly.JSONCodec[int]{ID: "integer", Version: "1"})
		return e
	case "candidate":
		v, e := evaly.DecodeWire[optimizer.CandidateRecord](raw)
		if e != nil {
			return e
		}
		_, e = optimizer.RestoreCandidate(v, evaly.JSONCodec[int]{ID: "integer", Version: "1"})
		return e
	default:
		return nil
	}
}

func checkStructuralCorpus(t *testing.T, name string, value any) {
	t.Helper()
	raw, e := json.Marshal(value)
	if e != nil {
		t.Fatal(e)
	}
	engine := compile(t, name)
	schemaBytes, e := os.ReadFile(wireSchemaPath(name))
	if e != nil {
		t.Fatal(e)
	}
	schema := decodeDocument(t, schemaBytes).(map[string]any)
	mutations := schemaMutations(decodeDocument(t, raw), schema, nil)
	coverage := map[string]int{}
	check := func(label string, b []byte) {
		t.Helper()
		doc := decodeDocument(t, b)
		schemaErr := engine.Validate(doc)
		runtimeErr := runtimeDecode(name, b)
		if (schemaErr == nil) != (runtimeErr == nil) {
			t.Fatalf("%s parity mismatch\nschema: %v\nruntime: %v\n%s", label, schemaErr, runtimeErr, b)
		}
	}
	// Act / Assert: positives and targeted absent/null/zero/unknown/enum/version/limit negatives agree.
	check("positive", raw)
	if e := restorePositive(name, raw); e != nil {
		t.Fatalf("semantic restore positive: %v", e)
	}
	for i, m := range mutations {
		coverage[m.label]++
		doc := applyMutation(decodeDocument(t, raw), m)
		b, e := json.Marshal(doc)
		if e != nil {
			t.Fatal(e)
		}
		check(fmt.Sprintf("%d/%s/%v", i, m.label, m.path), b)
	}
	for _, label := range []string{"missing", "null", "zero", "unknown"} {
		if coverage[label] == 0 {
			t.Fatalf("missing %s corpus coverage", label)
		}
	}
	// JSON Schema validates parsed instances: duplicate-key ambiguity is a byte-level
	// decoder invariant. The independent engine cannot recover discarded duplicates.
	duplicate := duplicateRootKey(t, raw)
	if e := runtimeDecode(name, duplicate); e == nil {
		t.Fatal("runtime accepted duplicate root key")
	}
	t.Logf("%d structural mutations; coverage %v", len(mutations), coverage)
}

func duplicateRootKey(t *testing.T, raw []byte) []byte {
	t.Helper()
	doc := decodeDocument(t, raw).(map[string]any)
	keys := make([]string, 0, len(doc))
	for k := range doc {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	key, _ := json.Marshal(keys[0])
	val, _ := json.Marshal(doc[keys[0]])
	duplicate := append([]byte{'{'}, key...)
	duplicate = append(duplicate, ':')
	duplicate = append(duplicate, val...)
	duplicate = append(duplicate, ',')
	duplicate = append(duplicate, raw[1:]...)
	return duplicate
}

// exactBoundaryNeighbour moves a trusted generated decimal boundary by exactly
// one unit, without passing through binary floating-point or silently rounding.
func exactBoundaryNeighbour(n json.Number, delta int64) json.Number {
	boundary, ok := new(big.Rat).SetString(n.String())
	if !ok {
		panic("invalid generated numeric boundary: " + n.String())
	}
	shifted := new(big.Rat).Add(boundary, new(big.Rat).SetInt64(delta))
	if shifted.IsInt() {
		return json.Number(shifted.Num().String())
	}
	literal := strings.ToLower(n.String())
	mantissa, exponentText, hasExponent := strings.Cut(literal, "e")
	exponent := 0
	if hasExponent {
		var err error
		exponent, err = strconv.Atoi(exponentText)
		if err != nil {
			panic("invalid generated decimal exponent")
		}
	}
	_, fraction, hasFraction := strings.Cut(mantissa, ".")
	precision := -exponent
	if hasFraction {
		precision += len(fraction)
	}
	if precision < 0 {
		panic("nonintegral generated boundary has invalid precision")
	}
	return json.Number(shifted.FloatString(precision))
}
