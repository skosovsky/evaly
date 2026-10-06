package optimizer

// Shared vocabulary preserves the existing persisted values.
const (
	deadlineReason    = "deadline"
	candidateEncoding = "candidate_encoding"
	identityConflict  = "identity_conflict"
	evaluationBudget  = "evaluation_budget"
	comparisonFailure = "comparison_failure"
	candidateDecoding = "candidate_decoding"

	selectedPathLength = 3
	incompleteState    = "incomplete"
	sealedState        = "sealed"
	selectingState     = "selecting"
	completedState     = "completed"
	evaluatedState     = "evaluated"
	failedState        = "failed"
	invalidState       = "invalid"
	proposingState     = "proposing"
	stoppedState       = "stopped"
)
