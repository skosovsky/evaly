package conformance

import (
	"testing"

	"github.com/skosovsky/evaly"
)

// Objective checks stable declared identity and reproducible native-scale values.
// The host supplies fresh permitted records and the expected reference result.
func Objective(
	t *testing.T,
	o evaly.Objective,
	cs evaly.CaseIdentity,
	factory func() evaly.TrialRecord,
	expected evaly.Measurement,
) {
	t.Helper()
	if err := evaly.ValidatePort(o); err != nil {
		t.Fatal(err)
	}
	identity := o.Identity()
	if err := evaly.ValidateObjectiveIdentity(identity); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		eligibility, err := o.Eligible(cs)
		if err != nil || !eligibility.Eligible || eligibility.Reason != "" {
			t.Fatal("reference eligibility", eligibility, err)
		}
		measurement, err := o.Measure(factory())
		if err != nil || measurement != expected || evaly.ValidateMeasurement(identity, measurement) != nil ||
			o.Identity() != identity {
			t.Fatal("unstable or invalid objective result", measurement, err)
		}
	}
}

// Calibration checks binary confusion counts and explicit unavailable rates.
func Calibration(
	t *testing.T,
	calculate func(evaly.GraderRevision, []evaly.CalibrationLabel, []evaly.CalibrationRecord) (evaly.CalibrationReport, error),
) {
	t.Helper()
	// Arrange: deliberately include one result in every confusion-matrix cell.
	var zeroUsage evaly.Usage
	revision := evaly.GraderRevision{
		ID:             "binary",
		Implementation: "v1",
		Rubric:         "binary-v1",
		Model:          "",
		Prompt:         "",
		Configuration:  "",
	}
	labels := []evaly.CalibrationLabel{
		{CaseRevision: "tp", Pass: true, Groups: nil},
		{CaseRevision: "tn", Pass: false, Groups: nil},
		{CaseRevision: "fp", Pass: false, Groups: nil},
		{CaseRevision: "fn", Pass: true, Groups: nil},
	}
	records := make([]evaly.CalibrationRecord, 0, len(labels))
	for _, label := range labels {
		pass := label.CaseRevision == "tp" || label.CaseRevision == "fp"
		records = append(
			records,
			evaly.CalibrationRecord{
				CaseRevision: label.CaseRevision,
				Grade: evaly.Grade{
					Revision: revision,
					Status:   evaly.Scored,
					Assertions: []evaly.Assertion{
						{Name: "binary", Pass: pass, Reason: ""},
					},
					Dispatched:   false,
					Metrics:      nil,
					Reasons:      nil,
					EvidenceRefs: nil,
					Usage:        zeroUsage,
				},
			},
		)
	}

	// Act.
	report, err := calculate(revision, labels, records)
	empty, emptyErr := calculate(revision, nil, nil)
	// Assert: no missing denominator is represented as a numeric zero or one.
	if err != nil || evaly.ValidateCalibrationReport(report) != nil || report.Counts.TP != 1 || report.Counts.TN != 1 ||
		report.Counts.FP != 1 ||
		report.Counts.FN != 1 ||
		report.Rates.Coverage.Value == nil ||
		*report.Rates.Coverage.Value != 1 {
		t.Fatal(report, err)
	}
	if emptyErr != nil || evaly.ValidateCalibrationReport(empty) != nil || empty.Rates.Coverage.Value != nil ||
		empty.Rates.Precision.Value != nil ||
		empty.Rates.Recall.Value != nil {
		t.Fatal("invented rate without denominator", empty, emptyErr)
	}
}
