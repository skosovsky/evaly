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
			"ComparisonPolicy":  true,
			"SavedViewRecord":   true,
			"ScenarioRecord":    true,
			"Assessment":        true,
			"CalibrationReport": true,
		},
		"github.com/skosovsky/evaly/observation": {
			"Record":     true,
			schemaResult: true,
		},
		optimizerPackagePath:                           {schemaResult: true, "CandidateRecord": true},
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
		exponent, e := strconv.ParseInt(literal[offset+1:], decimalRadix, 64)
		if e != nil || exponent > int64(len(literal)+maximumDecimalExponent) ||
			exponent < -int64(len(literal)+maximumDecimalExponent) {
			return boundedDecimal(literal, offset, exponent)
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
		return validateOptions(options, v)
	}
	if err := validateLiterals(s, v); err != nil {
		return nil, err
	}
	if err := validateTypes(s, v); err != nil {
		return nil, err
	}
	var err error
	v, err = normalizeNumber(s, v)
	if err != nil {
		return nil, err
	}
	if err = validateString(s, v); err != nil {
		return nil, err
	}
	if err = validateArray(s, v); err != nil {
		return nil, err
	}
	if err = validateObject(s, v); err != nil {
		return nil, err
	}
	if err = validateConditions(s, v); err != nil {
		return nil, err
	}
	return v, nil
}

func validateOptions(options []any, v any) (any, error) {
	for _, option := range options {
		branch, ok := option.(schema)
		if !ok {
			return nil, errShape
		}
		if normalized, err := validate(branch, v); err == nil {
			return normalized, nil
		}
	}
	return nil, errShape
}

func validateLiterals(s schema, v any) error {
	if c, ok := s["const"]; ok && !same(c, v) {
		if s["x-evaly-version"] == true && integralToken(v) {
			return ErrVersion
		}
		return errShape
	}
	if enums, ok := s["enum"].([]string); ok {
		for _, c := range enums {
			if same(c, v) {
				return nil
			}
		}
		return errShape
	}
	return nil
}

func integralToken(v any) bool {
	if _, ok := v.(json.Number); !ok {
		return false
	}
	r := rational(v)
	return r != nil && r.IsInt()
}

func validateTypes(s schema, v any) error {
	var names []string
	switch x := s[schemaType].(type) {
	case string:
		names = []string{x}
	case []string:
		names = x
	}
	if len(names) == 0 {
		return nil
	}
	for _, name := range names {
		if matchesType(name, v) {
			return nil
		}
	}
	return errShape
}

func matchesType(name string, v any) bool {
	switch name {
	case schemaNull:
		return v == nil
	case schemaObject:
		_, ok := v.(map[string]any)
		return ok
	case schemaArray:
		_, ok := v.([]any)
		return ok
	case schemaString:
		_, ok := v.(string)
		return ok
	case "boolean":
		_, ok := v.(bool)
		return ok
	case schemaNumber:
		number, ok := v.(json.Number)
		if !ok {
			return false
		}
		_, err := strconv.ParseFloat(string(number), 64)
		return err == nil
	case schemaInteger:
		return integralToken(v)
	default:
		return false
	}
}

func normalizeNumber(s schema, v any) (any, error) {
	r := rational(v)
	if r == nil {
		return v, nil
	}
	for _, name := range []string{schemaMinimum, schemaMaximum} {
		if bound, ok := s[name]; ok {
			limit := rational(bound)
			if limit == nil {
				return nil, errShape
			}
			comparison := r.Cmp(limit)
			if name == schemaMinimum && comparison < 0 || name == schemaMaximum && comparison > 0 {
				return nil, errShape
			}
		}
	}
	if s[schemaType] == schemaInteger {
		v = json.Number(r.Num().String())
	}
	return v, nil
}

func validateString(s schema, v any) error {
	str, ok := v.(string)
	if !ok {
		return nil
	}
	if pattern, ok := s["pattern"].(string); ok {
		compiled, err := regexp.Compile(pattern)
		if err != nil || !compiled.MatchString(str) {
			return errShape
		}
	}
	if s["format"] == "date-time" {
		if _, err := time.Parse(time.RFC3339Nano, str); err != nil {
			return errShape
		}
	}
	return nil
}

func validateArray(s schema, v any) error {
	array, ok := v.([]any)
	if !ok {
		return nil
	}
	if maximum, okLocal := s["maxItems"].(int); okLocal && len(array) > maximum {
		return errShape
	}
	item, ok := s["items"].(schema)
	if !ok {
		return nil
	}
	for i, value := range array {
		normalized, err := validate(item, value)
		if err != nil {
			return err
		}
		array[i] = normalized
	}
	return nil
}

func validateObject(s schema, v any) error {
	object, ok := v.(map[string]any)
	if !ok {
		return nil
	}
	required, _ := s[schemaRequired].([]string)
	for _, name := range required {
		if _, exists := object[name]; !exists {
			return errShape
		}
	}
	properties, _ := s["properties"].(map[string]any)
	if typed, typedOK := s["properties"].(schema); typedOK {
		properties = map[string]any(typed)
	}
	for key, value := range object {
		field, err := objectField(s, properties, key)
		if err != nil {
			return err
		}
		if field == nil {
			continue
		}
		normalized, err := validate(field, value)
		if err != nil {
			return err
		}
		object[key] = normalized
	}
	return nil
}

func objectField(s schema, properties map[string]any, key string) (schema, error) {
	if field, ok := properties[key]; ok {
		typed, typedOK := field.(schema)
		if !typedOK {
			return nil, errShape
		}
		return typed, nil
	}
	switch additional := s["additionalProperties"].(type) {
	case bool:
		if !additional {
			return nil, errShape
		}
	case schema:
		return additional, nil
	}
	return schema{}, nil
}

func validateConditions(s schema, v any) error {
	if constraints, ok := s["allOf"].([]any); ok {
		for _, constraint := range constraints {
			typed, typedOK := constraint.(schema)
			if !typedOK {
				return errShape
			}
			if _, err := validate(typed, v); err != nil {
				return err
			}
		}
	}
	if condition, ok := s["if"].(schema); ok {
		if _, err := validate(condition, v); err == nil {
			if err = validateConsequence(s, v); err != nil {
				return err
			}
		}
	}
	if negation, ok := s["not"].(schema); ok {
		if _, err := validate(negation, v); err == nil {
			return errShape
		}
	}
	return nil
}

func validateConsequence(s schema, v any) error {
	if consequence, ok := s["then"].(schema); ok {
		_, err := validate(consequence, v)
		return err
	}
	return nil
}

func boundedDecimal(literal string, offset int, exponent int64) *big.Rat {
	coefficient, ok := new(big.Rat).SetString(literal[:offset])
	if ok && coefficient.Sign() == 0 {
		return new(big.Rat)
	}
	// A nonzero value below representable precision still has a sign:
	// preserve it for minimum/maximum checks without an enormous denominator.
	if exponent < 0 || strings.HasPrefix(literal[offset+1:], "-") {
		parsed, parseError := strconv.ParseFloat(literal, 64)
		if parseError == nil && parsed == 0 {
			tiny := new(
				big.Rat,
			).SetFrac(big.NewInt(1), new(big.Int).Exp(big.NewInt(decimalRadix), big.NewInt(underflowDecimalExponent), nil))
			if strings.HasPrefix(literal, "-") {
				tiny.Neg(tiny)
			}
			return tiny
		}
	}
	return nil
}
