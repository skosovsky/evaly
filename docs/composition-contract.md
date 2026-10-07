# Streaming composition contract

This contract is normative for the optional `integrations/recipes` consumer module.
Core owns immutable evaluation records, lifecycle, grading and budget ports. The
host owns domain types, external SDKs, instructions, privacy, pricing, accounting
units and delivery recovery. Core imports no integration dependency.

## Adapter contracts (declared before implementation)

`StreamingTarget[I,O,E]` has an identity, conversion, opener, frame observer and
output projection. `Open` binds the trial context and returns a single-use stream;
`Observe` synchronously writes domain events through the trial evidence sink.
`Project` returns a permitted typed output only after complete consumption.
No deferred handle is supported. Callbacks must cooperate with context and must
not spawn work that survives their return. Configuration is immutable during use.

| Source state | Target result | Accounting / evidence |
| --- | --- | --- |
| completed, completed provider outcome | project output | terminal receipt, declared coverage |
| completed, refusal/tool calls/incomplete/paused | unsupported/error | retain receipt, mark incomplete |
| source error or premature EOF | target error | partial receipt/events, incomplete |
| cancellation or interrupted consumption | cancelled/error | partial receipt/events, incomplete |
| observation/projection/close error | target error | retain known receipt, incomplete |

Consume synchronously, always close, then read final status. Preserve all causal
errors with `errors.Join`, including cancellation and close errors; no error text
is persisted. Provider outcome never establishes business success. The host grades
actual effects independently from answer text. A source error is not retried.

`Conversion` binds identity, source/target units (source counters are tokens only), total or input/output mode,
positive rates (exact binary rationals), upward whole-unit rounding and a finite upper bound. The same
policy converts reservation receipts and final target/judge receipts. Presence is
explicit. Total-only receipts work in total mode, never fabricate a breakdown,
and are unknown in component mode. Unknown retains liability. Invalid metadata,
incompatible units, arithmetic overflow and receipts exceeding the bound return
an error with unknown usage. Cache/reasoning counters are overlapping components,
not additive expenses. Stream updates are accumulated by the source SDK; convert
only its final cumulative snapshot, never sum cumulative frames.

`CaptureBridge` serializes ordered events into an existing EvidenceSink. Original
positive source sequence and correlation are retained. Source omission, sampling,
truncation, failure or incomplete terminal report calls MarkIncomplete; contiguous
local numbering cannot prove source coverage. Payload retention is performed by
CapturePolicy before storage. The host must classify nested fields/references.

`JudgeAdapter[I,O,R]` passes the trusted rubric and projected data as separate
arguments to a host invocation callback returning a canonical response plus error.
Its decoder returns a Grade; usage is converted even on error. Unavailable is
InsufficientEvidence, not a zero score; invalid decoding is GraderError through
existing grading infrastructure. These boundaries do not establish live model
injection immunity. Response privacy and capture remain explicit host duties.

`ProjectEvaluations` projects a validated experiment into individual assertion,
metric and unmeasured-status records. IDs hash artifact revision, experiment/trial,
full grader identity, result kind/name and export policy identity. Error/missing
results have no score/outcome. Metric unit/scale/range/direction remain in the host
record: the external evaluation DTO has no corresponding fields and is not a
lossless artifact. Never infer metric semantics from telemetry alone.

`EvaluationSink` applies a required host projection before recording. It permits removal (not replacement) of association/provenance/artifact references
for privacy; association is retained or removed as a complete tuple. It rejects
identity/measurement metadata rewrites, validates every projected record before
side effects, and uses a host-provided atomic delivery callback per record. The
callback owns durable dedup and changed-content conflict. A local dedup reference
holds a mutex across record acceptance and supports repeat after ambiguous ack;
it is process-local only. Failure after some records means partial delivery; retry
keeps IDs and the host registry avoids duplicate observations. Export never alters
the artifact verdict. No exactly-once or durable outbox claim is made.

## Reproducibility

Store adapter identities and conversion policy identity in Provenance.Provider;
CapturePolicy and grading projection have their existing manifest identities.
Privacy for artifacts and exported records is independent of telemetry sampling.
Source and published dependency matrices run the same semantic tests. No network
credentials are required. SDK test transports prove contract composition, not
production service availability or model accuracy.
