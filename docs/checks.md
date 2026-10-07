# Local, CI and release gate

The contract is `checks/registry.json`: Go 1.27.1, golangci-lint v2.14.0,
module and release inventories, lane IDs/commands/profiles/tags/prerequisites,
supported and incompatible peer commit IDs. `make check-plan` prints JSON before
execution. No remote Makefiles or runtime library dependencies are used.

- `make test-fast`: fresh race tests in root and contracttest only. Omitted lanes
  are explicit SKIP in JSON. This is a reduced profile, never a full gate.
- `make test`: required Go, script, published/source/candidate/unsupported consumer
  semantics and the matching Linux test profile. Offline recipes use real SDKs
  with deterministic transports, no live LLM.
- `make lint`: config validation, formatting diff and pinned uncapped lint for
  all three modules. The recipes config is a byte-identical copy of the shared
  baseline; drift fails prerequisites. Root excludes nested module paths from
  its own traversal because each module is checked independently.
- `make check` (`validate` alias): lint, vet, fresh race tests, all script fixtures,
  schema byte/inventory drift, seven examples, four consumers and full Linux gate.

Go flags, user GOENV, go.work, private proxy/sum bypass settings are disabled.
The pinned linter is invoked with `go run`, independent of installed linter/PATH.
Caches live under `/tmp/evaly-*`; test commands always use `-count=1` and a 30-minute process timeout for the
large schema-mutation suite under race. This process bound does not change
assertions or deadline semantics in fixtures. Ordinary optimizer semantic fixtures
use a 1-minute search budget; a separate configured-timeout test blocks on
ctx.Done() and checks deadline settlement/dispatch prohibition.
Source/candidate peers are fetched into fresh disposable Git repositories from
exact registry SHAs, checked against fetched HEAD; neighboring dirty trees are
not used. Published baseline remains distinct from the new candidate lane.

All independent lanes run even after a failure. Dependency/prerequisite failure
is BLOCKED, actual executed failure is FAIL. Required SKIP is never success.
The runner writes a JSON summary and per-lane logs in a printed temporary evidence
path (or `EVALY_CHECK_REPORT` / `--report`). Each row records command, cwd, status,
duration and log; metadata records commit, tracked-byte fingerprint, module
inventory and pins. Actual resolved peer SHAs and module graphs/checksums are in
consumer logs. Linux child summary is embedded in its lane log, and its input
fingerprint must match the host input. The container makes a synthetic Git commit
from those copied bytes; its commit message names the real source SHA. Parent
metadata records that real SHA, child metadata records the synthetic SHA and
matching byte fingerprint. Reports are outside the candidate tree.

## Linux reproduction

Install Docker with a working daemon and allow access to the pinned image,
Debian package repositories, GitHub exact peer refs and the public Go proxy and
checksum service. `make check` on macOS automatically builds `checks/Dockerfile`
(Go 1.27.1 bookworm + Python), copies only tracked input bytes, mounts input
read-only, and runs the same runner/profile inside Linux. No host sibling
checkouts, Go workspace or host Go caches are mounted. A missing Docker binary or
unavailable daemon is BLOCKED and makes the full gate fail.

On Linux CI the equivalent command is:

```sh
python3 scripts/checks.py check --linux-native --report /tmp/evaly-check/summary.json
```

`--linux-native` is rejected outside Linux. The Linux lane denotes that native
execution, without recursion. CI chooses OS, installs Go from go.mod and uploads
reports; commands, refs and module coverage come from the registry. Root `v*`
tag pushes trigger this gate once; test-only nested modules have no release tags.

## Boundaries and negatives

Live-provider/manual/performance and historical baseline repro commands are
classified separately in registry.manual. They do not prove hermetic gate success
or LLM quality. Required consumers are not optional.

AAA negative fixtures exercise real failing Go tests (including inherited GOFLAGS
attempting to suppress them), pinned linter config verification, Python script
tests, missing peer consumer input, aggregate independent failures and BLOCKED
dependencies. Production missing-Docker, wrong-Go, incompatible-peer and unexpected
infrastructure responses are also covered at subprocess boundaries. Release
fixtures use local bare origins only; gate failure and tracked/ignored mutation
must produce no published tags and preserve source state. Public-verifier fixtures
reject mismatched checksums/version/origin, consumer failure, missing/failed/stale
CI and timeouts. These fixtures test enforcement and do not replace mandatory
real Linux/consumer/public verification.
