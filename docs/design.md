# evaly contracts

Status: implementation contract. Original requirement authority: `.cursor/tasks/task1.md`.
Current execution changes are governed by `.cursor/task/01-execution-and-assessments.md`
and the normative [execution contract revision 2](execution-contract.md).
Experiment, assessment and observation-result formats use revision 3; search uses revision 4;
scenario, comparison and HTTP formats use revision 2. Calibration uses revision 1.
Their previous major formats are unsupported. Unaffected formats retain
their declared revisions. No compatibility reader or migration layer is supplied.

Task 02 is governed by the normative [evidence and export contract](evidence-contract.md),
[wire validation contract](wire-contract.md), and [HTTP protocol](http-protocol.md).
HTTP requests and responses move to revision 2. A remote invocation explicitly
declares event delivery completeness independently of target success. Runtime
wire decoding checks required presence before callbacks; domain payloads remain
the responsibility of host codecs.

Task 03 introduces the normative [measurement contract](measurement-contract.md),
[paired execution schedule](paired-contract.md), and [binary calibration](calibration-contract.md).
Comparison requires an explicit objective declaring identity, scale, direction,
eligibility, missingness and aggregation revisions. Assertion and numeric objectives
are reference implementations of this same contract. Gates require both per-side
coverage and matched independent-case count/coverage; reported change and uncertainty
use the same matched cases. Paired dispatch interleaves sides within case/repeat
slots with isolated handles. Calibration reports confusion counts and rates with
explicit denominators; unavailable rates have no invented numeric value.
Host owns group/time/content independence and domain interpretation. Product gates
are thresholds, not significance tests; no cluster bootstrap or sequential testing
is claimed.

Task 04 is governed by the normative [bounded search contract](search-contract.md).
The optional optimizer executes bounded proposal rounds through one protocol,
including static enumeration. It preserves typed candidates and declared objective,
constraints, split, stop and tie revisions. Best measured quality is distinct from
a feasible selectable winner. Calibration feedback contains permitted summaries
and lineage; evidence references require explicit host projection. Holdout is
evaluated only after winner selection and cannot initiate another selection round.
Technical ID disjointness is necessary; a revisioned host split validator owns
group/content checks. No automatic semantic duplicate detection is claimed.

## Task 05 integration and CLI contract

The normative [integration contract](integration-contract.md) covers a host-owned
multi-action fixture exercised through the same behavior in process and over HTTP.
Semantic evidence and usage must survive an error after an effect. Delivery failure
must prevent proving the absence of forbidden actions. Persisted permitted views
support offline re-score without target, tools or lifecycle capability; a new grader
revision creates a child assessment and cannot change the original experiment.
An online assessment may retain partial paid results and explicit skipped graders.

CLI compare requires an explicit versioned gate and a serializable reference
objective. It must reject unsupported callback objectives and unknown policy
versions rather than silently choosing a fixture objective. Its reports identify
trial, case, revision, target and seed; runnable replay requires a supported host
command. CLI exit classes remain pass 0, quality fail 1, inconclusive 2 and
invalid/infrastructure/usage 3. The fixture command remains an offline demonstration.

Shared [failure conformance](conformance-contract.md) accepts host factories; it does not prescribe application
types or run an agent. The optional [live integration procedure](live-integration.md)
uses the same ports without introducing vendor SDKs, credentials or automatic paid
calls into core or CI. Scripted checks do not establish LLM accuracy or injection
immunity. Current checks and historical acceptance are distinguished in
[acceptance](acceptance.md) and [validation](validation.md).

## Ownership and package boundaries

The root package owns versioned envelopes, sealed datasets, trial lifecycle,
evidence capture, grader protocols, experiments, comparison and artifact ports.
Caller owns input `I`, output `O`, reference `R`, environment `E`, codecs,
target implementation and all authentication, provisioning and production state.
There is no universal message or agent model. The core imports only Go's standard
library. Optional `observation`, `optimizer` and `adapters/httpjson` packages use
core ports and do not become core dependencies. No vendor SDK capability is claimed.

Both modules require Go 1.27.1, matching the current stable toolchain.
Development and CI use golangci-lint v2.14.0. There are no dependencies on neighboring modules.

## Identity, codecs and datasets

