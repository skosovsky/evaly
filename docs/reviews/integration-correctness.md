# Task 05 independent correctness acceptance

Date: 2026-10-05. Reviewer: `/root/accept_integration_correctness`, independent of all task 05 implementers. Authority: `.cursor/task/05-integration-and-acceptance.md` and shared `.cursor/task/README.md`.

Verdict: **PASS**. No identified correctness defect remains open on the reviewed source. This is acceptance of the tested contract, not a claim that untested defects cannot exist.

Frozen fingerprint: `2e4897bfda0bc1778cdf6246fa45c5606e98fb6dd2e6172fc8cd0c28996b26e1` (134 source/authority files). Computed with `/private/tmp/evaly-review-state.py`; review reports and the validation journal are excluded. Rechecked unchanged after execution. The completeness reviewer must accept this same fingerprint before commit.

## Reviewed behavior

The CLI requires an explicit versioned comparison policy, resolves only the supported assertion/numeric objectives, preserves native numeric units and direction, and rejects callback-specific selection or missingness. The extracted gate validation preserves Compare's previous constraints. Store loading is preceded by policy decoding and semantic resolution; no target execution or arbitrary callback deserialization is introduced. Host reports retain case/trial/revision/seed, target, fixture/reset and evidence coordinates without manufacturing calculation replay commands. Subprocess cases cover pass/fail/inconclusive/invalid classes, caller thresholds, unsupported semantics and a foreign target with numeric lower-is-better measurement.

The host workflow shares its tool implementation across in-process and HTTP execution. Tests verify actual retained external balances and isolated namespaces, confident text without effect, allowed credit/refund alternatives, controlled tool failure, committed action followed by error, delivery interruption, unknown usage on transport disconnect, and zero dispatch after cancellation or exhausted budget. Partial failed-run evidence is assessed explicitly with an empty unknown output; completed saved views capture the actual host Project output. The accepted Runner is not altered to fabricate successful grades or failed-target outputs.

Publication/reopen and offline re-score preserve immutable experiment records and assessment parent/source revisions without target/tool/lifecycle capabilities. Online partial assessment preserves paid work and marks skipped grading. Generic conformance fault factories cover target, grader/pair judge, evidence, export and proposal without imposing workflow types. Export verifies source immutability despite a mutating post-effect failing sink; paid grader and reversed pair failures retain usage while dropping invalid judgment content.

## Independent execution

Toolchain: `go version go1.27.1 darwin/arm64`; `GOCACHE=/private/tmp/evaly-final-1.27.1-cache`.

- `make validate`: **exit 0**; both modules' formatting, vet and race suites passed. Independent generated-schema drift/inventory and structural JSON corpus passed. Contracttest race suite took 92.938 seconds. Log: `/private/tmp/evaly-integration-correctness-validation.log`.
- `go run ./examples/calculation`, `crm`, `protocols`, `http`, `observation`, `optimizer`, `integration`: each **exit 0**. Logs: `/private/tmp/evaly-integration-correctness-<example>.log`.
- External module `/private/tmp/evaly-integration-correctness-probes`, `go test -race ./...`: **exit 0**. Independently probes valid assertion-any and native lower numeric descriptors; rejection of duplicate, unknown, missing, null and old-version policy fields; forged assertion metric semantics; and unsupported custom eligibility. This probe module is outside the repository and does not modify implementation.
- `git diff --check`: **exit 0**; source fingerprint unchanged.

## Scope and limitations

All execution above is offline/scripted, including local HTTP. Live host agent/judge validation was **not run**; no credentials or paid external calls were used. The review does not establish LLM quality, injection immunity, production durability, exactly-once external effects, statistical independence from case IDs, or forced termination of uncooperative callbacks. Host codecs, domain policies, credential handling, durable accounting and deployment remain outside core ownership. Historical acceptance reports are explicitly distinguished from current task evidence.
