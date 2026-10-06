package evaly_test

import (
	"errors"
	"testing"

	"github.com/skosovsky/evaly"
)

func calibrationFixture() (evaly.GraderRevision, []evaly.CalibrationLabel, []evaly.CalibrationRecord) {
	rev := evaly.GraderRevision{ID: "binary", Implementation: "1", Rubric: "1"}
	labels := []evaly.CalibrationLabel{
		{CaseRevision: "tp", Pass: true, Groups: []string{"shared", "positive"}},
		{CaseRevision: "fn", Pass: true, Groups: []string{"positive"}},
		{CaseRevision: "tn", Pass: false, Groups: []string{"negative", "shared"}},
		{CaseRevision: "fp", Pass: false, Groups: []string{"negative"}},
		{CaseRevision: "error", Pass: true},
		{CaseRevision: "abstain", Pass: false},
		{CaseRevision: "missing", Pass: true, Groups: []string{"positive"}},
	}
	var records []evaly.CalibrationRecord
	for _, item := range []struct {
		name string
		pass bool
	}{{"tp", true}, {"fn", false}, {"tn", false}, {"fp", true}, {"unlabeled", true}} {
		records = append(
			records,
			evaly.CalibrationRecord{
				CaseRevision: item.name,
				Grade: evaly.Grade{
					Revision:   rev,
					Status:     evaly.Scored,
					Assertions: []evaly.Assertion{{Name: "binary", Pass: item.pass}},
				},
			},
		)
	}
	records = append(
		records,
		evaly.CalibrationRecord{CaseRevision: "error", Grade: evaly.Grade{Revision: rev, Status: evaly.GraderError}},
		evaly.CalibrationRecord{
			CaseRevision: "abstain",
			Grade:        evaly.Grade{Revision: rev, Status: evaly.InsufficientEvidence},
		},
	)
	return rev, labels, records
}

func TestCalibrationConfusionMissingAndGroups(t *testing.T) {
	// Arrange.
	rev, labels, records := calibrationFixture()
	// Act.
	r, err := evaly.Calibrate(rev, labels, records)
	// Assert.
	if err != nil || evaly.ValidateCalibrationReport(r) != nil {
		t.Fatal(err, r)
	}
	want := evaly.CalibrationCounts{
		Eligible:       8,
		Labeled:        7,
		Reviewed:       4,
		TP:             1,
		TN:             1,
		FP:             1,
		FN:             1,
		Unreviewed:     4,
		MissingLabels:  1,
		MissingRecords: 1,
		Errors:         1,
		Abstentions:    1,
	}
	if r.Counts != want || r.Rates.Coverage.Numerator != 4 || r.Rates.Coverage.Denominator != 8 ||
		*r.Rates.Coverage.Value != .5 ||
		*r.Rates.LabelCoverage.Value != .875 {
		t.Fatal(r)
	}
	for _, rate := range []evaly.CalibrationRate{r.Rates.Accuracy, r.Rates.Precision, r.Rates.Recall, r.Rates.Specificity, r.Rates.FalsePositiveRate, r.Rates.FalseNegativeRate} {
		if rate.Value == nil || *rate.Value != .5 {
			t.Fatal(rate)
		}
	}
	if len(r.Groups) != 3 || r.Groups[0].Name != "negative" || r.Groups[1].Name != "positive" ||
		r.Groups[2].Name != "shared" ||
		r.Groups[1].Counts.MissingRecords != 1 ||
		r.Groups[1].Rates.Coverage.Denominator != 3 {
		t.Fatal(r.Groups)
	}
	// Input order is not provenance: the same aggregates have the same identity.
	labels[0], labels[1] = labels[1], labels[0]
	records[0], records[1] = records[1], records[0]
	permuted, err := evaly.Calibrate(rev, labels, records)
	if err != nil || permuted.Revision != r.Revision {
		t.Fatal(err, permuted.Revision, r.Revision)
	}
}