`Codec[T]` declares an ID, domain schema version, Encode and Decode. Bytes used
for identity MUST be canonical and reversible. Built-in `JSONCodec[T]` accepts
JSON values with finite numbers, string object keys, exported struct fields and
acyclic slices/maps/pointers. Custom JSON marshalers must produce valid JSON;
canonicalization sorts object keys, rejects duplicate keys and keeps JSON number
spelling (1 and 1.0 are distinct). It rejects invalid UTF-8, nonfinite numbers,
trailing data and unrepresentable values. No implicit reflection-based store codec.

`DatasetDraft[I,R]` validates then seals through codecs. Sealing encodes all caller
values and stores private bytes; every access decodes new values. Identity includes
case ID, input/reference bytes, metadata, evidence requirements, generation lineage,
codec/schema IDs, selection, parent revision and ordered case revisions. Duplicate
IDs are rejected even with identical payloads. Reference is optional; a grader
requiring it reports not applicable/insufficient evidence explicitly. Generated
labels remain unvalidated until an explicit host validation. Generation preserves
parent, generator/model revision, seed and replay/search mode. Labels cannot seal
themselves. Migration is a new dataset with an explicit parent revision; no in-place
mutation or implicit codec conversion. Aliases must resolve to a sealed revision
before dispatch.

## Trial contract and concurrency

`Target[I,O,E]` receives input and `TrialContext[E]` (identity, case revision,
seed, environment, evidence sink and budget). `Lifecycle[E]` declares fixture/reset
revision and isolation: isolated or serial/shared. Prepare may return a partial
handle together with an error; Cleanup receives it and must tolerate partial setup.
Every trial uses Prepare -> Reset -> target -> collect -> Cleanup. Failed reset
never dispatches target. Cleanup has a separate status, runs with a fresh bounded
context, and never overwrites execution/quality. Ports must honor context deadlines;
Go cannot kill an uncooperative callback. No detached goroutines are used to fake
cancellation. This cooperative bound is an explicit capability requirement.

`RunPlan` fixes repeats, concurrency, timeout, cleanup timeout, retry attempts,
seed, stop-on-infrastructure policy and dispatch reservation units. Worker count is
bounded; unisolated fixtures force serial execution and comparisons cannot assert
independence. Each case/repeat/attempt has a stable unique ID. Only setup errors
can retry automatically; target errors/side effects never silently retry. All
attempts survive. Cancellation stops dispatch, forwards context and retains already
performed effects. A retry changes attempt, not case revision. Plan changes change
experiment identity. A host reservation budget is required for a hard monetary cap;
without one usage is observation only. `MemoryBudget` is a mutex-atomic local
reference adapter. Reservations are idempotent by dispatch identity; `Claim` atomically authorizes
exactly one dispatch per reservation (not exactly-once external effect). Reusing
a spent identity cannot fund a second dispatch. Reconcile including unknown marks
liability dispatched; only unclaimed reservations can be released. Unknown actual
usage retains the full reservation, known actual usage reconciles it. Undispatched
reservations can be explicitly released. Units must be nonnegative finite; actual
usage exceeding a reservation is a contract violation, not automatically affordable.

Paired execution uses the same sealed dataset/fixture revision and separate prepared
handles for each side. Each scheduled case/repeat runs both sides in seeded order;
bounded workers may overlap different slots. Schedule provenance records the planned
within-slot order, not a promise of simultaneous or deterministic global wall-clock
order. Manifest timestamps bound the time window. Shared environments or inconsistent
reset revisions reject paired dispatch. Seeds do not guarantee provider determinism.

## Evidence and grading

Capture states: open -> sealed or incomplete. Evidence events are ordered,
versioned, correlated and projected by a host `CapturePolicy` BEFORE retention.
The default policy drops payload and references. Allowlisting fields is an explicit
host choice; callers must classify sensitive data. Reference syntax excludes
userinfo, fragments, malformed query strings and recognized credential query keys;
this bounded check does not detect arbitrary secrets. Safe artifact IDs and non-HTTP
URIs remain available under host retention policy. Outcome references use this
same capture path. Unknown event versions/types, sequence gaps, recording errors, explicit
truncation and missing required coverage prevent completeness. Completeness is per
event kind, never inferred from sampled telemetry. Sink bounds event count, raw/retained payload bytes and diagnostics (at most 32
errors and 32 gap markers). Seal is irreversible, snapshots and policy inputs are copied.

