# Performance and memory implementation acceptance evidence

Scope: evaly only, baseline ca98aafa445eb6ab82f1825941aa4b2cb82ee63d. Public API, schemas, dependencies, checksum/revision algorithms and scheduler remain unchanged. This package records local CPU/allocation pressure, not leaks or retained heap.

## Implementation

- Artifact ID: equivalent bounded ASCII predicate, without compilation or allocations.
- Capture: copy and normalize Event input using RawMessage marshaling plus JSON-equivalent string normalization; validate projection before capacity rejection. Policy callbacks still execute on saturated capture. For retained Events, use the already-owned canonical payload and copy references. Conservative container-count detection retains the original whole-Event input/projection/retention and whole-EvidenceRecord JSON checks for deep or structurally complex payloads, preserving nesting limits and error precedence. Encode the whole Event before capacity checks on every path. For shallow payloads, measure bytes with json.Marshal after projection canonicalization; sorting the fixed Event wrapper changes order, never its byte count. Complex payloads retain canonical(out) before capacity. Seal returns owned wire copies, preserving the legacy JSON path for non-UTF-8 host configuration strings.
- SavedView: copy mutable fields of normalized private records directly. Decode fresh domain values for every grader. Only the concrete built-in JSONCodec avoids repeated canonical validation of already-validated private bytes; arbitrary host codecs still execute Decode unchanged. Save/Restore/Rescore retain complete codec preflight/reversibility callbacks. A zero SavedView.Record still follows legacy JSON normalization.

## Evidence and reproducibility

The frozen denominator is 25 in requirements.md; PASS requires all substantive checks and both independent approvals on one candidate commit. Review reports and the candidate SHA are stored separately from the committed candidate tree so recording a verdict does not recursively change the commit being reviewed.

Run `bash docs/reviews/performance-memory-2026-10-12/reproduce.sh` from repository root on the recorded Go 1.27.1 darwin/arm64 runner. Benchmarks include the complete original audit matrix plus representative invalid artifact IDs; baseline and candidate fixtures are identical. Invocations include capture construction/sealing and every required fresh decode; no measured work was moved into setup. Seven 1-second samples are used, GOMAXPROCS=8. All Go processes are serialized; profiles and race timings are excluded from performance gates. Raw logs, benchstat.txt, gates.json and check-gates.py permit independent verification. Task gates use medians, not p95. A short 150ms/five-sample repeated pair failed the saturated timing gate; both raw logs and its benchstat are retained as short-*. The full matrix was then lengthened identically for baseline and candidate to investigate transient wall-clock variation, without changing thresholds or the denominator.

baseline-characterization.log records callback/error precedence, caller/policy/record isolation, byte accounting and codec callback counts on the unchanged baseline before candidate acceptance. Permanent tests cover nested mutable fields, nil/empty/zero values, numeric lexemes, UTF-8 normalization and invalid external projections. Review B found a nesting-depth regression in the first optimized candidate: input depth 10000 invoked policy where baseline rejected before dispatch, and saturated projected depth 10000 returned capacity before the original JSON depth error. depth-*-probe.log retains the baseline/before-fix evidence (probe source depth-probe.go.txt); permanent TestCaptureDeepEventBoundary/DeepSealBoundary and the conservative legacy fallback fix these cases. The invalid-string size suspicion was not reproduced on Go 1.27.1; both raw probes are retained and a permanent exact-budget regression test covers it. FuzzArtifactIDLanguage and FuzzCaptureInputNormalization are each run for 30 seconds.

The complete repository CI targets are make lint, make test (all three modules, race, count=1), make test-integration and make test-e2e. test.log separately records root go test -count=1 ./.... Race checks include unchanged bounded concurrency, SerialShared fixtures, cleanup, cancellation, budget and retained cancelled slot tests.

See profile-analysis.md for the separately captured baseline/candidate CPU and allocation profiles. See measured-results.md for final measured medians and gate decisions.

## Interpretation limits

These results use one local Go 1.27.1 / Apple M1 Max runner and small reproducible fixtures. They do not estimate provider latency, end-to-end consumer throughput, peak RSS or production leak behavior. Independent adversarial reviews report found/unfixed errors; approval is not an absolute guarantee of correctness. No release is published and no consumer pins are changed.
