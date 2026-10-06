# Host adapter failure conformance

The `conformance` package is a test dependency. Its generic factories receive fault modes and return fresh host adapters, typed inputs and explicit expected observations. The suites contain no calculation, CRM, message or agent model. Factories must inject the stated fault; host verification checks actual effects rather than trusting a target's text.

`TargetFailures` runs `AfterEffect` and `IncompleteDelivery` through `evaly.Run`. The first requires a target-error trial, retained events and the declared usage after an effect. The second requires incomplete evidence: retained events cannot prove absence of an omitted action. Both require successful cleanup and mandatory host verification. A host may run the same factory contract against in-process and HTTP adapters; timestamps and transport metadata are not compared.

`GraderFailures` requires a dispatched callback returning a partial grade plus an error: usage survives, assertions and metrics do not become scores. `PairJudgeFailures` requires a successful forward judgment and a failed reverse judgment with usage; disagreement cannot be certified from one reviewed order. `EvidenceFailures` checks that a recorded event survives a delivery fault, sealing reports incompleteness, and a cancelled write does not enter the retained record. `ExportFailures` requires a sink that commits an effect and then reports delivery failure: the wrapper reports failure without altering the source or promising exactly once. `ProposalFailures` requires a proposer that returns partial candidates and usage plus an error; these remain observable to the caller, and pre-cancellation returns cancellation.

All factories have explicit expected usage/counts and, where effects belong to the host, a verification callback. Reference adapters and adversarial cases are executable in `conformance/failures_test.go`. These checks verify adapter contracts and failure accounting, not LLM quality, production durability or injection immunity.


Lifecycle conformance owns each handle once Prepare returns, including a partial
handle returned with an error. It immediately registers test cleanup before
checking Prepare/Reset errors. Fatal and Goexit after that point still run cleanup
once. Cleanup has its own detached context and deadline; its error fails the test
without replacing the primary Prepare/Reset failure. A callback that exits inside
Prepare before returning has provided no handle to the helper; the host owns
recovery for that boundary. Cleanup is an attempt, not proof of rollback (D04/F10).

`Lifecycle` uses explicit local-test defaults: ID `conformance-lifecycle`, one
second for cooperative Prepare/Reset work, and one second for cleanup.
`LifecycleWithOptions` lets an external fixture supply ID, Timeout and
CleanupTimeout. All must be nonempty/positive. The options are test configuration,
not serialized runtime policy. Cleanup runs at test exit, not immediately when the
helper returns. Hosts that need an earlier lifetime should call the suite in an
isolated subtest.

D57: factories must own independent external namespaces as well as fresh adapter
objects. A factory returning a new object over the same remote state does not
isolate hardcoded conformance IDs. Artifact, target, export and protocol suites
still use deterministic IDs; hosts must prefix/map them within fresh namespaces
and dispose those namespaces through their fixture lifecycle. Budget factories now
run once per protocol subtest (seven fresh adapters), rather than once for the
whole suite. They must provision exactly two units each; the host controls remote
namespace creation/disposal and bounded network calls.

D51: Budget verifies atomic capacity/unknown liability, reservation lookup and
changed-unit conflict, serial zero-unit claim-once and concurrent claim-once,
release idempotency before claim, conflict after claim/settlement, known settlement
idempotency/conflict, available remaining capacity and pre-cancellation of every
operation. The no-op Claim adversarial adapter is rejected in a subprocess.
Stock MemoryBudget passes; this strengthens adapter certification rather than
changing its accounting contract.

D58: Target and Grader cancellation cases exercise contexts cancelled before Run
or Assess dispatch. They verify wrapper cancellation classification, not in-flight
cooperation of a target or judge. A callback must observe the context itself;
none of these tests certifies hard interruption, rollback, or an external effect
that completes after cancellation. Lifecycle operation/cleanup deadlines are also
cooperative; no detached callback goroutine manufactures a hard timeout.

D59: the shared structural corpus creates minimum/maximum neighbours as exact
JSON numeric tokens using decimal rational arithmetic. Integer limits, fractional
and exponent bounds are not converted to float64 before ±1. Dedicated tests cover
MaxInt64, MinInt64, MaxUint64, ±MaxFloat64 decimal tokens, fractions and very small
exponents, asserting an exact one-unit difference. The independent Draft 2020-12
engine/runtime parity corpus is structural evidence; separately authored rehashed
semantic negatives remain necessary and unchanged. Parity alone does not establish
semantic validity.

Executable F10 coverage: successful/partial Prepare, failed Reset, Goexit,
cleanup-only and concurrent primary/cleanup failures, cooperative cleanup deadline.
Each child records exactly one correctly owned handle and an uncancelled cleanup
context with a deadline. The baseline script separately reproduces the original
Reset leak and D51 no-op Claim acceptance on 76c224a, without real infrastructure.
