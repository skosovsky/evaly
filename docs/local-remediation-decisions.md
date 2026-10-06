# Local remediation decisions (T09)

This is the decision record for the remaining review items. T09 was independently accepted and committed as `7a37ea3`.
Final integrated acceptance is recorded separately in remediation verification.

## D15 — localized measurement diagnostics

Comparison retains at most one MeasurementDiagnostic for the first failed
measurement contract: baseline/candidate side, declared case ID, repeat index
and a fixed category (trial_clone, objective_identity_changed,
measurement_callback or measurement_invalid). Raw callback errors, measurements
and outputs are not copied into it. The existing measurement_contract_invalid
reason and fail-closed GateInvalid remain; no fallback score is supplied.
The field is informational localization, not proof of callback authenticity.
Regression cases fail on the second case/repeat on each side and verify privacy,
serialized finite diagnostics and unavailable matched means. The diagnostic is
part of the new comparison v3/search v6 schema break already made in T09.

## D16 — planned numeric source vs runtime missing metric

Built-in NumericObjective values/pointers must name a grader planned on both
restored experiments. Compare rejects a typo with ErrInvalid before measurement.
The planned manifests are authoritative for that check; no metric catalog is
inferred from successful rows. A planned grader without the requested metric
still produces metric_missing and an inconclusive coverage gate. Host Objective
implementations retain their own semantics even when their descriptor contains
source labels. The public regression exercises value/pointer stock objectives,
typo rejection and a planned-but-unavailable metric.

## D05/D60 — local invariant and zero-state cleanup

Execution recovery paths now use mustValidatedCapture for the already validated
base capture configuration. Its error is never ignored: a subsequent failure
panics with a fixed invariant message, indicating configuration/policy instability.
The unsupported-case-evidence path can still produce its ordinary SetupError when
the base configuration remains valid; it cannot silently conceal a changed policy.
This is not global recovery of host callback panics. Ports/configuration must stay
stable after preflight. Existing setup and codec-failure regressions exercise these
paths; no host error text enters retained capture diagnostics.

codecFailure now starts from a zero TrialRecord and assigns its meaningful fields
instead of listing every zero field. The grading finalizer still supplies explicit
planned/skipped results. This small cleanup is limited to this repeated terminal
record construction. Wire fields/schemas and lint settings remain complete and
unchanged; no library-wide style rewrite is performed.

## D18 — binary agreement name

Replace Calibrate with EvaluateBinaryAgreement, without an alias. The operation
reports binary assertion agreement/confusion, missing/error/abstention counts and
rates, not fitted probabilities or learned thresholds. CalibrationReport and its
wire layout remain unchanged. Host labels and holdout independence are explicit
GoDoc responsibilities. Existing confusion/rate/overlap/denominator regressions,
conformance and examples migrate to the new function name.

## D32 — worker terminal vocabulary

Result.State and Reason now use exported State/Reason types. Graded, Cancelled,
Expired, Failed and BudgetExhausted are terminal state constants. Reason constants
cover deadline, cancellation, encoding, invalid observation, accounting and budget
authorization categories. Missing projections use GradingProjection; cancelled
assessments retain the existing deadline/worker-cancellation classification.
The serialized strings are unchanged, so observation-result stays v3. Graded
means grading finished, not quality passed. Event kinds, host revisions and
sampling labels remain open host vocabulary. Custom consumers converting these
fields to an ordinary string must do that explicitly; migrated public tests use
the state constants for expiration and cancellation.

## Retained execution and host boundaries

