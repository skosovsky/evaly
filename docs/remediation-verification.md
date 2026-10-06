# Remediation verification

Baseline: `76c224a97e5f03a44e87ab7ea308216a7e10d139`.
This is a durable record of task acceptance, not a claim of live provider quality,
distributed durability or exhaustive absence of bugs.

For each task record: frozen reviewed diff/base, mandatory criterion count and
completeness percentage, independent reviewer reports, commands/outcomes/limits,
and commit mapping. A commit can be located by its unique subject below; later
records also record its SHA. Pending tasks have no acceptance claim.

## T01 — accepted

Changes: ordered checklist with complete F/D ownership; numeric/JSON/preflight/
optimizer target contracts. Implementation behavior is unchanged.

Checks: source task, current budget/codec/preflight/restore/search/release/Makefile
read; clean starting worktree on main. Documentation-only task; runtime tests are
not evidence of target-contract compliance and are deferred to implementation.

Initial independent review: completeness 2/3 (66.67%), missing D22 ownership;
correctness identified the same gap, missing accepted candidate ID uniqueness and
unspecified proposal usage overflow policy. Corrected by assigning D22 to T09,
requiring unique accepted IDs/revisions, and specifying exact rational aggregate
with upward finite reporting and ErrInvalid on overflow. Both reviewers must
recheck this corrected version.

Final independent acceptance: `/root/t01_completeness` — 3/3, 100%;
`/root/t01_correctness` — no errors found in the reviewed specification scope.
Both reviewed plan SHA256 `1e1a971c3c692f36530a9d76252be1e2b14534ac0112c0ac6f030935b23c84ed`
and contracts SHA256 `49932f6f010193183bffdca241f2e5dd69ee83fbd7b81c0f24d41c340ea06dbb`.
After review only acceptance metadata and the T01 checkbox were updated.
Whitespace checks on both new documents produced no diagnostics (`git diff
--no-index --check /dev/null PATH`; exit 1 indicates added content).
Commit subject: `docs: define remediation contracts`.

## T02 — accepted

Previous accepted task: T01 commit `e7ba3c2`.
Changes: isolated root release repository, exact go.mod staging and tag refspec,
remote version selection, atomic push, explicit source/ref/recovery policies;
local AAA release fixtures and release documentation.
Initial review: completeness 3/3, 100%; correctness reproduced automatic global
tag signing creating an annotated ref despite the lightweight contract. Corrected
with explicit `git tag --no-sign` and a global-tag-signing regression fixture.
Final independent acceptance: `/root/t02_completeness` — 3/3, 100%;
`/root/t02_correctness` — no errors found, signing finding closed.
Reviewed script SHA256 `5c5e8cf89a3207fec25ca42a5104f2e420f23ecae08afe0dd55a86d90d99bd54`;
fixtures SHA256 `5c420db9787ea5b63d2b794bc1bb8f015066bf3bbc575a28231a38bcb3209d57`.
Commands: `python3 scripts/release_test.py` — 10/10 PASS;
`python3 scripts/release_test.py --baseline` — F01/F02 behavioral repro PASS,
8 new-contract tests skipped; `bash -n scripts/release.sh`, `git diff --check` — PASS.
Platform: Go 1.27.1, Git 2.55.0, Bash 5.3.15, Python 3.14.6, macOS arm64.
Limits: local bare remotes, no real publication; Linux and live network/commit
signing backends not executed. Only acceptance metadata changed after review.
Commit subject: `fix: isolate release publishing`.

F01/F02 fixed; source checkout and refs preserved by isolated preparation, exact
staging and atomic root refspec. Portability decision: portable Go editing instead
of BSD sed; Linux execution remains unverified.

## T03 — accepted

Previous accepted task: T02 commit `4b0bdb8`.
Changes: MemoryBudget exact binary rational liability, conservative float reporting,
AAA numeric/claim/concurrent regressions and settlement fuzz target; durable baseline
Go overlay repro and budget semantics/migration docs. D02/D03/D46 preserve explicit
host liability with reasons in `docs/budget.md`.
Final independent acceptance: `/root/t03_completeness` — 3/3, 100%;
`/root/t03_correctness` — no errors found in scope.
Reviewed budget SHA256 `0f085b5b52b4d7ee7e60f0f63c72da3d076802b1994e6a71734ff3405f3eaa70`;
exact tests SHA256 `70a9ee8cf6d4c89328080475bac779f40244323cc262f3a7d79c2f10503ecdad`.
Commands: `go test -race ./...` — PASS (root module; unchanged dependent package
results may be cached); final added budget tests `go test -race -run='TestMemoryBudget|FuzzMemoryBudget' .` — PASS;
pinned golangci-lint 2.14.0 root run — 0 issues; baseline overlay — both repros PASS;
settlement fuzz 10s, parallel 2 — 134,539 executions PASS; independent reviewer fuzz
3s — 83,898 executions PASS; `git diff --check` — PASS.
Limits: finite binary float receipts, process-local adapter; short settlement fuzz
does not prove all possible transition sequences or distributed host recovery.
Public API/wire representation unchanged. Only acceptance metadata changed after
review. Commit subject: `fix: preserve exact budget accounting`.

