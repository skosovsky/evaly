# Execution and assessment contract, revision 2

Status: implementation contract for `.cursor/task/01-execution-and-assessments.md`.
This document is normative alongside design.md. It is written before the changes
to runtime and supersedes the affected revision-1 behavior.

## Preflight and immutable case access

Structural configuration is validated before any lifecycle, target, judge or
proposal callback. Codec identities require both ID and Version; grader identities
require ID, Implementation and Rubric, and grader IDs must be unique within a plan.
The optional StructuralValidator port exposes a side-effect-free Validate method.
Library-owned callback adapters implement it to reject nil functions, missing
judge/proposal ports and malformed trusted instructions before effects. All entry
points invoke this hook when provided; opaque host implementations own their
internal configuration and may implement the same hook. Validation never makes
paid probe calls or copies arbitrary caller types by reflection.
Unsupported enums, missing revisions, unsealed inputs and malformed policies are
configuration errors. Runtime codec failures remain recorded failures rather than
proof that a paid callback was never executed. Ports promise stable identities.

Dataset.CaseAt(index) decodes only the selected private case, including a private
copy of metadata, reference and provenance. Each attempt and grader receives a new
value. Full enumeration remains available but is not used inside a per-trial loop.
No shared cache of decoded caller values is introduced.

## Grading inputs and pairwise execution

Assess takes a typed view factory `func() (View[I,O,R], error)` instead of a mutable
View. The host must produce a fresh permitted snapshot on every invocation; the
factory has no target capability supplied by evaly. SavedView.View is a reference
implementation. Run constructs fresh cases, output and evidence for each grader.
Factory failure produces an explicit grading error and cannot dispatch that judge.
Grade.Dispatched records actual invocation; an assessment or trial moves an
undispatched diagnostic into Skipped rather than retaining it as a paid grade.
The requirement for an independent copy is part of the port contract, not a claim
that arbitrary closures are mechanically sandboxed.

Pairwise inputs are codec-sealed Snapshot[T] values with explicit codec identity and
canonical digest. Each direction decodes both inputs anew. CheckPair checks context
before each dispatch and retains valid usage on every response, including errors,
invalid preferences and cancellation. An undispatched direction is separately
identified from a dispatched abstention. Caller owns any actual budget adapter.

## Partial assessments

Assessment wire revision 2 names Planned grader revisions and retains all obtained
Grades. State is complete or partial; StopReason identifies the interruption and
Skipped names planned graders that were not dispatched with their reasons. A
failed but dispatched grader is retained as GraderError, including known usage.
Completeness means every planned grader has a retained result and no collection or
reconciliation interruption; it does not mean quality pass. A reconciliation
failure can therefore leave a partial assessment with all grades present.

Canonical sealing and validation are shared between Rescore and observation worker.
Source, Parent, View, Mode, Planned, Grades, State, StopReason and Skipped all enter
revision identity. All exits after a valid observation was accepted produce an
assessment, including expiry before dispatch and queued cancellation. Grade identity
must match exactly one planned grader, with no duplicates. Unknown usage remains
unknown; missing dispatch never fabricates a scored result. Revision 1 assessments
are unsupported. Consumers and generated schemas are updated without legacy readers.

## Stop semantics and scenarios

StopOnInfrastructure means stop future target dispatch on setup, cleanup, grader,
budget or usage-accounting failure; quality assertions alone never trigger it.
In-flight work receives cooperative cancellation. Every scheduled slot retains a
trial record, including explicit infrastructure_stop for undispatched slots.
Setup retries finish before applying final-attempt stop policy; cleanup failure
prevents retrying against a possibly contaminated environment.

ScenarioStep.Step receives ScenarioContext {Seed, Mode, Step}; Step is zero based.
Driver Revision and this execution context are provenance. The scenario wire format
is revision 2, and restoring revision 1 is unsupported. Executing replay mode is a
new execution, not restoration of an old effect. Seeds do not guarantee provider
determinism. Callback cancellation remains cooperative.
The scenario retains the latest successfully encoded state with StateStep naming
the number of performed steps represented by that state. Initial state is encoded
before dispatch. A post-step codec failure retains previous state and already
encoded outputs, marks codec_error, and never represents an earlier snapshot as
the final performed state. Completed scenarios require StateStep == Steps.

## Verification requirements

Task 01 has seven normative implementation requirements and nine acceptance rows;
independent acceptance derives its full denominator from the entire task, including
subclauses and common README invariants. Tests cover preflight with zero callbacks,
partial budget/deadline/reconciliation outcomes, mutating graders, pair cancellation
and usage, linear decode counts, benchmark allocations, lifecycle stop policies,
scenario context and wire restore. Historical PASS is not evidence of this state.