A separate typed grading projection is required in run configuration; its revision
is identity-bearing. It is the host boundary for sanitizing input/output/reference;
raw target output is not persisted. Graders receive that projection and sealed
permitted evidence, never target/tools. Grader outcomes: scored, not_applicable,
insufficient_evidence, grader_error. Scored results preserve named metric scale,
direction, assertions, bounded reason codes and evidence references. Nil/invalid
results are grader errors; timeout is not zero. Multiple graders use explicit
`all` or `any` assertion policy while keeping every original result; missing/error
required graders cannot produce a pass. Revision includes implementation, rubric,
model, prompt and configuration IDs; usage is distinct from target usage.

LLM judge requests separate trusted instructions and untrusted typed data; this is
a protocol boundary, not an injection-proof claim. Scripted adapters make tests
repeatable; real model accuracy remains unverified. Pairwise requests use blind A/B,
run both orders, normalize preference, and retain abstention/disagreement. Calibration
reports TP/TN/FP/FN, unreviewed/error/abstain counts, missing labels and coverage
per grader revision. Rates retain numerator/denominator and are unavailable when
the denominator is zero. Host slice labels produce aggregate reports without raw
inputs; no human disagreement claim is made without corresponding human labels.
SavedViewRecord v1 stores the permitted view, explicit input/output/reference codec
identities and checksum revision; SaveSavedView/LoadSavedView restores after
filesystem reopen without target access. Offline re-score receives saved permitted views and evidence only; creates a new
revision with parent observation/assessment. It cannot dispatch a target.

## Experiments, comparisons, gates and uncertainty

Experiments: planned -> running -> sealed or incomplete. Sealed means the complete
execution record has been collected, including explicit setup/target failures; it
does not mean quality pass or full grading coverage. Cancelled/undispatched runs
are incomplete. Private frozen manifests/results are exposed as deep copies and
hashed by canonical wire content. Provenance includes target/model/prompt/tool/policy,
fixture/reset, unknown versions, dataset/selection, grading/capture/projection,
provider parameters, repeat/retry plan and pair metadata. Unknown is explicitly
recorded. Re-score never rewrites sealed results.

Comparison requires sealed revisions, compatible datasets/case revisions, fixture,
grader/capture/projection protocols, plans and isolation, and an explicit metric.
Aggregates use final attempt per case/repeat but retain earlier attempts. Coverage
is measured cases / declared eligible cases; failed setup/judge/missing evidence
cannot reduce the eligible denominator. A case is measured only when every planned
repeat yields the required measurement. Native case values average repeats;
independent unit is case. The same explicit Objective port serves assertion and
numeric measurements, including unit, scale, direction and selection provenance.
Setup, target, grader and cleanup failures are separate counters.
Gate: pass/fail/inconclusive/invalid_comparison, with versioned policy and reasons.
Minimum side and matched coverage and matched independent-case count take precedence
over high observed quality. Quality thresholds, delta, regression and uncertainty
use the same matched case set. Product thresholds are not significance tests.

Uncertainty method: seeded paired case bootstrap, percentile interval for mean case
difference (default 95%). `Interval.Method` identifies
`paired_case_bootstrap_percentile_pcg_v2`: local `math/rand/v2` PCG, with
`uint64(seed)` preserving the signed seed's two's-complement bits as the first
word and zero as the second. Percentile indices use floor(0.025 × (samples−1))
and ceil(0.975 × (samples−1)). Bootstrap samples differ from the legacy generator;
paired schedules retain the unchanged `case-repeat-v1` SplitMix64 mapping.
Assumes independent representative cases; repeated trials
are averaged within case. No matched pairs -> no interval; one pair -> degenerate
interval explicitly labeled insufficient independent cases. Multiple comparisons
are exploratory without a correction; optimizer does not claim significance or
holdout independence after reuse.

## Artifacts, schemas and interoperability

