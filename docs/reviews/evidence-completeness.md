# TASK02 independent completeness acceptance

Reviewer: `accept_evidence_completeness`, independent of implementation. Authority:
the entire `.cursor/task/02-evidence-and-wire-contracts.md` and its README.
Implementation files are read only for this review. Final independently computed fingerprint:
`aabce87e5dec89ddeaeaae3592c44f0319d23260e150ec63242f9c63819c27b0`
(93 files). The implementation and checks below use that frozen tree.

The fixed denominator is **55 conjunctive rows**: 35 numbered-requirement rows,
all nine acceptance rows, and 11 common README rows. All clauses in each row are
required; a partially satisfied row receives no credit. No denominator reduction
or deferred requirement credit is permitted.

| ID | Required result (all clauses required) | Evidence and status |
|---|---|---|
| N1a | Distinguish reference syntax from host retention authorization | docs/evidence-contract.md separates bounded syntax from host permission; FieldPolicy projects before retention; complete |
| N1b | Reject ParseQuery errors and userinfo; fragment policy explicit and enforced | safeReference uses url.ParseQuery, rejects userinfo and all literal fragment delimiters; reference table regressions; complete |
| N1c | Reject ambiguous/malformed references | safeReference rejects whitespace/control/invalid percent escapes including opaque URI; independent malformed-opaque reproduction now rejects; complete |
| N1d | Known credential-bearing URL cases rejected | reference table covers semicolon query, fragment token, encoded key, mixed case, signed URI and SAS sig; complete |
| N1e | No claim of arbitrary-secret detection through blacklist | evidence contract explicitly excludes arbitrary secrets in paths, opaque values and unrecognized query names; complete |
| N1f | Safe artifact IDs and non-HTTP/opaque URI support retained | reference table accepts artifact IDs, urn, s3, HTTP and escaped nonfragment path; no HTTP-only restriction; complete |
| N2a | Same reference invariants in capture, restore and grading | same safeReference used by FieldPolicy/Capture.Record, ValidateEvidence and ValidateGrade; cross-path test; complete |
| N2b | Default policy retains no sensitive data | FieldPolicy defaults remove payload/references; TestDefaultPolicyDropsUntrustedReferences; complete |
| N2c | Host policy/projection retain domain classification ownership | docs/evidence-contract.md and design ownership; domain codecs/projection and host nested-value classification retained; complete |
| N2d | Rejection reason/log does not copy rejected secret | ErrInvalid plus fixed capture_error/host_incomplete; test proves private-secret absent from errors; export labels never sink text; complete |
| N3a | HTTP envelope/protocol validated before events trusted | Target.Run DecodeWire plus capability/status/evidence/output contract before usage/event retention; untrusted-response tests; complete |
| N3b | Allowed available events and known usage retained before target_error | Target.Run assigns Usage then records policy events before ErrTarget; complete/incomplete target-failure regression; complete |
| N3c | Target failure distinguishable from unsupported protocol | ErrTarget separate from ErrUnsupported; unsupported wire major classified via wirecontract.ErrVersion; loss checks pre-retention; complete |
| N3d | Partial effects retained without successful output | Invocation carries usage/events on error without output; delivered-prefix regression retains first event and paid usage; complete |
| N4a | Explicit verifiable evidence delivery completeness | HTTP v2 requires explicit EvidenceDelivery Complete/Reason; Validate rejects contradictory or unspecified declaration; complete |
| N4b | Transport/decode/size/timeout loss marks incomplete unless proved complete | transport/status/type/read/size/wire/protocol/retention failures MarkIncomplete; truncation/oversize/malformed/deadline regressions; complete |
| N4c | Proven complete target_error retains complete trace; quality separate | complete target_error remains sealed; output domain decode error preserves proven delivery; independent quality/evidence docs; complete |
| N5a | TargetResult.OutcomeRefs removed without alias | TargetResult consists only of Output and Usage; no OutcomeRefs in source; complete |
| N5b | One policy-controlled outcome reference path; examples/docs updated | design documents sole capture path; CRM fixture emits policy-filtered tool/outcome events; CRM example independently runs; complete |
| N5c | No arbitrary-ref bypass or universal domain outcome model | typed BYOT Output preserved; no added arbitrary refs, outcome model or bypass API; complete |
| N6a | Required presence defined for every supported wire kind | Schema derives required JSON tags; DecodeWire enforces same definitions; shared corpus removes every populated required field in all14 roots; complete |
| N6b | Nullability defined for every supported wire kind | Schema/validator distinguish nonnull scalars/structs from nullable pointers/slices/maps; RawMessage host-owned; null corpus all14 roots; complete |
| N6c | Zero-value rules defined for every supported wire kind | shared contract zeros valid unless rule minimum/const; seed0 positive; zero mutations all14 roots; complete |
| N6d | Enum rules defined for every supported wire kind | wirecontract.constrain centralizes enum rules and runtime enforcement; corpus enum mutations plus independent nested status checks; complete |
| N6e | Major versions defined for every supported wire kind | version constraints shared; HTTP2 replacesHTTP1; prior experiment/assessment/scenario/result/search2 and independent capability1 retained; complete |
| N6f | Limits defined for every supported wire kind | integer storage bounds, finite float bounds, positive plan limits, bounded grades/errors/gaps and explicit transport limits; numeric overflow corpus; complete |
| N6g | Additional-properties policy defined for every supported wire kind | service structs additionalProperties false; maps/extension/domain JSON permitted under documented ownership; unknown corpus all14 roots; complete |
| N6h | Shared corpus checks schema/runtime decode/restore including HTTP | TestSharedStructuralCorpus all14 schema/runtime roots, positive Restore where public, hundreds of nested mutations; HTTP request/response included; complete |
| N6i | Structural-schema versus additional semantic validation boundary documented | docs/wire-contract.md and checksum separation test distinguish structural shape from checksum/state/identity/protocol semantic checks; duplicates byte-level invariant; complete |
| N7a | Required field presence checked before target dispatch | Handler DecodeWire before input decode/invoke; missing/null/duplicate seeds produce zero calls; complete |
| N7b | Explicit seed=0 accepted; missing seed distinct | explicit0 invokes once, missing0 rejected; int decimal/exponent normalization respects integral values; complete |
| N7c | No independent dev validator dependency in core runtime | root go.mod no third-party dependencies; core imports only stdlib plus own internal/wirecontract; independent engine isolated contracttest; complete |
| N8a | Export invalid/conflict/unsupported/cancelled/delivery_failure distinguishable | deliveryReason errors.Is taxonomy; wrapped sentinels/cancel/deadline/temporary/private-error tests; complete |
| N8b | Source quality verdict immutable during delivery/retry | Export detached clone; malicious mutation test verifies source bytes unchanged; dedup retry/conflict test retains immutable artifact checksum; complete |
| N8c | Host retry choice preserved; no core retry/outbox/transport | host-owned retry policy documented; exactly one Deliver per Export; no network/retry/outbox in core; complete |
| A1 | Both credential examples rejected before retention and in restored/grade refs; safe opaque refs accepted | TestReferenceContractAcrossCaptureRestoreAndGrades and defaultdrop test; independent opaque malformed reproduction passes finalfix; complete |
| A2 | HTTP effect then target error retains event/usage and target-failure status | TestTargetFailurePreservesUsageAndPolicyControlledEffects covers effect/usage/ErrTarget for both complete and incomplete delivery; complete |
| A3 | Truncated/oversized/invalid response cannot yield complete empty evidence or prove absence | TestUndeliverableResponseNeverProvesAbsence checks incomplete false toolcoverage and zero untrusted events on all response-loss variants; complete |
| A4 | Invalid/unknown version does not trust events or dispatch server target | old version response rejected before usage/events; handler unknown-version test zero dispatch; nested/corpus version checks; complete |
| A5 | Missing/null seed rejected preinvoke, zero accepted; corpus covers missing/null/zero/unknown/duplicate keys | Handler missing/null/zero/duplicate seed table plus all14 corpus missing/null/zero/unknown/rootduplicate cases; complete |
| A6 | Independent positive/negative contracttest; reproducible schema generation check without working-tree mutation | independent contracttest positive/negative parity and TestSchemaGenerationDoesNotDrift generate disposable directory and compare bytes/inventory; complete |
| A7 | OutcomeRefs and legacy bypass removed; CRM outcome evidence example runs | no source OutcomeRefs; CRM allowed outcome/tool event pipeline and independently successful example; complete |
| A8 | Export conflict versus temporary failure distinct; redelivery cannot change quality verdict | Export conflict/retry test separates temporary failure, dedup delivery and conflicting identity; detached source mutation check; complete |
| A9 | Design, HTTP protocol, schema versions, conformance, restore tests and examples updated | normative design/HTTP/evidence/wire docs, HTTP2 schemas/consumers, Export conformance, Restore positives/semantic corruption and all6 examples; complete |
| C1 | Clear break across all consumers; no aliases/shims/legacy readers/dead fields | old tuple Handler/API field/HTTP1 schema paths removed; no aliases or dual reader; all consumers build; complete |
| C2 | Spec-first normative design covers input/result/state/errors/ownership/identity/cancellation/limits | design links normative contracts authored before implementation; inputs/results/errors/ownership/versions/limits and delivery cancellation described; complete |
| C3 | Changed wire semantics explicitly versioned; current fixtures/schemas match; old version unsupported | HTTP2 schema inventory only, oldHTTP1 unsupported; reference rejection fixes enforce documented existing invariant without changed persisted layout; complete |
| C4 | BYOT/codecs preserved; no mandatory agent/message/prompt/map model or reflection domain clone | generic I/O/E and codecs preserved; reflection restricted named service roots, RawMessage opaque, unsupported host roots refused; complete |
| C5 | Core stdlib, no network/vendor/credentials/neighbor dependency; optional packages outside core graph | go list -deps core contains only stdlib, evaly and internal/wirecontract; optional HTTP/observation/optimizer not imported by core; complete |
| C6 | Library/host ownership respected; no runtime/dashboard/scheduler/storage/deploy/prompt optimizer | bounded transport stays optional; host owns fixtures/isolation/retention/classification/retries; no unrelated service/runtime features; complete |
| C7 | No false forced-stop/exactly-once/arbitrary-secret/independence/judge-reliability promises | docs explicitly deny arbitrary-secret detection, forced cancellation and retries/exactly-once; prior host/judge/independence caveats retained; complete |
| C8 | AAA behavioral/adversarial/failure tests | new reference/export/HTTP/corpus tests explicit AAA comments, behavioral adversarial/cancellation/malformed inputs; complete |
| C9 | New ports/contracts have working reference implementation and conformance, no TODO interfaces | reference Handler/Target/EvidenceDelivery implemented; HTTP Target conformance and expanded Export reusable conformance; DecodeWire exercised independent roots; complete |
| C10 | Current affected-package/contracttest checks and concrete commands; final validate/examples, no historical PASS or weakened tests | independent make validate exit0 bothmodules fmt/vet/race; all6 examples exit0; finalhash stable; git diff --check pass; complete |
| C11 | Task boundaries respected; no DLP/reputation/streaming/observability/auth/storage/style/ignore/cache scope creep | diff reviewed: noDLP/reputation/streaming/auth/observability/storage/new model; no cache deletion or ignore changes; task03/04 designs untouched; complete |

