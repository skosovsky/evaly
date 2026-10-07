# evaly

Independent evaluation for typed Go targets. Cases, outputs, references and fixture
handles are your types; evaly owns immutable snapshots, lifecycle and measurement.
Core has no third-party dependencies, network calls, API keys or harness requirement.

Execution revision 2 validates configuration before dispatch and reads only the
selected immutable case for each attempt. `Assess` requires a fresh typed view
factory; `SavedView.View` is a reference implementation. Pair judges receive
codec-sealed snapshots, with independent values for both orders and explicit
dispatch/usage records. Assessments retain planned graders, partial results and
skipped reasons even after budget, deadline or accounting failure. Scenario drivers
receive seed, mode and step directly. See the normative
[execution contract](docs/execution-contract.md). Experiment, assessment,
observation-result formats use revision 3; search uses revision 6; scenario uses revision 2; comparison uses revision 3
and comparison-policy uses revision 2. Old formats are unsupported.

HTTP request/response revision 3 declares evidence delivery completeness separately
from target success and preserves delivered events on target failure. Wire readers
use `DecodeWire[T]` for published service envelopes, followed by semantic validation;
host domain codecs stay independent. References reject malformed syntax, fragments,
userinfo and recognized credential query keys, while host policy owns retention and
domain classification. Export reports invalid/conflict/unsupported/cancelled/delivery
failure without selecting retries. See [wire contracts](docs/wire-contract.md),
[evidence and export](docs/evidence-contract.md), and [HTTP protocol](docs/http-protocol.md).

```go
codec := evaly.JSONCodec[int]{ID: "integer", Version: "1"}
dataset, err := (evaly.DatasetDraft[int, int]{
    Selection: "all",
    Cases: []evaly.Case[int, int]{{ID: "one", Input: 1}},
}).Seal(codec, codec)
```

Handle `err`, then provide a `Target[I,O,E]`, `Lifecycle[E]`, output codec,
versioned capture policy, typed grading projection and graders to `evaly.Run`.
[Calculation](examples/calculation/main.go) shows a complete paired run;
[CRM](examples/crm/main.go) checks a separate outcome store despite confident text.
[Protocols](examples/protocols/main.go) demonstrates generated drafts, bounded
scenarios and blind pair order checking. [HTTP](examples/http/main.go) runs a local
JSON bridge. [Observations](examples/observation/main.go) grades saved evidence;
[Optimizer](examples/optimizer/main.go) evaluates immutable typed recipes with
training/calibration/holdout splits and demonstrates a separate resource-plan
candidate domain through the same bounded rounds protocol.
[Integration](examples/integration/main.go) composes a host-owned multi-action
workflow in process and over local HTTP, retained effects after failure, saved-view
reopen/re-score and explicit comparison. See [integration](docs/integration-contract.md)
and [workflow fixture](docs/workflow-contract.md).

Optimizer requires revisioned proposer, typed constraints and objective. Static
enumeration follows the same accounting and stop path. `BestMeasured` records
quality; `Winner` additionally requires complete measurement, host feasibility
and a passing gate. Feedback contains lineage and service summaries; references
require explicit projection. Holdout runs after winner selection and never feeds
another proposal round. Host split validation checks declared group/content keys;
ID disjointness alone does not establish independence. Search v6 supports validated
round-trip restoration without invoking proposal or evaluation callbacks. See
[bounded search](docs/search-contract.md).

```sh
go run ./examples/calculation
go run ./examples/crm
go run ./examples/protocols
go run ./examples/http
go run ./examples/observation
go run ./examples/optimizer
go run ./examples/integration

go run ./cmd/evaly fixture --store /tmp/evaly --id baseline
go run ./cmd/evaly fixture --store /tmp/evaly --id candidate --behavior bad
# Save the explicit JSON policy from docs/cli-contract.md as policy.json.
go build -o /tmp/evaly-cli ./cmd/evaly
/tmp/evaly-cli compare --store /tmp/evaly --baseline baseline --candidate candidate --policy policy.json
# Exit 1 indicates regression for the supplied policy.
# Replay one fixture case with a new trial identity/environment:
/tmp/evaly-cli fixture --store /tmp/evaly --id replay --case case-4 --behavior bad
```

CLI compare accepts a versioned policy with an assertion or numeric reference
objective. It rejects unsupported objective callbacks; use the library API for
host-defined Go objectives. Reports provide trial/case/revision/seed and host
metadata. They do not invent fixture replay commands for arbitrary targets.
See [CLI contract](docs/cli-contract.md).

CLI compare exits 0 pass, 1 quality fail, 2 insufficient coverage/inconclusive,
3 invalid comparison, infrastructure or usage error. Fixture exits 0 when the
record is successfully persisted, even if a quality assertion fails; run compare
to apply the gate. Artifacts are source of truth; export delivery cannot change
verdict. Reuse an artifact ID only for byte-identical publication. A new evaluation
uses a new ID; it can perform external effects again.