## T04 — accepted

Previous accepted task: T03 commit `5872faa`.
Changes: generic UseNumber decoding, stock reversibility validation, standard
encoder cycle semantics, typed/extent-aware UTF-8 traversal excluding ignored
fields, number/alias/cycle/ownership regression and fuzz coverage. D23/D24 decisions
and migration are documented in `docs/codecs.md`.
Initial correctness review reproduced invalid UTF-8 bypass for named string map
keys implementing MarshalJSON and false rejection of addressable pointer-receiver
MarshalJSON. Fixed direct key-string validation and addressable custom method-set
recognition, with explicit regression tests; both reviewers must recheck.
Final independent acceptance: `/root/t04_completeness` — 3/3, 100%;
`/root/t04_correctness` — no errors found, both initial findings closed.
Reviewed codec SHA256 `8aaf2922bce5696ed964890c6764aa2cdb366736fa094f6aebbb768098208ad7`;
roundtrip tests `ae90691b9fa153020bd3dc986f9d305eb03ed28cfa0173119093723f73881031`;
fuzz test `4ff1b615060a27d55334555c445b8b052193de669ea9ef4a134576c17eb73160`;
codec docs `b1ed04e41ab0c7b9a74549d6ab57c902d52d7c5ddf38b863dafc61456e3f1e0e`.
Commands: final root `go test -race ./...` — PASS; final focused codec race — PASS;
pinned lint 2.14.0 root run — 0 issues; baseline overlay — F04/F05 PASS;
canonical roundtrip fuzz 10s parallel 2 — 23,341 executions PASS (before final
source-string custom-method fixes); final independent fuzz — 44,083 and 56,801
executions PASS; independent final `go test ./...` — PASS; `git diff --check` — PASS.
Limits: conservative source UTF-8 for shadowed/omitzero fields, stable pure custom
callbacks and caller-owned host buffers required; short generic fuzz does not
certify arbitrary host callbacks. Generic decoded representation changes to
json.Number, without rewriting valid canonical bytes or wire schemas.
Only acceptance metadata changed after review.
Commit subject: `fix: preserve canonical JSON values`.

## T05 — accepted

Previous accepted task: T04 commit `bd8ba96`.
Changes: structural codec preflight for dataset/scenario/saved-view/candidate/search
restore and Save/Load helpers, Absence stock type with private validated config,
finite typed-nil kind detection and corrected GoDoc. D08/D20/D21/D27/D28 decisions
and constructor/error migration recorded in `docs/preflight.md`.
Final independent acceptance: `/root/t05_completeness` — 3/3, 100%;
`/root/t05_correctness` — no errors found in scope.
Commands: final root `go test -race ./...` — PASS; pinned golangci-lint 2.14.0
root run — 0 issues; independent focused preflight/absence/ownership tests — PASS;
`python3 scripts/preflight_baseline_repro.py` — PASS (original F06 typed-nil
panics and F07 empty-pass/matching-event-panic); `git diff --check` — PASS.
Limits: arbitrary host panics, unstable Identity/Validate and callback thread
safety remain host responsibilities. Only acceptance metadata changed after review.
Commit subject: `fix: validate capabilities before dispatch`.

## T06 — accepted

Previous accepted task: T05 commit `dd27a6f`.
Changes: shared audit semantics, binding before canonical assignment, exact proposal
usage, ParentRevision migration, concrete provenance, static proposal preflight,
bounded received counts and checked JSON clones. Decisions D39–D45/D47–D50 are in
`docs/optimizer-remediation.md`. Search v5 and candidate v2 schemas synchronized.
Commit subject: `fix: validate optimizer artifact semantics`.

