package evaly

import "math"

// ObjectiveIdentity records the meaning and aggregation of native-scale values.
type ObjectiveIdentity struct {
	AssertionPolicy     string  `json:"assertion_policy,omitempty"`
	SourceGrader        string  `json:"source_grader,omitempty"`
	SourceMetric        string  `json:"source_metric,omitempty"`
	ID                  string  `json:"id"`
	Revision            string  `json:"revision"`
	Unit                string  `json:"unit"`
	ScaleRevision       string  `json:"scale_revision"`
	Minimum             float64 `json:"minimum"`
	Maximum             float64 `json:"maximum"`
	Direction           string  `json:"direction"`
	EligibilityRevision string  `json:"eligibility_revision"`
	MissingnessRevision string  `json:"missingness_revision"`
	AggregationRevision string  `json:"aggregation_revision"`
}
type Eligibility struct {
	Eligible bool
	Reason   string
}
type Measurement struct {
	Present bool
	Value   float64
	Reason  string
}

// Objective sees only persisted permitted results, not domain input or target capabilities.
type Objective interface {
	Identity() ObjectiveIdentity
	Eligible(CaseIdentity) (Eligibility, error)
	Measure(TrialRecord) (Measurement, error)
}
type ObjectiveFuncs struct {
	Descriptor ObjectiveIdentity
	Select     func(CaseIdentity) (Eligibility, error)
	Evaluate   func(TrialRecord) (Measurement, error)
}

func (o ObjectiveFuncs) Identity() ObjectiveIdentity { return o.Descriptor }
func (o ObjectiveFuncs) Validate() error {
	if o.Select == nil || o.Evaluate == nil {
		return ErrInvalid
	}
	return ValidateObjectiveIdentity(o.Descriptor)
}
func (o ObjectiveFuncs) Eligible(c CaseIdentity) (Eligibility, error) {
	if o.Select == nil {
		return Eligibility{}, ErrInvalid
	}
	return o.Select(c)
}
func (o ObjectiveFuncs) Measure(t TrialRecord) (Measurement, error) {
	if o.Evaluate == nil {
		return Measurement{}, ErrInvalid
	}
	return o.Evaluate(t)
}
func ValidateObjectiveIdentity(i ObjectiveIdentity) error {
	if i.ID == "" || i.Revision == "" || i.Unit == "" || i.ScaleRevision == "" || i.EligibilityRevision == "" ||
		i.MissingnessRevision == "" ||
		i.AggregationRevision != "repeat-mean-case-mean-v1" ||
		math.IsNaN(i.Minimum) ||
		math.IsInf(i.Minimum, 0) ||
		math.IsNaN(i.Maximum) ||
		math.IsInf(i.Maximum, 0) ||
		i.Minimum >= i.Maximum ||
		math.IsInf(i.Maximum-i.Minimum, 0) ||
		(i.Direction != "higher" && i.Direction != "lower") {
		return ErrInvalid
	}
	return nil
}
func ValidateMeasurement(i ObjectiveIdentity, m Measurement) error {
	if !m.Present {
		if m.Reason == "" || m.Value != 0 {
			return ErrInvalid
		}
		return nil
	}
	if m.Reason != "" || math.IsNaN(m.Value) || math.IsInf(m.Value, 0) || m.Value < i.Minimum || m.Value > i.Maximum {
		return ErrInvalid
	}
	return nil
}

// AssertionObjective measures each repeat as pass=1 or fail=0.
type AssertionObjective struct{ ID, Revision, Policy string }

func (o AssertionObjective) Identity() ObjectiveIdentity {
	return ObjectiveIdentity{
		ID:                  o.ID,
		Revision:            o.Revision,
		AssertionPolicy:     o.Policy,
		Unit:                "pass_fraction",
		ScaleRevision:       "binary-v1",
		Minimum:             0,
		Maximum:             1,
		Direction:           "higher",
		EligibilityRevision: "all-declared-v1",
		MissingnessRevision: "all-repeats-required-v1",
		AggregationRevision: "repeat-mean-case-mean-v1",
	}
}
func (o AssertionObjective) Validate() error {
	if o.Policy != "all" && o.Policy != "any" {
		return ErrInvalid
	}
	return ValidateObjectiveIdentity(o.Identity())
}
func (o AssertionObjective) Eligible(CaseIdentity) (Eligibility, error) {
	return Eligibility{Eligible: true}, nil
}
func (o AssertionObjective) Measure(t TrialRecord) (Measurement, error) {
	pass, scored := AssertionOutcome(t.Grades, o.Policy)
	if !scored {
		return Measurement{Reason: "assertions_unavailable"}, nil
	}
	v := 0.0
	if pass {
		v = 1
	}
	return Measurement{Present: true, Value: v}, nil
}

// NumericObjective selects one metric without normalization or cross-unit averaging.
type NumericObjective struct {
	Descriptor           ObjectiveIdentity
	GraderID, MetricName string
}

func (o NumericObjective) Identity() ObjectiveIdentity {
	i := o.Descriptor
	i.SourceGrader = o.GraderID
	i.SourceMetric = o.MetricName
	return i
}
func (o NumericObjective) Validate() error {
	if o.GraderID == "" || o.MetricName == "" {
		return ErrInvalid
	}
	return ValidateObjectiveIdentity(o.Descriptor)
}
func (o NumericObjective) Eligible(CaseIdentity) (Eligibility, error) {
	return Eligibility{Eligible: true}, nil
}
func (o NumericObjective) Measure(t TrialRecord) (Measurement, error) {
	for _, g := range t.Grades {
		if g.Revision.ID != o.GraderID {
			continue
		}
		if g.Status != Scored {
			return Measurement{Reason: "metric_grader_unavailable"}, nil
		}
		for _, m := range g.Metrics {
			if m.Name != o.MetricName {
				continue
			}
			i := o.Descriptor
			if m.Unit != i.Unit || m.ScaleRevision != i.ScaleRevision || m.Minimum != i.Minimum ||
				m.Maximum != i.Maximum ||
				m.Direction != i.Direction {
				return Measurement{}, ErrConflict
			}
			v := Measurement{Present: true, Value: m.Value}
			return v, ValidateMeasurement(i, v)
		}
		return Measurement{Reason: "metric_missing"}, nil
	}
	return Measurement{Reason: "metric_grader_missing"}, nil
}