| Item | Decision and reason | Implementation evidence |
|---|---|---|
| D01 | Keep StopOnInfrastructure. Its GoDoc specifies final setup attempt after allowed retries; target failure alone is not an infrastructure class. Fixed categories determine stopping, never parsing host error text. | executeSlot and trialInfrastructureFailure; standalone and paired schedulers inspect final slot results. |
| D06 | Keep privacy-safe failure categories. Correlate private logs by trial/case/repeat/attempt identity; raw callback errors and outputs must not enter service diagnostics. | runner trial reasons; grader normalization; execution contract. |
| D07 | Keep config safety bounds: repeats ≤10000, workers ≤1024, attempts ≤100, scheduled slots ≤1000000; scenario steps ≤10000. These bound counts, not total payload bytes, throughput or distributed capacity. | ValidateRunPlan, ValidateExperimentManifest and scenario validation. |
| D10 | Keep cooperative scenario timeout beginning after initial encoding and before driver execution. Initial encoding and final sealing are outside that deadline. Host codec callbacks must terminate; detached goroutines would not provide hard cancellation. | RunScenario ordering of Encode, WithTimeout and sealing. |
| D11 | Keep shared executeSlot for standalone/paired execution and the existing isolated side handles. Further scheduler unification has no measured need and would obscure pair ordering and cleanup ownership. No workflow DSL is introduced. | Run, RunPair and executeSlot. |
| D14 | Keep quality and cleanup health separate. A GatePass can coexist with CleanupFailures. A consumer promotion rule must explicitly require healthy cleanup/setup/target/grading, appropriate coverage and its quality gate. Quality alone is not deployment permission. | Aggregate.CleanupFailures and comparisonGate; execution contract. |
| D17 | Keep case-level bootstrap and point thresholds. Repeats are averaged within a case and do not increase independent N. Hosts own representative/independent cases and any clustered, sequential or multiple-testing analysis. | comparison aggregation/bootstrap and measurement contract. |
| D22 | Keep observation Deadline as detached bounded reconciliation timeout. Accounting may need to settle a claimed dispatch after cancellation; cancellation does not prove zero cost. Separate ReconcileTimeout lacks an operational requirement here. | evaluateGrade and missingGrade use WithoutCancel plus Deadline. |
| D29 | Keep conservative reference credential filter with its stated limits. Substring key matching can reject monkey/keynote; arbitrary opaque path secrets are host-classified. This is not a universal secret detector. | safeReference and evidence contract. |
| D30 | Keep mutable reference adapter settings with configure-before-use GoDoc. MemoryExport.Dedup/Fail and FileStore.MaxBytes/Fault must remain stable during concurrent use; a method mutex does not protect unsynchronized configuration writes. MemoryExport is not a durable production outbox. | MemoryExport and FileStore field access. |
| D31 | Keep FileStore local fsync/hardlink/no-replace behavior. Hostile swaps, network/cloud-synchronized filesystems, Windows and multi-host guarantees remain unverified; no distributed storage/ACL subsystem is added. | FileStore publication and declared StoreCapabilities. |
| D34 | Keep Flush as waiting for pending work, without closing admission. Stop external producers first, then Flush; Cancel closes admission and requests cooperative cancellation, waiting only until its caller context permits. No CloseAndDrain is added without need. | Worker.Flush and Cancel. |
| D35 | Keep shared host ports without hidden global mutexes. Slice copies do not clone grader/clock/budget implementations. Hosts supply concurrency-safe callbacks/codecs/ports or use Concurrency=1; immutable records remain distinct from config. | Worker.Start configuration copy and concurrent loops; Run workers. |
| D36 | Keep Sampling/OutcomeDelay as metadata. It does not schedule work or assign A/B traffic; expiry starts from enqueue and producers own outcome readiness and sampling. Observation grants no production permission. | Sampling and Worker.enqueue deadline. |
| D37 | Keep explicit partial count bounds without claiming a total result memory bound. Grade Reasons/EvidenceRefs are limited to 32, but metric/assertion/reason bytes and domain payloads need host limits. Do not silently truncate them or discard cost receipts; a broader byte policy requires a concrete workload. | ValidateGrade and result records; optimizer diagnostics bounds are separate. |
| D38 | Keep no implicit judge retries/fallback. GraderError is retained as an outcome. Changing judge or replaying target requires explicit host cost/identity policy; setup retries are a different protocol. | Assess/LLMGrader invoke each configured grader once; Rescore records lineage. |

## D12 — direction-neutral quality threshold

Rename GatePolicy.MinimumQuality to QualityThreshold and its serialized key to
quality_threshold. Gate failure uses that same safe reason category. The math is
unchanged: higher values must meet or exceed the threshold; lower values must
meet or fall below it. Equality passes. The numeric comparison regressions now
exercise equality and the adjacent failing float in both directions, in addition
to native means and independent regression limits.

The clear break deliberately advances comparison to v3, comparison-policy to v2
and search to v6 (its nested comparisons and gate also change). Old schemas are
removed. Resolve rejects policy v1; optimizer validation rejects old result and
nested comparison versions. Regenerate/reseal from authoritative original inputs
using the new API; do not relabel an old artifact's version or reuse its checksum.
Unrelated experiment/scenario/HTTP formats retain their current revisions.
The current schema inventory, CLI JSON example and all Go call sites use the new
field. No compatibility alias or legacy reader is provided.

## D09 — executable configuration is not a wire artifact

Remove JSON tags from RunConfig, LifecycleFuncs, TrialContext, CaptureConfig,
ObjectiveFuncs, grader/judge adapters, FileStore, observation/optimizer Config,
optimizer callback adapters/requests and Split, HTTP Target and live-port
conformance fixtures (including PairFault).
These values contain live ports, functions or runtime resources; they are not
portable recipes. Removing tags does not make encoding/json reject all such
values: some configurations can still marshal partially. That output has no
supported persistence/restore contract and must not be used as a wire artifact.

Serialize the explicit Record/Envelope types and ComparisonPolicy instead.
Callback lookup, credentials and executable reconstruction belong to the host.
Schema generation addresses explicit wire records, whose tags remain unchanged.
No schema revision is needed for this cleanup; executable configs were not among
the supported envelope types.

## D13 — canonical numeric objective reference