func TestCalibrationUnavailableRates(t *testing.T) {
	rev := evaly.GraderRevision{ID: "binary", Implementation: "1", Rubric: "1"}
	for _, tt := range []struct {
		name    string
		labels  []evaly.CalibrationLabel
		records []evaly.CalibrationRecord
	}{
		{name: "empty"},
		{name: "missing_record", labels: []evaly.CalibrationLabel{{CaseRevision: "missing", Pass: true}}},
		{name: "missing_label", records: []evaly.CalibrationRecord{{CaseRevision: "unknown", Grade: evaly.Grade{Revision: rev, Status: evaly.GraderError}}}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange / Act.
			r, err := evaly.Calibrate(rev, tt.labels, tt.records)
			// Assert.
			if err != nil || evaly.ValidateCalibrationReport(r) != nil || r.Rates.Accuracy.Value != nil ||
				r.Rates.Precision.Value != nil ||
				r.Rates.Recall.Value != nil ||
				r.Rates.Specificity.Value != nil ||
				r.Rates.FalsePositiveRate.Value != nil ||
				r.Rates.FalseNegativeRate.Value != nil {
				t.Fatal(err, r)
			}
			if len(tt.labels) == 0 && len(tt.records) == 0 &&
				(r.Rates.Coverage.Value != nil || r.Rates.LabelCoverage.Value != nil) {
				t.Fatal(r)
			}
		})
	}
}

func TestCalibrationRejectsMalformedUnlabeledAndDuplicates(t *testing.T) {
	rev, labels, records := calibrationFixture()
	for _, tt := range []struct {
		name   string
		mutate func(*[]evaly.CalibrationLabel, *[]evaly.CalibrationRecord)
		want   error
	}{
		{"duplicate_label", func(l *[]evaly.CalibrationLabel, _ *[]evaly.CalibrationRecord) { *l = append(*l, (*l)[0]) }, evaly.ErrConflict},
		{"duplicate_record", func(_ *[]evaly.CalibrationLabel, r *[]evaly.CalibrationRecord) { *r = append(*r, (*r)[0]) }, evaly.ErrConflict},
		{"empty_case", func(l *[]evaly.CalibrationLabel, _ *[]evaly.CalibrationRecord) { (*l)[0].CaseRevision = "" }, evaly.ErrInvalid},
		{"duplicate_group", func(l *[]evaly.CalibrationLabel, _ *[]evaly.CalibrationRecord) { (*l)[0].Groups = []string{"a", "a"} }, evaly.ErrConflict},
		{"empty_group", func(l *[]evaly.CalibrationLabel, _ *[]evaly.CalibrationRecord) { (*l)[0].Groups = []string{""} }, evaly.ErrInvalid},
		{"invalid_unlabeled", func(_ *[]evaly.CalibrationLabel, r *[]evaly.CalibrationRecord) { (*r)[4].Grade.Usage.Units = -1 }, evaly.ErrInvalid},
		{"wrong_revision", func(_ *[]evaly.CalibrationLabel, r *[]evaly.CalibrationRecord) {
			(*r)[0].Grade.Revision.Rubric = "different"
		}, evaly.ErrConflict},
	} {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange.
			l := append([]evaly.CalibrationLabel(nil), labels...)
			r := append([]evaly.CalibrationRecord(nil), records...)
			tt.mutate(&l, &r)
			// Act.
			_, err := evaly.Calibrate(rev, l, r)
			// Assert.
			if !errors.Is(err, tt.want) {
				t.Fatal(err)
			}
		})
	}
}

func TestCalibrationReportRejectsMutation(t *testing.T) {
	// Arrange.
	rev, labels, records := calibrationFixture()
	r, err := evaly.Calibrate(rev, labels, records)
	if err != nil {
		t.Fatal(err)
	}
	// Act.
	r.Counts.FP++
	// Assert.
	if evaly.ValidateCalibrationReport(r) == nil {
		t.Fatal("accepted mutated counts")
	}
}

func TestCalibrationNumericOnlyDoesNotInventBinaryPrediction(t *testing.T) {
	// Arrange.
	rev := evaly.GraderRevision{ID: "numeric", Implementation: "1", Rubric: "1"}
	labels := []evaly.CalibrationLabel{{CaseRevision: "numeric", Pass: true}}
	records := []evaly.CalibrationRecord{{CaseRevision: "numeric", Grade: evaly.Grade{
		Revision: rev,
		Status:   evaly.Scored,
		Metrics: []evaly.Metric{
			{
				Name:          "quality",
				Unit:          "points",
				ScaleRevision: "1",
				Value:         .8,
				Minimum:       0,
				Maximum:       1,
				Direction:     "higher",
			},
		},
	}}}
	// Act.
	r, err := evaly.Calibrate(rev, labels, records)
	// Assert.
	if err != nil || evaly.ValidateCalibrationReport(r) != nil || r.Counts.Reviewed != 0 ||
		r.Counts.Abstentions != 1 || r.Counts.Unreviewed != 1 || r.Rates.Accuracy.Value != nil ||
		r.Rates.Coverage.Value == nil || *r.Rates.Coverage.Value != 0 {
		t.Fatal(err, r)
	}
}
