# Task 04 — independent correctness acceptance

Verdict: **PASS**. No identified unclosed correctness defects remain in the reviewed task scope.

Reviewer: `accept_search_correctness`, independent of implementation. Review included the complete task 04, common task rules, search contract, optimizer source, affected conformance/schema code, tests and examples. The reviewer changed only this report; independent adversarial probes were kept outside the repository.

Frozen source fingerprint: `08eba985354d909f1ac16e7bc9012618d604ee897f41e00c46f3524dfb0a802b` (119 files, computed by `/private/tmp/evaly-review-state.py`). The fingerprint was independently verified before and after final verification. Review reports and the validation journal are excluded from this fingerprint.

## Reproduced defects and closure

- A duplicate candidate rejection retained the same candidate revision as an earlier evaluation, but restore rejected the legitimate audit history. An independent producer-to-restore probe now passes while preserving the rejection and single measured ranking entry.
- A successful budget Claim could cancel the context and still permit a subsequent proposer dispatch. An independent cancelling budget probe now observes zero proposer calls, with a stopped result; the shared authorization boundary covers evaluation and holdout dispatches too.
- Callback service views initially shared persisted score/experiment/comparison state. Independent mutating constraint and feedback projector probes now leave persisted quality and aggregate unchanged. Official tests additionally preserve sealed experiment state and gate verdict.
- Five correctly rehashed semantic falseclaims were accepted by restore: a score without comparison, omitted BestMeasured despite measured ranking, a candidate limit below retained history, a stopped round inside a completed result, and a holdout comparison pointing to calibration. Independent probes now reject all five. Structural wire validation also enforces persisted numeric limits and usage bounds.
- Early codec preflight inspection identified unsafe typed-nil handling. The final independent Seal probe returns `ErrInvalid` without a panic.
- Two preliminary frozen trees failed the configured formatter check. These were rejected. The accepted fingerprint passes the real `make validate` formatter check; no checks were weakened.

## Final verification on the accepted fingerprint

- `env GOCACHE=/private/tmp/evaly-final-1.27.1-cache make validate`: terminal exit 0. Includes configured formatting, vet and race tests in the root and nested `contracttest` modules. Independent log: `/private/tmp/evaly-search-correctness-final-validation.log`; nested contract tests completed in 92.396 seconds.
- All six examples (`calculation`, `crm`, `protocols`, `http`, `observation`, `optimizer`) were independently rerun after the final freeze and returned exit 0. Logs: `/private/tmp/evaly-search-correctness-example-<name>.log`. The optimizer example completes two feedback rounds for both `recipe` and `resource-plan` candidate domains.
- Independent temporary review module: `go test -count=1 -v ./...`, exit 0, covering nine adversarial cases including callback mutation, cancellation after Claim, duplicate-history restore and the five rehashed falseclaims. Log: `/private/tmp/evaly-search-correctness-probes.log`.
- `go run /private/tmp/evaly-candidate-preflight-review.go`: `err=evaly: invalid contract`, `panic=<nil>`.
- `git diff --check`: exit 0.

Code and executable regressions establish unified static/proposer accounting, deterministic revision ties, lower-is-better ranking, separate measured-best and feasible winner, feedback-driven repair, bounded rounds/candidates/deadlines/budget, retained partial usage and lineage, replay-resistant claims, fixed selection before holdout, absence of holdout inputs in proposal rounds, revision drift rejection, host-controlled group/content split validation and detached typed candidate/record restore.

This verdict is scoped to identified defects and observed checks, not a proof that arbitrary host callbacks or codecs are safe. The library does not infer semantic independence, forcibly interrupt callbacks, replace a durable host ledger, redact arbitrary domain secrets or implement an agent/runtime optimizer algorithm. These remain explicit host boundaries in the accepted contract.
