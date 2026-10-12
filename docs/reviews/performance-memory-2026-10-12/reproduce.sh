#!/usr/bin/env bash
set -euo pipefail
# Run from repository root; run only one instance, without other Go builds.
export GOWORK=off GOMAXPROCS=8
export GOCACHE="${TMPDIR:-/tmp}/evaly-performance-cache"
export GOLANGCI_LINT_CACHE="${TMPDIR:-/tmp}/evaly-performance-lint-cache"
export GOPATH="${TMPDIR:-/tmp}/evaly-performance-gopath"
export GOMODCACHE="${TMPDIR:-/tmp}/evaly-performance-modcache"
export GOBIN="${TMPDIR:-/tmp}/evaly-performance-bin"
results="$PWD/docs/reviews/performance-memory-2026-10-12"
baseline="$(mktemp -d "${TMPDIR:-/tmp}/evaly-baseline.XXXXXX")"
git archive ca98aafa445eb6ab82f1825941aa4b2cb82ee63d | tar -x -C "$baseline"
cp performance_benchmark_test.go artifact_memory_test.go capture_memory_test.go "$baseline/"
cp "$results/baseline-savedview-characterization.go.txt" "$baseline/savedview_memory_test.go"
characterization='TestCapture(DeepEventBoundary|DeepSealBoundary|InvalidMetadataCanonicalRetentionSize|SaturatedCallbackErrorPrecedence|PolicyInputNormalizationAndIsolation|ReturnedRecordAndPolicyOutputIsolation|CanonicalRetentionSize|SealConfigurationStringNormalization|SealRejectsNormalizedCoverageKeyCollision)|TestSavedViewCodecCallbackCounts'
(cd "$baseline" && go test . -run "$characterization" -count=1 -v) > "$results/baseline-characterization.log"
go test . -run "$characterization" -count=1 -v > "$results/candidate-characterization.log"
# Ordinary benchmark fixtures are copied identically; characterization asserts baseline contracts.
bench='BenchmarkAudit|BenchmarkSavedViewValidation|BenchmarkArtifactIDCorpus'
(cd "$baseline" && go test . -run '^$' -bench "$bench" -benchmem -benchtime=1s -count=7) > "$results/baseline.log"
go test . -run '^$' -bench "$bench" -benchmem -benchtime=1s -count=7 > "$results/candidate.log"
go install golang.org/x/perf/cmd/benchstat@v0.0.0-20261009192801-be2c69fb417e
"$GOBIN/benchstat" "$results/baseline.log" "$results/candidate.log" > "$results/benchstat.txt"
python3 "$results/check-gates.py" > "$results/gates.json"
go test -count=1 ./... > "$results/test.log"
make test > "$results/race.log" 2>&1
make lint > "$results/lint.log" 2>&1
make test-integration > "$results/integration.log" 2>&1
make test-e2e > "$results/e2e.log" 2>&1
for target in FuzzArtifactIDLanguage FuzzCaptureInputNormalization; do
 go test . -run '^$' -fuzz "^$target$" -fuzztime=30s > "$results/$target.log"
done
# Profiles deliberately separate from performance gate samples.
(cd "$baseline" && go test . -run '^$' -bench 'BenchmarkAuditCapture/16384|BenchmarkSavedViewValidation/bytes_16384' -benchtime=1s -cpuprofile="$results/baseline.cpu.pprof" -memprofile="$results/baseline.alloc.pprof") > "$results/baseline-profile-run.log"
go test . -run '^$' -bench 'BenchmarkAuditCapture/16384|BenchmarkSavedViewValidation/bytes_16384' -benchtime=1s -cpuprofile="$results/candidate.cpu.pprof" -memprofile="$results/candidate.alloc.pprof" > "$results/candidate-profile-run.log"
for side in baseline candidate; do
 go tool pprof -top -alloc_space "$results/$side.alloc.pprof" > "$results/$side-alloc-top.txt"
 go tool pprof -top "$results/$side.cpu.pprof" > "$results/$side-cpu-top.txt"
done
