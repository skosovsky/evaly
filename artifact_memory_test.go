package evaly

import (
	"regexp"
	"strings"
	"testing"
)

var legacyArtifactPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,128}$`)

func TestArtifactIDLanguageAndAllocations(t *testing.T) {
	// Arrange.
	corpus := []string{
		"",
		"a",
		"experiment_123",
		"A-Z_09",
		"a/b",
		"é",
		"foo\n",
		"a b",
		strings.Repeat("a", 128),
		strings.Repeat("a", 129),
		"\x00",
		string([]byte{255}),
	}
	for i := range 256 {
		corpus = append(corpus, string([]byte{byte(i)}))
	}
	for _, id := range corpus {
		// Act.
		got := validArtifactID(id)
		allocs := testing.AllocsPerRun(100, func() {
			if validArtifactID(id) != got {
				t.Fatal("unstable result")
			}
		})
		// Assert.
		if got != legacyArtifactPattern.MatchString(id) {
			t.Fatalf("ID %q: language mismatch", id)
		}
		if allocs != 0 {
			t.Fatalf("ID %q: %v allocations", id, allocs)
		}
	}
}

func FuzzArtifactIDLanguage(f *testing.F) {
	// Arrange.
	for _, id := range []string{"", "experiment_123", strings.Repeat("a", 128), strings.Repeat("a", 129), "a/b", "é", "a\n", "\x00"} {
		f.Add(id)
	}
	f.Fuzz(func(t *testing.T, id string) {
		// Act.
		got := validArtifactID(id)
		// Assert.
		if got != legacyArtifactPattern.MatchString(id) {
			t.Fatalf("ID %q: language mismatch", id)
		}
	})
}

func BenchmarkArtifactIDCorpus(b *testing.B) {
	for name, id := range map[string]string{"valid": "experiment_123", "empty": "", "slash": "a/b", "unicode": "é", "too_long": strings.Repeat("a", 129), "nul": "a\x00"} {
		b.Run(name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				validArtifactID(id)
			}
		})
	}
}
