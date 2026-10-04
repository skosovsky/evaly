package optimizer_test

import (
	"context"
	"testing"

	"github.com/skosovsky/evaly"
	"github.com/skosovsky/evaly/optimizer"
)

func TestSearchRoundTripKeepsTypedCandidates(t *testing.T) {
	// Arrange.
	config := searchConfig(t, 20)
	// Act.
	result, err := optimizer.Search(context.Background(), config)
	if err != nil {
		t.Fatal(err)
	}
	restored, err := optimizer.RestoreResult(result, config.Codec)
	// Assert.
	if err != nil || restored.Revision != result.Revision {
		t.Fatalf("restore: %v", err)
	}
	result.Version = 3
	if _, err = optimizer.RestoreResult(result, config.Codec); err != evaly.ErrUnsupported {
		t.Fatalf("old version: %v", err)
	}
}
func TestCandidateRestoreRejectsCodecDrift(t *testing.T) {
	// Arrange.
	codec := evaly.JSONCodec[recipe]{ID: "candidate", Version: "1"}
	candidate, err := optimizer.Seal("candidate", "", "algorithm", recipe{}, codec)
	if err != nil {
		t.Fatal(err)
	}
	changed := codec
	changed.Version = "2"
	// Act.
	_, err = optimizer.RestoreCandidate(candidate.Record(), changed)
	// Assert.
	if err == nil {
		t.Fatal("changed codec accepted")
	}
}

func TestStaticProposerDecodesFreshValues(t *testing.T) {
	// Arrange.
	codec := evaly.JSONCodec[map[string]int]{ID: "allocation", Version: "1"}
	original := map[string]int{"workers": 2}
	candidate, err := optimizer.Seal("allocation", "", "static", original, codec)
	if err != nil {
		t.Fatal(err)
	}
	proposer := optimizer.NewStaticProposer[map[string]int, int, int](
		"static-v1",
		[]optimizer.Candidate[map[string]int]{candidate},
	)
	original["workers"] = 99
	request := optimizer.ProposalRequest[int, int]{Round: 0, Maximum: 1}
	// Act.
	first, err := proposer.Propose(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	first.Candidates[0].Value["workers"] = 44
	second, err := proposer.Propose(context.Background(), request)
	// Assert.
	if err != nil || !second.Exhausted || second.Candidates[0].Value["workers"] != 2 {
		t.Fatal(second, err)
	}
}