Remove NumericObjective.GraderID and MetricName. Descriptor.SourceGrader and
SourceMetric now select the metric and are the only stored identity. Identity
returns the descriptor unchanged. Direct validation, eligibility and measurement
reject missing sources or an assertion policy (ErrInvalid), and reject custom
eligibility/missingness labels (ErrUnsupported). The built-in algorithm supports
only all-declared-v1 and all-repeats-required-v1; custom semantics use a host
Objective rather than relabeling this algorithm.

ComparisonPolicy.Resolve constructs that same descriptor-only objective. The
canonical-reference table tests both paths, including unsupported labels and
invalid source references. Existing native-scale, lower/higher, subnormal and
conformance tests use the canonical descriptor. Policy wire identity already
stored SourceGrader/SourceMetric; this change requires no wire revision by itself.
Migration: move direct GraderID/MetricName values into Descriptor.SourceGrader/
SourceMetric, and use the built-in labels or supply a custom Objective.

## D19 — serialized policy and runtime Objective

Keep the separation. ComparisonPolicy supports only AssertionObjectiveKind and
NumericObjectiveKind. AllDeclaredEligibility and AllRepeatsRequiredMissingness
are exported constants for their implemented revision vocabulary. Values remain
unchanged on the wire. Arbitrary closures, service registry lookup and callback
deployment remain host-owned; Resolve does not load plugins or reconstruct host
functions. The unsupported-kind/selection/missingness regression table checks
that unsupported semantics fail explicitly.

## D25 — FieldPolicy ownership

Keep the public value configuration and require configure-before-use. `Allowed`
and its nested slices remain host-owned and must be immutable while the policy is
used. A struct copy or revision label does not freeze them. The policy GoDoc now
states that contract. This preserves direct configuration without inventing a
constructor that could freeze arbitrary custom CapturePolicy implementations.
Hosts needing new settings construct a separate policy/capture after stopping
users of the previous configuration; runtime map mutation is not supported.

## D26 — explicit incomplete evidence

Remove the unused reason argument: EvidenceSink and Capture now expose
`MarkIncomplete()`. Capture continues to retain only the fixed `host_incomplete`
category and invalidates channel coverage; callers correlate private diagnostics
outside the artifact. HTTP client failure helpers now take a zero-argument
callback. Existing transport, capture, absence and workflow regressions exercise
the migrated interface. No wire revision changes because stored evidence is
unchanged. Migration: replace `sink.MarkIncomplete(reason)` with
`sink.MarkIncomplete()` and update custom EvidenceSink implementations.

## D33 — measurement before optimization

Measured with Go 1.27.1, darwin/arm64, Apple M1 Max, local stock JSON codecs and
empty sealed evidence. Each fixture has an input and output string of the stated
size. The graders return scripted passing assertions; this measures local record
work and allocation, without provider latency. Command:

```sh
env GOCACHE=/tmp/evaly-go-build go test . -run '^$' \
  -bench '^BenchmarkSavedViewValidation$' -benchtime=200ms -count=3
```

Median of three samples (before benchmark helper extraction):

| Bytes per field | Operation | ns/op | B/op | allocs/op |
|---|---|---:|---:|---:|
| 128 | RestoreSavedView | 118020 | 50909 | 626 |
| 128 | fresh View | 10753 | 7411 | 49 |
| 128 | Rescore, 1 grader | 133753 | 71959 | 860 |
| 128 | Rescore, 8 graders | 300591 | 172781 | 1801 |
| 16384 | RestoreSavedView | 1559216 | 2065308 | 756 |
| 16384 | fresh View | 327226 | 453321 | 80 |
| 16384 | Rescore, 1 grader | 1943807 | 2608283 | 1029 |
| 16384 | Rescore, 8 graders | 4258606 | 5848361 | 2202 |

The current Rescore implementation already validates/restores the full immutable
saved record once before dispatch, then requests a fresh View for each grader.
Keep that implementation. Restore cost is material, but is not repeated once per
grader here; the fresh decode supplies required ownership isolation. Do not cache
mutable decoded views or share them between graders. Grade-specific evidence
validation remains separate (for example CompleteFor used by Absence).

Observation invokes Rescore separately for budgeted grader operations and hence
does repeat full record validation between those operations. The eight-grader
Rescore benchmark measures one call, not that worker path. Retain the worker
boundary too: this synthetic benchmark alone does not justify adding a public
prevalidated handle/unchecked entry point across packages. A worker workload
profile and a separately measured batch implementation would be required before
changing it. This is an explicit decision to retain the repeated worker check,
not a claim that all observation validation was optimized away.

These short measurements are not a throughput guarantee, nor do they cover large
event collections, custom codecs or production workloads. They justify retaining
the existing one-record-validation/fresh-view arrangement. Revisit optimization
when a host workload/profile demonstrates a bottleneck, preserving fail-closed
integrity and the mutating-grader isolation regression.
