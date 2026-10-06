package evaly

import "sort"

// CalibrationLabel contains one host supplied human label and optional opaque slices.
type CalibrationLabel struct {
	CaseRevision string   `json:"CaseRevision"`
	Pass         bool     `json:"Pass"`
	Groups       []string `json:"Groups"`
}
type CalibrationRecord struct {
	CaseRevision string `json:"CaseRevision"`
	Grade        Grade  `json:"Grade"`
}
type CalibrationCounts struct {
	Eligible       int `json:"eligible"`
	Labeled        int `json:"labeled"`
	Reviewed       int `json:"reviewed"`
	TP             int `json:"tp"`
	TN             int `json:"tn"`
	FP             int `json:"fp"`
	FN             int `json:"fn"`
	Unreviewed     int `json:"unreviewed"`
	MissingLabels  int `json:"missing_labels"`
	MissingRecords int `json:"missing_records"`
	Errors         int `json:"errors"`
	Abstentions    int `json:"abstentions"`
}

// CalibrationRate is unavailable (Value == nil) exactly when Denominator is zero.
type CalibrationRate struct {
	Numerator   int      `json:"numerator"`
	Denominator int      `json:"denominator"`
	Value       *float64 `json:"value"`
}
type CalibrationRates struct {
	Coverage          CalibrationRate `json:"coverage"`
	LabelCoverage     CalibrationRate `json:"label_coverage"`
	Accuracy          CalibrationRate `json:"accuracy"`
	Precision         CalibrationRate `json:"precision"`
	Recall            CalibrationRate `json:"recall"`
	Specificity       CalibrationRate `json:"specificity"`
	FalsePositiveRate CalibrationRate `json:"false_positive_rate"`
	FalseNegativeRate CalibrationRate `json:"false_negative_rate"`
}
type CalibrationGroupReport struct {
	Name   string            `json:"name"`
	Counts CalibrationCounts `json:"counts"`
	Rates  CalibrationRates  `json:"rates"`
}
type CalibrationReport struct {
	Version  int                      `json:"version"`
	Revision string                   `json:"revision"`
	Grader   GraderRevision           `json:"grader"`
	Counts   CalibrationCounts        `json:"counts"`
	Rates    CalibrationRates         `json:"rates"`
	Groups   []CalibrationGroupReport `json:"groups"`
}

// Calibrate reports binary assertion predictions; missing labels never become predictions.
func Calibrate(rev GraderRevision, labels []CalibrationLabel, records []CalibrationRecord) (CalibrationReport, error) {
	var zeroCalibrationCounts CalibrationCounts
	var zeroCalibrationRates CalibrationRates
	var zero CalibrationReport
	if err := ValidateGraderRevisions([]GraderRevision{rev}); err != nil {
		return zero, err
	}
	expected, groups, err := calibrationLabels(labels)
	if err != nil {
		return zero, err
	}
	observed := make(map[string]Grade, len(records))
	for _, record := range records {
		if record.CaseRevision == "" {
			return zero, ErrInvalid
		}
		if record.Grade.Revision != rev {
			return zero, ErrConflict
		}
		if _, ok := observed[record.CaseRevision]; ok {
			return zero, ErrConflict
		}
		if gradeErr := ValidateGrade(record.Grade); gradeErr != nil {
			return zero, gradeErr
		}
		observed[record.CaseRevision] = record.Grade
	}
	r := CalibrationReport{
		Version:  1,
		Grader:   rev,
		Groups:   make([]CalibrationGroupReport, 0, len(groups)),
		Revision: "",
		Counts:   zeroCalibrationCounts,
		Rates:    zeroCalibrationRates,
	}
	r.Counts = calibrationCounts(expected, observed)
	r.Rates = calibrationRates(r.Counts)
	for name, members := range groups {
		groupLabels := make(map[string]CalibrationLabel, len(members))
		groupRecords := make(map[string]Grade, len(members))
		for _, l := range members {
			groupLabels[l.CaseRevision] = l
			if g, ok := observed[l.CaseRevision]; ok {
				groupRecords[l.CaseRevision] = g
			}
		}
		counts := calibrationCounts(groupLabels, groupRecords)
		r.Groups = append(r.Groups, CalibrationGroupReport{Name: name, Counts: counts, Rates: calibrationRates(counts)})
	}
	sort.Slice(r.Groups, func(i, j int) bool { return r.Groups[i].Name < r.Groups[j].Name })
	b, err := canonical(r)
	if err != nil {
		return zero, err
	}
	r.Revision = digest(b)
	return r, nil
}

func calibrationCounts(labels map[string]CalibrationLabel, records map[string]Grade) CalibrationCounts {
	c := CalibrationCounts{
		Eligible:       len(labels),
		Labeled:        len(labels),
		Reviewed:       0,
		TP:             0,
		TN:             0,
		FP:             0,
		FN:             0,
		Unreviewed:     0,
		MissingLabels:  0,
		MissingRecords: 0,
		Errors:         0,
		Abstentions:    0,
	}
	for key := range labels {
		if _, ok := records[key]; !ok {
			c.MissingRecords++
		}
	}
	for key, g := range records {
		label, labeled := labels[key]
		if !labeled {
			c.Eligible++
			c.MissingLabels++
		}
		prediction, binary := AssertionOutcome([]Grade{g}, assertionAll)
		if g.Status == GraderError {
			c.Errors++
		} else if !binary {
			c.Abstentions++
		}
		if !labeled || !binary {
			continue
		}
		c.Reviewed++
		switch {
		case label.Pass && prediction:
			c.TP++
		case !label.Pass && !prediction:
			c.TN++
		case !label.Pass && prediction:
			c.FP++
		case label.Pass && !prediction:
			c.FN++
		}
	}
	c.Unreviewed = c.Eligible - c.Reviewed
	return c
}
func calibrationRate(n, d int) CalibrationRate {
	r := CalibrationRate{Numerator: n, Denominator: d, Value: nil}
	if d > 0 {
		value := float64(n) / float64(d)
		r.Value = &value
	}
	return r
}
func calibrationRates(c CalibrationCounts) CalibrationRates {
	return CalibrationRates{
		Coverage: calibrationRate(c.Reviewed, c.Eligible), LabelCoverage: calibrationRate(c.Labeled, c.Eligible),
		Accuracy: calibrationRate(c.TP+c.TN, c.Reviewed), Precision: calibrationRate(c.TP, c.TP+c.FP),
		Recall: calibrationRate(c.TP, c.TP+c.FN), Specificity: calibrationRate(c.TN, c.TN+c.FP),
		FalsePositiveRate: calibrationRate(c.FP, c.FP+c.TN), FalseNegativeRate: calibrationRate(c.FN, c.FN+c.TP),
	}
}

