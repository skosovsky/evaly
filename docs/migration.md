# Current API and artifact migration

This is the authoritative migration guide for the 6 October 2026 remediation.
The current contracts and schemas define supported formats. Historical reports
describe the commits they checked; they are not alternate current contracts.
There are no deprecated aliases, legacy readers or automatic artifact migration.

| Artifact | Supported version |
|---|---:|
| envelope, dataset, evidence, view, observation, calibration | 1 |
| candidate, scenario, comparison-policy | 2 |
| experiment, assessment, observation-result, comparison, HTTP request/response | 3 |
| search | 6 |

All 16 schema files are under `schemas/`. Replaced versions are unsupported.
Do not change a stored version number and recompute its checksum to simulate
migration. Rebuild/reseal using authoritative source inputs and the current
contract, or keep the original artifact with the original reader outside evaly.
An integrity checksum does not establish producer truth or dataset independence.

## Numeric values and codecs

Budget units remain finite nonnegative float64. MemoryBudget accounts for the
exact binary rational value of each accepted input, with one upward rounding for
Used reporting. No epsilon permits overspend; small liabilities survive large
reservations, settlement and release. Admission may differ from the old rounded
sum. Unknown usage retains liability after Claim; error does not prove no dispatch.
See [budget accounting and recovery](budget.md).

JSONCodec decodes interface-valued numbers as json.Number. Typed int/struct decode
still works and overflow is rejected. Canonical identity retains decimal/exponent
spelling; it does not promise preserving arbitrary Go dynamic types. Seal must
reject a representation that cannot round-trip canonically. Valid acyclic aliases
work; true cycles fail. Custom codecs must keep identities/callbacks stable and transfer caller-owned
Encode bytes that remain stable after return, without reusing them on a later
call. Decode must produce independent mutable values.
See [codec representation and limits](codecs.md).

## Constructors and executable configuration

Replace HTTP Handler with `NewHandler(inputCodec, outputCodec, maxBytes, invoke)`
and handle its `(http.Handler, error)` during setup. HTTP is v3; redundant target
capability flags were removed. Prior response layouts are not read. MaxBytes is
a symmetric wire limit, not a total callback memory bound. Contexts provide
cooperative cancellation and pre-dispatch guards, without rollback/hard interruption.
See [HTTP protocol](http-protocol.md).

AbsenceGrader returns a validated Absence stock adapter rather than GraderFunc.
Its kind/predicate must be valid before grading; malformed configuration fails
preflight even on empty evidence. Replace `sink.MarkIncomplete(reason)` with
`sink.MarkIncomplete()`, including custom EvidenceSink implementations. The fixed
host_incomplete category preserves privacy. Complete empty channels still support
absence proofs; graded/complete is not itself a quality pass.

Executable configs and callback adapters have no wire persistence contract.
Their JSON tags were removed; encoding/json may still marshal some values, but
that output is not a supported recipe. Use explicit Record/Envelope types and
ComparisonPolicy. Configure mutable stock settings before use and keep them
stable while concurrent operations run. Host callback registries remain host-owned.
See [preflight](preflight.md) and [concurrency](concurrency.md).

## Measurement and optimizer

Rename GatePolicy.MinimumQuality to QualityThreshold (`quality_threshold` on the
wire). It is a lower bound for higher-is-better and an upper bound for
lower-is-better; equality passes. The safe failure reason also becomes
quality_threshold. Comparison is v3, policy v2 and search v6 because they contain
the changed gate/comparison representation.

NumericObjective has only Descriptor. Move GraderID/MetricName into
Descriptor.SourceGrader/SourceMetric. Built-in descriptors require
AllDeclaredEligibility and AllRepeatsRequiredMissingness; custom semantics use a
host Objective. A stock numeric source grader must be planned on both experiments;
a missing runtime metric remains missing/inconclusive. MeasurementDiagnostic
localizes the first failed measurement by side/case/repeat/fixed category without
serializing a raw callback error. Quality and infrastructure health gates remain
separate. Rename Calibrate to EvaluateBinaryAgreement: it reports binary agreement
and confusion, without fitting probabilities or learning thresholds.

Candidate/Proposal/Lineage Parent becomes ParentRevision; candidate artifacts use
v2. Search v6 validates ordered attempts, derived lists/ranking/usage/states and
legal terminal phase combinations; it binds callback experiments to concrete
datasets and derived IDs before accepting them. Provenance includes dataset and
baseline revisions. Candidate limits bound received counts, not arbitrary host
string/description bytes. A calibration winner is not holdout pass or deployment
permission. See [optimizer migration details](optimizer-remediation.md).

Observation Result.State/Reason use exported State/Reason types and constants;
string conversions in consumers must be explicit. The wire values are unchanged,
so observation-result stays v3. Flush does not close admission: stop producers
before final Flush, or Cancel to close admission and request cancellation.

## Conformance and release tooling

LifecycleWithOptions allows host namespaces and test-only timeout settings.
Cleanup is registered immediately after Prepare returns, including partial-handle
errors; it runs with a detached bounded context. Prepare that never returns a
handle cannot supply one for cleanup. Budget conformance now tests Claim/Release,
idempotency, unknown liability, conflicts and concurrent claim-once. Hosts should
rerun conformance for their adapters. Passing pre-cancellation tests does not
prove in-flight callback cooperation. See [conformance](conformance-contract.md).

Only the root module is released; contracttest is test-only. Release uses a
disposable checkout, exact staged go.mod and exact atomic tag refspec. Source
HEAD/index/worktree/tags are preserved. Read the explicit remote-outcome recovery
instructions in [release contract](release.md). Linux-compatible editing is used;
local fixtures were executed on macOS, not certified on Linux or real remotes.
