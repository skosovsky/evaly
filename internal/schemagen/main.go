// Command schemagen emits JSON wire schemas from explicit envelope types.
// This is development tooling, never a domain codec or artifact store path.
package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strconv"

	"github.com/skosovsky/evaly/internal/wirecontract"

	"github.com/skosovsky/evaly"
	"github.com/skosovsky/evaly/adapters/httpjson"
	"github.com/skosovsky/evaly/observation"
	"github.com/skosovsky/evaly/optimizer"
)

func main() {
	directory := "schemas"
	if len(os.Args) == 2 {
		directory = os.Args[1]
	}
	types := map[string]reflect.Type{
		"envelope":           reflect.TypeFor[evaly.Envelope](),
		"dataset":            reflect.TypeFor[evaly.DatasetRecord](),
		"experiment":         reflect.TypeFor[evaly.ExperimentRecord](),
		"evidence":           reflect.TypeFor[evaly.EvidenceRecord](),
		"comparison":         reflect.TypeFor[evaly.Comparison](),
		"comparison-policy":  reflect.TypeFor[evaly.ComparisonPolicy](),
		"calibration":        reflect.TypeFor[evaly.CalibrationReport](),
		"view":               reflect.TypeFor[evaly.SavedViewRecord](),
		"scenario":           reflect.TypeFor[evaly.ScenarioRecord](),
		"assessment":         reflect.TypeFor[evaly.Assessment](),
		"observation":        reflect.TypeFor[observation.Record](),
		"observation-result": reflect.TypeFor[observation.Result](),
		"search":             reflect.TypeFor[optimizer.Result](),
		"candidate":          reflect.TypeFor[optimizer.CandidateRecord](),
		"http-request":       reflect.TypeFor[httpjson.Request](),
		"http-response":      reflect.TypeFor[httpjson.Response](),
	}
	for name, value := range types {
		document := wirecontract.Schema(value)
		document["$schema"] = "https://json-schema.org/draft/2020-12/schema"
		version := 1
		switch name {
		case "scenario", "comparison":
			version = 2
		case "experiment", "assessment", "observation-result", "http-request", "http-response":
			version = 3
		case "search":
			version = 5
		case "candidate":
			version = 2
		}
		document["$id"] = "urn:evaly:" + name + ":" + strconv.Itoa(version)
		b, e := json.MarshalIndent(document, "", "  ")
		if e != nil {
			panic(e)
		}
		// #nosec G703 -- the local operator explicitly selects the schema output directory.
		if e = os.WriteFile(
			filepath.Join(directory, name+"-v"+strconv.Itoa(version)+".json"),
			append(b, '\n'),
			0600,
		); e != nil {
			panic(e)
		}
	}
}