Final result: **55/55 (100%)**. No incomplete requirement rows. Percentage
measures the entire fixed task scope; the separate correctness reviewer owns the
absence-of-identified-errors verdict. Preliminary malformed opaque escapes and
unbounded schema float range gaps were reported and repaired before this freeze.

Independent final checks:

- `python3 /private/tmp/evaly-review-state.py` before and after checks: hash above,
  93 files. Scope includes implementation/tests/docs/schema plus README and tasks
  01/02, excludes reviewer outputs and validation journal. Helper scope inspected.
- `GOCACHE=/private/tmp/evaly-final-1.27.1-cache make validate`: exit 0. Both modules
  formatter, vet and race tests passed. Independent log:
  `/private/tmp/evaly-evidence-completeness-validate.log`.
- All six `go run ./examples/{calculation,crm,protocols,http,observation,optimizer}`:
  exit 0. Logs `/private/tmp/evaly-evidence-completeness-example-*.log`.
- Independent opaque-reference reproduction: malformed `%zz` and `%` rejected,
  valid `%2F` retained, both originally reported credential examples rejected.
- `go list -deps` core module: only standard library, evaly and its own internal
  wirecontract package; no optional-package or third-party runtime dependency.
- `git diff --check`: exit 0. Generation drift check runs only in a temporary
  directory and checks schema filenames plus exact bytes.

All55 rows apply only to this fingerprint. Any implementation repair requires
new freeze and reacceptance. The normative boundary between structural JSON
shape and semantic checksum/state validation, and the byte-level duplicate-key
invariant that a parsed JSON Schema instance cannot recover, are explicitly
accounted for; they do not remove requirements from the denominator.