Every case/repeat/attempt is retained. Failed reset blocks target dispatch; cleanup
has a fresh bounded context and an independent status. Missing/judge-error evidence
is never zero or success. Repeats average within cases; coverage denominators retain
setup failures. Comparisons name a paired case bootstrap and its assumptions;
thresholds are product policy, not proof of significance. Scores retain declared
units, scales and direction. `Compare` requires an explicit `Objective`; assertion
and numeric reference objectives use the same measurement path. Both side coverage
and matched case count/coverage must satisfy the host gate. Delta, quality and
bootstrap use the same matched cases. Missing means are explicitly unavailable.
The bootstrap method is `paired_case_bootstrap_percentile_pcg_v2`: a local
`math/rand/v2` PCG receives the signed seed's full two's-complement bit pattern as
its first seed word, with zero as the second word. The 95% percentile interval
uses floor(0.025 × (samples−1)) and ceil(0.975 × (samples−1)). This revision changes
bootstrap samples; the paired scheduling algorithm and its seed mapping remain
`case-repeat-v1` SplitMix64.

See [measurement](docs/measurement-contract.md), [paired execution](docs/paired-contract.md),
and [calibration](docs/calibration-contract.md). Case IDs do not prove independence;
the host owns content, group and time splits.

`StopOnInfrastructure` stops future target dispatch on setup, cleanup, grading,
budget or usage-accounting failure; assertion failure alone does not stop a run.
Already active callbacks receive cooperative cancellation, and undispatched trials
retain an explicit infrastructure-stop record. Cleanup failure prevents setup retry.

Capture defaults to dropping payloads and references. Explicit field allowlists
require host classification, including any nested data. Grade projections must
sanitize domain input/output/reference before judges. Raw target output is not
persisted. Preserve evidence and policy revisions if replay matters; retention or
removal can make replay unavailable. Sampled telemetry cannot prove absence of tools.

Ports must cooperate with contexts. Go cannot forcibly stop uncooperative external
callbacks; the library bounds worker count/queue and never hides this limitation
behind detached work. Memory budget and holdout ledger are process-local reference
adapters; host durable adapters must provide atomic guarantees. Unknown actual usage
retains reservations. Atomic Claim allows one dispatch per reservation; reusing
a spent ID cannot pay for another call. No exactly-once external effects, multi-host filesystem
atomicity, global optimum or real LLM robustness claim.

Filesystem publication supports local Linux/macOS filesystems with atomic hard
links and fsync, same-device staging, checksum/reopen verification. Network mounts,
Windows and cloud-sync durability semantics are unverified. Host owns access,
provisioning, credentials, retention and scheduling. HTTP adapter guarantees apply
to the supplied JSON protocol, not any vendor SDK. Optional packages live in the
same module because they introduce no third-party dependencies; core never imports
them. Consumers choose what to import.

Development and CI use Go 1.27.1 and golangci-lint v2.14.0. Both modules
declare Go 1.27.1.

```sh
make validate # config verification, formatting, vet, uncapped lint, race tests and seven examples
make lint     # template-based golangci-lint checks in both modules
make fmt      # apply configured formatters to both modules
make bench    # benchmarks and allocations in both modules
make cover    # coverage in both modules
make fuzz     # each discovered fuzz function runs separately
```

The shared [conformance suite](conformance/conformance.go) can validate host ports.
Tests follow Arrange–Act–Assert, include privacy/adversarial contexts, failed reset,
unknown usage, partial/corrupt artifacts, fake clocks and CLI subprocesses. The
[acceptance matrix](docs/acceptance.md) maps the complete task to evidence.
[Design](docs/design.md) defines API/state/error, revision and statistical policies;
[wire schemas](schemas) version portable envelopes. The isolated contracttest
module executes them with a pinned independent JSON Schema validator; runtime
core dependencies remain empty. SavedViewRecord supports offline re-score after
filesystem reopen using explicit consumer codecs. Breaking domain or wire changes
create new revisions with explicit parent lineage; unknown required major versions
are rejected. Prior affected major formats are unsupported; no migration layer or compatibility
reader is supplied. The format inventory is in [acceptance](docs/acceptance.md).
[Live integration](docs/live-integration.md) is separately opt-in and has not been
run without host credentials. Scripted checks do not establish LLM accuracy.

Release targets validate both modules before invoking the shared script:

```sh
make release-patch # increment patch
make release-break # increment minor before v1, major from v1 onward
make release       # alias for release-break
```

The script requires a branch with clean tracked files and asks for confirmation.
It calculates the version from root tags on the push remote, prepares the selected
commit in a temporary repository, and publishes only the new root tag with an
atomic push. Source HEAD, index, worktree and local tags stay unchanged; untracked
files are never copied. Only root `go.mod` may enter a generated release commit.
Contracttest is a development module and receives no release tag.
See [release safety and recovery](docs/release.md) for failure handling and local
fixtures. Commit identity/signing use the resolved source Git configuration.

Optimizer candidate records use schema v2 (`ParentRevision`); search v6 includes concrete provenance and bounded received counts. See [migration](docs/optimizer-remediation.md).


For the current API, use the [runnable quickstart](docs/quickstart.md),
[authoritative migration guide](docs/migration.md) and
[ownership/concurrency matrix](docs/concurrency.md). Remediation decisions and
executed checks are recorded in [the plan](docs/remediation-plan.md) and
[verification record](docs/remediation-verification.md).

Optional [streaming composition recipes](integrations/recipes/README.md) live in
an isolated consumer module. They demonstrate final stream metadata, partial
billed errors, explicit accounting units, conservative source capture and separate
assertion/metric export through real in-memory SDK evaluation spans. Run
`make consumer-test` for pinned published dependencies or
`make consumer-source CONSUMER_SOURCE_ROOT=..` for supplied SDK sources. Core and
its normal validation remain independent of these optional dependencies.
