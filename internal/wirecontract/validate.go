package wirecontract

import (
	"encoding/json"
	"errors"
	"math/big"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var errShape = errors.New("invalid wire structure")

// ErrVersion identifies a structurally integral unsupported major version.
var ErrVersion = errors.New("unsupported wire version")

// Supported restricts reflection to published service envelopes.
func Supported(t reflect.Type) bool {
	roots := map[string]map[string]bool{
		"github.com/skosovsky/evaly": {
			"Envelope":          true,
			"DatasetRecord":     true,
			"ExperimentRecord":  true,
			"EvidenceRecord":    true,
			"Comparison":        true,
			"SavedViewRecord":   true,
			"ScenarioRecord":    true,
			"Assessment":        true,
			"CalibrationReport": true,
		},
		"github.com/skosovsky/evaly/observation":       {"Record": true, "Result": true},
		"github.com/skosovsky/evaly/optimizer":         {"Result": true, "CandidateRecord": true},
		"github.com/skosovsky/evaly/adapters/httpjson": {"Request": true, "Response": true},
	}
	return roots[t.PkgPath()][t.Name()]
}

// Validate returns JSON values normalized to integer tokens for integral numbers.
func Validate(t reflect.Type, value any) (any, error) {
	if !Supported(t) {
		return nil, errShape
	}
	return validate(Schema(t), value)
}
func rational(v any) *big.Rat {
	b, e := json.Marshal(v)
	if e != nil {
		return nil
	}
	literal := string(b)
	if offset := strings.IndexAny(literal, "eE"); offset >= 0 {
		exponent, e := strconv.ParseInt(literal[offset+1:], 10, 64)
		if e != nil || exponent > int64(len(literal)+400) || exponent < -int64(len(literal)+400) {
			coefficient, ok := new(big.Rat).SetString(literal[:offset])
			if ok && coefficient.Sign() == 0 {
				return new(big.Rat)
			}
			// A nonzero value below representable precision still has a sign:
			// preserve it for minimum/maximum checks without an enormous denominator.
			if exponent < 0 || strings.HasPrefix(literal[offset+1:], "-") {
				parsed, parseError := strconv.ParseFloat(literal, 64)
				if parseError == nil && parsed == 0 {
					tiny := new(big.Rat).SetFrac(big.NewInt(1), new(big.Int).Exp(big.NewInt(10), big.NewInt(401), nil))
					if strings.HasPrefix(literal, "-") {
						tiny.Neg(tiny)
					}
					return tiny
				}
			}
			return nil
		}
	}
	r, ok := new(big.Rat).SetString(literal)
	if !ok {
		return nil
	}
	return r
}
func same(a, b any) bool {
	ar, br := rational(a), rational(b)
	if ar != nil && br != nil {
		return ar.Cmp(br) == 0
	}
	return reflect.DeepEqual(a, b)
}
func validate(s schema, v any) (any, error) {
	if options, ok := s["anyOf"].([]any); ok {
		for _, option := range options {
			if normalized, e := validate(option.(schema), v); e == nil {
				return normalized, nil
			}
		}
		return nil, errShape
	}
	if c, ok := s["const"]; ok && !same(c, v) {
		if s["x-evaly-version"] == true {
			if _, ok := v.(json.Number); ok {
				if r := rational(v); r != nil && r.IsInt() {
					return nil, ErrVersion
				}
			}
		}
		return nil, errShape
	}
	if enums, ok := s["enum"].([]string); ok {
		found := false
		for _, c := range enums {
			if same(c, v) {
				found = true
			}
		}
		if !found {
			return nil, errShape
		}
	}
	types := []string{}
	switch x := s["type"].(type) {
	case string:
		types = []string{x}
	case []string:
		types = x
	}
	if len(types) > 0 {
		matched := false
		for _, typ := range types {
			switch typ {
			case "null":
				matched = matched || v == nil
			case "object":
				_, ok := v.(map[string]any)
				matched = matched || ok
			case "array":
				_, ok := v.([]any)
				matched = matched || ok
			case "string":
				_, ok := v.(string)
				matched = matched || ok
			case "boolean":
				_, ok := v.(bool)
				matched = matched || ok
			case "number":
				number, ok := v.(json.Number)
				if ok {
					_, e := strconv.ParseFloat(string(number), 64)
					ok = e == nil
				}
				matched = matched || ok
			case "integer":
				_, ok := v.(json.Number)
				r := rational(v)
				matched = matched || (ok && r != nil && r.IsInt())
			}
		}
		if !matched {
			return nil, errShape
		}
	}
	if r := rational(v); r != nil {
		if bound, ok := s["minimum"]; ok && r.Cmp(rational(bound)) < 0 {
			return nil, errShape
		}
		if bound, ok := s["maximum"]; ok && r.Cmp(rational(bound)) > 0 {
			return nil, errShape
		}
		if s["type"] == "integer" {
			v = json.Number(r.Num().String())
		}
	}
	if str, ok := v.(string); ok {
		if pattern, ok := s["pattern"].(string); ok && !regexp.MustCompile(pattern).MatchString(str) {
			return nil, errShape
		}
		if s["format"] == "date-time" {
			if _, e := time.Parse(time.RFC3339Nano, str); e != nil {
				return nil, errShape
			}
		}
	}
	if array, ok := v.([]any); ok {
		if maximum, ok := s["maxItems"].(int); ok && len(array) > maximum {
			return nil, errShape
		}
		if item, ok := s["items"].(schema); ok {
			for i, x := range array {
				normalized, e := validate(item, x)
				if e != nil {
					return nil, e
				}
				array[i] = normalized
			}
		}
	}
	if object, ok := v.(map[string]any); ok {
		required, _ := s["required"].([]string)
		for _, name := range required {
			if _, ok := object[name]; !ok {
				return nil, errShape
			}
		}
		properties, _ := s["properties"].(map[string]any)
		if typed, ok := s["properties"].(schema); ok {
			properties = map[string]any(typed)
		}
		for key, x := range object {
			field, ok := properties[key]
			if !ok {
				switch additional := s["additionalProperties"].(type) {
				case bool:
					if !additional {
						return nil, errShape
					}
				case schema:
					field = additional
					ok = true
				}
			}
			if ok {
				normalized, e := validate(field.(schema), x)
				if e != nil {
					return nil, e
				}
				object[key] = normalized
			}
		}
	}
	if constraints, ok := s["allOf"].([]any); ok {
		for _, constraint := range constraints {
			if _, e := validate(constraint.(schema), v); e != nil {
				return nil, e
			}
		}
	}
	if condition, ok := s["if"].(schema); ok {
		if _, e := validate(condition, v); e == nil {
			if consequence, ok := s["then"].(schema); ok {
				if _, e = validate(consequence, v); e != nil {
					return nil, e
				}
			}
		}
	}
	if negation, ok := s["not"].(schema); ok {
		if _, e := validate(negation, v); e == nil {
			return nil, errShape
		}
	}
	return v, nil
}
