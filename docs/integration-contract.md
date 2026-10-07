# Integration contract

Task 05 authority: `.cursor/task/05-integration-and-acceptance.md`. This contract
composes existing typed ports; it adds no agent runtime or production scheduler.

The concrete host fixture is specified in [workflow contract](workflow-contract.md).

## Host reference fixture

The host owns typed workflow input, output, state and an external outcome store.
Each lifecycle handle has an isolated namespace; reset establishes a clean state
and cleanup removes it. A common host invocation performs lookup and a refund or
credit action, records tool events and snapshots the actual external outcome. A
confident answer without the action fails outcome grading. A permitted credit path
may pass without matching a single exact tool sequence. A controlled lookup error
is distinct from a failure after a committed refund.

The in-process target and local HTTP handler execute this same host invocation.
Tests compare statuses, usage, retained action/outcome evidence and grades rather
than timestamps, transport metadata or hashes from distinct executions. Target
failure retains paid usage and delivered effects. Undelivered evidence is marked
incomplete; absence assertions cannot pass from a disconnected stream. Lifecycle
isolation, budget exhaustion and cancellation are checked with observable dispatch
counts and state, not inferred from response text.

## Artifact and assessment lineage

Publish experiments and permitted saved views, close/reopen the filesystem store,
then restore with declared host codecs. Offline re-score receives saved views and
grader ports only: no target, tools or lifecycle capability is supplied. A grader
revision change creates a new assessment with explicit parent revision; the old
experiment and its verdict remain immutable. Online partial assessment records
completed work, usage, planned/skipped graders and reasons. Baseline/candidate
comparison receives an explicit objective and gate.

## Failure conformance and CLI

Failure conformance is parameterized by host factories that produce partial work
and an error. Suites must inspect delivered usage/evidence/results and cancellation
without requiring this fixture's domain types. Target, grader/pair judge, evidence,
export and proposal each have executable reference cases.

The [CLI contract](cli-contract.md) specifies the policy JSON and flags.
CLI policy is explicit, versioned and limited to supported assertion or numeric
reference objectives. Arbitrary Go callbacks cannot be serialized into executable
JSON policy. Unsupported objectives and versions fail before comparison. Reports
retain trial/case/revision/seed and host metadata, and must not invent calculation
fixture replay commands for artifacts produced by another target. Exit codes are
0 pass, 1 quality fail, 2 inconclusive and 3 invalid/infrastructure/usage.

## Limits

All examples run without credentials using scripted local ports. They establish
behavior of the infrastructure contract, not real model quality, injection immunity,
exactly-once external effects or production durability. Live testing is separately
opt-in and documented in [live integration](live-integration.md).

## Optional streaming recipes

[Composition contract](composition-contract.md) defines completed streaming targets,
separate trusted judge instructions, explicit accounting conversion, conservative
capture and per-result export. The executable consumer lives in
`integrations/recipes`; core does not import its dependencies.
