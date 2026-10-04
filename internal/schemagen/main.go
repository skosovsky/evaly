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
		document := wirecontract.Schema(reflect.TypeOf(value))
		document["$schema"] = "https://json-schema.org/draft/2020-12/schema"
		version := 1
		switch name {
		case "experiment", "scenario", "assessment", "observation-result", "search", "http-request", "http-response":
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
