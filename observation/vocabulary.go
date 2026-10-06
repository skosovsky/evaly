package observation

// State is a terminal worker outcome, not a quality verdict.
type State string

// Terminal states preserve the existing wire values.
const (
	Graded          State = "graded"
	Cancelled       State = "cancelled"
	Expired         State = "expired"
	Failed          State = "failed"
	BudgetExhausted State = "budget_exhausted"
)

// Reason is a fixed service failure category. Host errors are not serialized.
type Reason string

// Worker reason categories preserve the existing wire values.
const (
	ObservationDeadline Reason = "observation_deadline"
	WorkerCancelled     Reason = "worker_cancelled"
	AssessmentEncoding  Reason = "assessment_encoding"
	InvalidObservation  Reason = "invalid_observation"
	UsageReconciliation Reason = "usage_reconciliation"
	GraderReservation   Reason = "grader_reservation"
	GraderDispatchClaim Reason = "grader_dispatch_claim"
	GradingProjection   Reason = "grading_projection"
)
