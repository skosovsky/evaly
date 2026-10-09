# Opt-in live integration

Status: **not run**. No host agent/judge credentials were supplied for the current
acceptance. Local HTTP and scripted suites do not constitute a real LLM evaluation.
No paid external call is launched by examples, ordinary/integration/e2e Make targets or CI.

The host can connect a real agent as `Target[I,O,E]` and a real judge through
`Grader`/`PairJudge`, with its own codecs, SDKs, credentials and isolated lifecycle.
The SDK and secret configuration belong to the consumer, outside core. Supply an
explicit budget with reservation/claim accounting before dispatch; retain unknown
usage liabilities. Enable live dispatch only through an explicit host opt-in.

For a reproducible integration record, retain the sealed dataset and case revisions,
input/reference codec identities, target and model/prompt/tool versions, fixture and
reset revisions, parameters and seeds, capture/projection policy revisions, grader
implementation/rubric/model revisions, planned reservations, known actual usage and
reconciliation failures. Persist permitted evidence, target/cleanup statuses,
assessment lineage and comparison objective/gate. Host policy classifies and removes
secrets and determines retention; do not commit credentials or raw confidential
payloads. Missing or unknown provider versions remain explicit.

Run controlled cases through both a native host adapter and the HTTP adapter when
both exist. Inspect actual external state, retained evidence after an effect followed
by error, delivery gaps, lifecycle isolation, cancellation and budget stop. Reopen
artifacts and re-score with a changed judge revision using saved permitted views;
verify no target/tool dispatch and no mutation of the parent. Record the exact host
command, adapter/provider versions, time window, sample size, per-case outcomes,
coverage, costs and unresolved failures in a separate validation entry.

A successful live run demonstrates that those particular adapters and versions
honor the tested port behavior. Judge accuracy requires host-labelled calibration
with declared denominators and missingness; passing scripted or live port tests does
not certify model quality, representative production performance, injection immunity
or durability of a remote filesystem. The host owns deployment, traffic selection,
retention and any decision based on measured quality.
