package evaly

import "fmt"

const ComparisonPolicyVersion = 1

// ComparisonPolicy describes only the built-in serializable objectives.
// Host-defined callbacks remain available through Compare's Objective port.
type ComparisonPolicy struct {
	Version       int               `json:"version"`
	ObjectiveKind string            `json:"objective_kind"`
	Objective     ObjectiveIdentity `json:"objective"`
	Gate          GatePolicy        `json:"gate"`
}

// Resolve validates the complete descriptor without loading or running a target.
func (p ComparisonPolicy) Resolve() (Objective, error) {
	if p.Version != ComparisonPolicyVersion {
		return nil, ErrUnsupported
	}
	if ValidateObjectiveIdentity(p.Objective) != nil {
		return nil, ErrInvalid
	}
	if p.Objective.EligibilityRevision != "all-declared-v1" ||
		p.Objective.MissingnessRevision != "all-repeats-required-v1" {
		return nil, fmt.Errorf("%w: callback measurement", ErrUnsupported)
	}
	var objective Objective
	switch p.ObjectiveKind {
	case "assertion":
		o := AssertionObjective{ID: p.Objective.ID, Revision: p.Objective.Revision, Policy: p.Objective.AssertionPolicy}
		if o.Validate() != nil || o.Identity() != p.Objective {
			return nil, fmt.Errorf("%w: assertion descriptor", ErrInvalid)
		}
		objective = o
	case "numeric":

		if p.Objective.AssertionPolicy != "" {
			return nil, ErrInvalid
		}
		o := NumericObjective{
			Descriptor: p.Objective,
			GraderID:   p.Objective.SourceGrader,
			MetricName: p.Objective.SourceMetric,
		}
		if o.Validate() != nil {
			return nil, ErrInvalid
		}
		objective = o
	default:
		return nil, fmt.Errorf("%w: objective kind", ErrUnsupported)
	}
	if validateGatePolicy(p.Objective, p.Gate) != nil {
		return nil, ErrInvalid
	}
	return objective, nil
}