Wire envelope v1 contains kind, stable ID, canonical JSON data and SHA-256 checksum.
Schemas live in `schemas/`. Unknown required major/kind is unsupported. Extensions
are preserved as raw JSON where the envelope promises round-trip. Inner typed
schemas reject unknown required versions. Filesystem store writes staged bytes,
validates envelope and checksum, syncs file, then atomically publishes a same-device
file via exclusive link. Existing identical identity is success; conflicting bytes
are conflict. Directory sync completes commit. A failed durability sync is delivery
failure even if a valid file was published; retry uses the same identity. Staged
files are ignored. Reopen verifies checksums before returning any baseline. No
multi-host atomicity/remote filesystem promise. Supported local Linux/macOS regular
filesystems with atomic link and fsync; Windows/network/cloud-sync semantics are
unverified. Host owns permissions, access, retention and deletion. Missing retained
evidence marks replay unavailable; never promises eternal replay.

Human report shares permitted records and names IDs, denominators, reasons and
reproduction metadata. A runnable replay command requires explicit host support. Export uses the same stable observation ID; delivery is separate
from verdict. Backend advertises dedup capability; absent it retries may duplicate.
Interop capability mappings report lost outcome/reset/evidence/status/scale; a
boolean-only sink cannot silently consume incomplete/error records. Export rejection
distinguishes invalid, conflict, unsupported, cancelled and delivery failure; it
does not choose retries or alter quality verdicts. HTTP JSON v2
is optional, bounded request/response and context-aware, with explicit domain
codecs and status mapping. No advertised SDK compatibility.

CLI `evaly fixture --store DIR --id ID` runs offline fixture evaluation and commits
its experiment; `evaly compare --store DIR --baseline ID --candidate ID --policy FILE` reads sealed
artifacts, writes human comparison and exits: 0 pass, 1 quality fail, 2 inconclusive,
3 invalid comparison/infrastructure/usage. CLI never deploys production changes.

## Online observations and optimizer

Observation identity/revision is immutable, with sampling rule/reason/probability,
population/window, outcome delay and lineage. Controlled and observational results
have distinct modes. Optional worker uses a bounded queue, fixed worker count,
clock-injected deadline, reservation budget and nonblocking enqueue. Overload and
closed/cancelled are explicit rejected results; flush/cancel respect caller deadlines.
Production never waits for grader. Host owns storage/retention. Delayed evidence
creates a new observation/grading revision linked to the original.

Optimizer runs bounded proposal rounds over typed immutable host candidates;
static enumeration is a reference proposer. Proposal receives training/calibration
and copied permitted feedback, never holdout. Technical split IDs are disjoint;
an optional revisioned host validator checks declared group/content keys. This
records successful host validation, not a statistical proof of independence.
Candidate codecs seal descriptions. Evaluations are ordinary experiments, preserved
even if failed/incomplete. Budget claims authorize unique proposal/evaluation
dispatches, while target/grader reservations account for their work. Stop prevents
further proposal, evaluation or holdout calls. Best measured quality is distinct
from a complete, feasible, passing-gate winner; objective direction and lexical
revision ties determine ranking. Holdout runs once after selection, records ledger
reuse and cannot trigger fallback selection. Search identity preserves objective,
constraints, proposal, split, stop, tie and feedback revisions and ordered rounds.
Restore validates structure, identity links, ranking, bounds and stop semantics
before returning a detached record with caller-codec-restored candidates. No global
optimum, distributed search or automatic production update is claimed.

## Scenario provenance and replay

ScenarioStep requires behavior Revision and receives ScenarioContext with seed,
mode and zero-based step index. RunScenario records v2 driver revision,
replay/search mode, seed, bounds and optional generation lineage, using explicit
state/output codecs to snapshot each step before the next mutation. RestoreScenario
reconstructs the saved trajectory; executing RunScenario in replay mode is a
separate operation with a different revision than searching. DraftFromScenario
attaches driver/trajectory revisions to an unvalidated generated draft. Post-Step
cancellation takes precedence over a final answer.

## Executable wire contracts

`internal/schemagen` emits all explicit envelope schemas including nested plan,
usage, grading, aggregate, observation, search, saved view and HTTP records.
`contracttest` is an isolated test module with a pinned independent Draft 2020-12
validator. It validates produced artifacts and rejects invalid nested usage,
cleanup, grade status, plan and major version. These test dependencies do not enter
the core module graph. Domain RawMessage fields are governed by caller codecs.