// ValidateCalibrationReport checks the canonical identity and aggregate invariants.
func ValidateCalibrationReport(r CalibrationReport) error {
	if r.Version != 1 {
		return ErrUnsupported
	}
	if ValidateGraderRevisions([]GraderRevision{r.Grader}) != nil || r.Revision == "" || r.Groups == nil {
		return ErrInvalid
	}
	if !validCalibrationSummary(r.Counts, r.Rates) {
		return ErrInvalid
	}
	previous := ""
	for _, group := range r.Groups {
		if group.Name == "" || group.Name <= previous || !validCalibrationSummary(group.Counts, group.Rates) ||
			group.Counts.Eligible != group.Counts.Labeled || group.Counts.MissingLabels != 0 {
			return ErrInvalid
		}
		// Overlapping slices are allowed, but each slice is a subset of the full population.
		if !calibrationSubset(group.Counts, r.Counts) {
			return ErrInvalid
		}
		previous = group.Name
	}
	revision := r.Revision
	r.Revision = ""
	b, err := canonical(r)
	if err != nil || digest(b) != revision {
		return ErrCorrupt
	}
	return nil
}
func validCalibrationSummary(c CalibrationCounts, r CalibrationRates) bool {
	for _, n := range []int{c.Eligible, c.Labeled, c.Reviewed, c.TP, c.TN, c.FP, c.FN, c.Unreviewed, c.MissingLabels, c.MissingRecords, c.Errors, c.Abstentions} {
		if n < 0 || n > c.Eligible {
			return false
		}
	}
	if c.Labeled != c.Eligible-c.MissingLabels ||
		!calibrationSum(c.Reviewed, c.TP, c.TN, c.FP, c.FN) ||
		c.Unreviewed != c.Eligible-c.Reviewed || c.Reviewed > c.Labeled || c.MissingRecords > c.Labeled ||
		c.MissingRecords > c.Labeled-c.Reviewed {
		return false
	}
	remaining := c.Eligible
	for _, n := range []int{c.Reviewed, c.MissingRecords, c.Errors, c.Abstentions} {
		if n > remaining {
			return false
		}
		remaining -= n
	}
	// All remaining records are unlabeled binary predictions.
	if remaining > c.MissingLabels {
		return false
	}
	want := calibrationRates(c)
	pairs := [][2]CalibrationRate{
		{r.Coverage, want.Coverage},
		{r.LabelCoverage, want.LabelCoverage},
		{r.Accuracy, want.Accuracy},
		{r.Precision, want.Precision},
		{r.Recall, want.Recall},
		{r.Specificity, want.Specificity},
		{r.FalsePositiveRate, want.FalsePositiveRate},
		{r.FalseNegativeRate, want.FalseNegativeRate},
	}
	for _, pair := range pairs {
		a, b := pair[0], pair[1]
		if a.Numerator != b.Numerator || a.Denominator != b.Denominator || (a.Value == nil) != (b.Value == nil) {
			return false
		}
		if a.Value != nil && *a.Value != *b.Value {
			return false
		}
	}
	return true
}
func calibrationSubset(a, b CalibrationCounts) bool {
	av := []int{
		a.Eligible,
		a.Labeled,
		a.Reviewed,
		a.TP,
		a.TN,
		a.FP,
		a.FN,
		a.Unreviewed,
		a.MissingLabels,
		a.MissingRecords,
		a.Errors,
		a.Abstentions,
	}
	bv := []int{
		b.Eligible,
		b.Labeled,
		b.Reviewed,
		b.TP,
		b.TN,
		b.FP,
		b.FN,
		b.Unreviewed,
		b.MissingLabels,
		b.MissingRecords,
		b.Errors,
		b.Abstentions,
	}
	for i, n := range av {
		if n > bv[i] {
			return false
		}
	}
	return true
}

func calibrationSum(total int, values ...int) bool {
	for _, value := range values {
		if value > total {
			return false
		}
		total -= value
	}
	return total == 0
}

func calibrationLabels(labels []CalibrationLabel) (map[string]CalibrationLabel, map[string][]CalibrationLabel, error) {
	expected := make(map[string]CalibrationLabel, len(labels))
	groups := map[string][]CalibrationLabel{}
	for _, l := range labels {
		if l.CaseRevision == "" {
			return nil, nil, ErrInvalid
		}
		if _, ok := expected[l.CaseRevision]; ok {
			return nil, nil, ErrConflict
		}
		seen := map[string]bool{}
		for _, group := range l.Groups {
			if group == "" {
				return nil, nil, ErrInvalid
			}
			if seen[group] {
				return nil, nil, ErrConflict
			}
			seen[group] = true
			groups[group] = append(groups[group], l)
		}
		expected[l.CaseRevision] = l
	}
	return expected, groups, nil
}