Initial T06 acceptance: completeness 3/4, 75% (candidate v1 stale inventory);
correctness found rehashed phase/stop contradictions retaining holdout after
candidate/round/proposal stop or ledger failure. Fixed phase/artifact consistency
and limit bounds with explicit mutations; synchronized candidate v2 inventory.
A subsequent lint check required extracting selection readiness and the deadline
literal; no thresholds were relaxed. Both reviewers rechecked the final version.
Final independent acceptance: `/root/t06_completeness` — 4/4, 100%;
`/root/t06_correctness` — no errors found in scope, previous finding closed.
Commands: final root `go test -race ./...` — PASS; contracttest `go test -race
./...` — PASS (98.310s); root and contracttest pinned lint 2.14.0 — 0 issues;
final independent optimizer race — PASS (3.859s/7.645s); independent contracttest
race `-count=1` — PASS (97.839s); baseline F08/F09 — all 5+4 original defects
reproduced; exact usage fuzz 10s — 159,859 executions PASS; final semantic audit
fuzz 10s — 1,156 executions PASS (before semantics-preserving lint extraction);
reviewer semantic fuzz 3s — 408 executions PASS; eight phase mutation regressions
and independent adversarial overlay — PASS; `git diff --check` — PASS.
Reviewed semantics SHA256 `28bd71a2787b4cb441370e95c36750ea7585d8e9a632a28f631e795c4c2be858`;
execution SHA256 `bf8bde4e1d82ce95b3016ae61c05bcf4d00fd53b77bfabe2704574dcd60ecc85`;
search schema `693c46582cac8a305fae7a69c1c7076343019dac2f64237b35a1411e88b210e8`;
candidate schema `0edd387359e8e6264b428e51af63b89b30ec66362864bd4d7c2821bc9ffb9485`.
Limits: short targeted fuzz; provenance is internally bound, not proof of host
truth/independence. Received count cap does not bound host string/description bytes.
Private must-clone failures indicate internal invariant breaches; arbitrary callbacks
remain host responsibility. Only acceptance metadata changed after final reviews.

## T07 — accepted

Previous accepted task: T06 commit `e56e0d8`.
Changes: immediate owned lifecycle cleanup with explicit fixture options/defaults,
expanded atomic budget conformance and no-op Claim rejection, exact decimal schema
boundary corpus, subprocess regressions and original F10/D51 reproduction.
D04/D51/D57–D59 contracts and namespace/cancellation limits are documented in
`docs/conformance-contract.md`.
Final independent acceptance: `/root/t07_completeness` — 3/3, 100%;
`/root/t07_correctness` — no errors found in scope.
Commands: root `go test -race ./...` — PASS; contracttest `go test -race ./...`
— PASS (103.260s); both modules pinned lint 2.14.0 — 0 issues;
independent conformance race — PASS (2.614s/2.713s); independent full contracttest
race `-count=1` — PASS (90.620s); independent exact-boundary/semantic-checksum
race and optimizer rehashed/terminal/restore negatives — PASS;
`python3 scripts/conformance_baseline_repro.py` — PASS F10 Reset leak and D51
no-op Claim acceptance; `git diff --check` — PASS.
Reviewed lifecycle suite SHA256 `78fb09b5066df9ef8973bd5b30900380d578cf457406dc661ce478c62862eff7`;
budget suite `46a4c124e8054e64ced35ae5de27228eaacf009e9daebcb89709efc029a6e2b2`;
corpus `01688907bea2928554ac2715fbf2e85e445f2c54c448ffe56be4723b179a04a5`.
Limits: local subprocess/in-process fixtures; real external namespaces and host
in-flight cooperation are not certified. Cleanup cannot forcibly stop callbacks,
and Prepare Goexit before returning supplies no handle to this helper.
Only acceptance metadata changed after final reviews.
Commit subject: `fix: enforce adapter conformance contracts`.

## T08 — accepted

Previous accepted task: T07 commit `2e81d4d`.
Changes: fallible HTTP NewHandler constructor, cancellation guards at dispatch
boundaries, HTTP v3 without redundant capability flags, per-command CLI argument
validation before store creation. D52–D55 and schema/API migration are documented
in `docs/http-protocol.md`; D56 in `docs/cli-contract.md`.
Final independent acceptance: `/root/t08_completeness` — 3/3, 100%;
`/root/t08_correctness` — no errors found in scope.
Commands: root `go test -race ./...` — PASS; contracttest `go test -race ./...`
— PASS (98.635s); both modules pinned lint 2.14.0 — 0 issues;
`make examples` — all seven PASS; independent uncached HTTP/CLI race — PASS;
independent full contracttest race — PASS (103s); independent schema generation,
generated artifacts and shared structural corpus race — PASS (107.073s);
independent runtime probe confirms constructor-only identities, read failure 400
without Invoke and oversized callback response 500; `git diff --check` — PASS.
Reviewed HTTP implementation SHA256 `daaae188c5342f092ef449ebb4609a081b469743ebc066bd90ab7e2d6ae2ca9f`;
CLI `1553abedc686780c2983e29d9a07e7bdc80d98229164c349b1a2564e7d885a20`;
request schema `03307b9eb3e1524e34a19977df8fc65b04df2fe50dbd12c58c249fac7b1fe450`;
response schema `cd4346d18f61375947304eb2536c212457cf6e4ce850489167e2353930d85471`.
Limits: local HTTP/scripted fixtures; hard cancellation, live provider interoperability
and total memory bound are not certified. Only acceptance metadata changed after reviews.
Commit subject: `fix: validate adapter setup before dispatch`.
