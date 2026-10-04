package optimizer_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"testing"

	"github.com/skosovsky/evaly"
	"github.com/skosovsky/evaly/optimizer"
)

func resealSearch(t *testing.T, r optimizer.Result) optimizer.Result {
	t.Helper()
	r.Revision = ""
	bytes, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	bytes, err = evaly.CanonicalJSON(bytes)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(bytes)
	r.Revision = hex.EncodeToString(digest[:])
	return r
}

func TestSearchRestoreRejectsRehashedFalseClaims(t *testing.T) {
	// Arrange: a valid two-candidate history with a selected winner and holdout.
	c := searchConfig(t, 30)
	result, err := optimizer.Search(context.Background(), c)
	if err != nil || result.HoldoutComparison == nil || len(result.History) != 2 {
		t.Fatal(err, result)
	}
	bytes, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	variants := map[string]func(*optimizer.Result){
		"quality_without_comparison":        func(r *optimizer.Result) { r.History[0].Comparison = nil },
		"missing_best_measured":             func(r *optimizer.Result) { r.BestMeasured = "" },
		"candidate_limit":                   func(r *optimizer.Result) { r.MaximumCandidates = 1 },
		"stopped_round_in_completed_search": func(r *optimizer.Result) { r.RoundHistory[0].State = "stopped"; r.RoundHistory[0].Reason = "deadline" },
		"holdout_wrong_comparison":          func(r *optimizer.Result) { r.HoldoutComparison = r.History[0].Comparison },
		"negative_proposal_usage":           func(r *optimizer.Result) { r.ProposalUsage.Units = -1 },
		"negative_evaluation_bound":         func(r *optimizer.Result) { r.EvaluationUnits = -1 },
		"negative_round_usage":              func(r *optimizer.Result) { r.RoundHistory[0].Usage.Units = -1 },
	}
	for name, mutate := range variants {
		t.Run(name, func(t *testing.T) {
			// Arrange: use a fresh copy; recompute the checksum after each corruption.
			var bad optimizer.Result
			if err := json.Unmarshal(bytes, &bad); err != nil {
				t.Fatal(err)
			}
			mutate(&bad)
			bad = resealSearch(t, bad)
			// Act / Assert: semantic false claims reject independently of checksum.
			if _, err := optimizer.RestoreResult(bad, c.Codec); err == nil {
				t.Fatal("accepted inconsistent search", name)
			}
		})
	}
}
