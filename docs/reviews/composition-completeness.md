# Independent completeness review — issue #1

Authority: `.cursor/docs/task1.md`; https://github.com/skosovsky/evaly/issues/1.
Review is technical acceptance before release/closeout. Source: current worktree,
including untracked optional module, workflow and consumer script. Unrelated user
files were excluded. Final remediation and both dependency matrices have been independently inspected.
Reviewed implementation fingerprint (SHA-256 over Makefile, consumer script/workflow,
optional Go sources and go.mod/go.sum): `aa3be57f5abc3aa64716a33cb308c235d040105b9a7e2c3f19dad34e653709a8`.

## Final result

**100%: 10/10 AC fully proven.** Partial/pending criteria receive no credit.
Published and source consumer logs both contain PASS with `-race -count=1`;
`make validate` finished successfully; final root config/format checks also passed. Tests were inspected, not merely
counted. No core Go source/API/schema changes occur in the reviewed diff.

| AC | Verdict | Evidence / outstanding action |
|---|---|---|
| AC1 | Complete | Normative composition contract plus integration/evidence/budget/migration additions; generic StreamingTarget/JudgeAdapter preserve BYOT; separate go.mod isolates SDK imports; repository search finds no SDK imports outside integrations. |
| AC2 | Complete | TestRoundtripAndRealEvaluationDelivery seals dataset, runs complete real SDK stream, grades, writes/reopens/loads FileStore, compares complete records, exports actual SDK spans. Config fixes target, conversion, export, capture and grading projection provenance. |
| AC3 | Complete | TestStreamingTerminalFailures covers EOF, source error, incomplete outcome, close failure, output decode, cancellation and observation error; asserts one dispatch, synchronous source unwind, retained receipt/event, no Completed/pass/complete evidence. TestRequiredCaptureAndDeferredFailure covers pre-cancellation, unsupported handle and open result+error close. |
| AC4 | Complete | JudgeErrorsUnavailableAndTrustedData now runs real SDK chat template, struct render plan and execution; deterministic SDK transport witnesses system trusted rubric versus adversarial user content. Error+usage, decode failure, unavailable and negative grades remain distinct. Reset/required-capture tests fail closed. Final source and published matrices PASS. |
| AC5 | Complete | TestConversionPresenceAndLiability covers unknown/zero/total-only/components, negative/delta/over-bound, invalid/nonfinite rates and incompatible units. Actual and reservation use same converter. Exact-upward-rounding test prevents undercount above float integer precision. |
| AC6 | Complete | CaptureConservativeAndRequiredFailure covers sampling, gap, truncation, failed/missing source report and record rejection; AbsenceGrader is insufficient evidence. OutcomeIndependentOfConfidentText distinguishes answer text from real effect. Failure tests retain permitted partial event. |
| AC7 | Complete | Roundtrip test proves default disk artifacts, nested payloads/references and SDK spans marker-free; terminal tests cover partial evidence/diagnostics. IndependentExportPrivacyAndSampling separately preserves a host-allowed opaque sensitive reference in restricted artifact, removes it in export projection, checks all delivered DTOs even when sampling off and actual spans when on, and proves source artifact unchanged. Final matrices PASS. |
| AC8 | Complete | Individual assertion/metric/unmeasured records, full metric semantics in sidecar; roundtrip verifies real created/ended SDK evaluation spans, partial accepted-then-ambiguous delivery retry, changed-content conflict and unchanged verdict. Concurrent dedup and illegal projection tests; repeated/policy-dependent IDs and absent scores for unavailable/error. |
| AC9 | Complete | Consumer script copies portable module, disables workspace, enforces published no replacements, prints actual resolved identities and dirty source status; workflow checks out clean SDK sources. Both semantic matrix logs PASS, no credentials or private audit materials required. Hosted run is subsequent publication evidence, not a substitute for local matrix. |
| AC10 | Complete | make validate completed successfully; final root config/format check passed after optional-module exclusion changes. Both final consumer matrices run fmt/lint/vet and race tests including remediation, all PASS. ExampleStreamingTarget and migration show exact host changes. No core schema change or replaced legacy path exists. |

## Boundaries and original scope

Implementation matches the optional target/capture/export composition scope. Core
remains universal and dependency-independent. Host retains privacy, units, domain
outcomes, callbacks, credentials and durable dedup ownership. No deferred scheduler,
common domain DTO or core vendor client was introduced. Sidecar retains metric
semantics absent from telemetry DTO; this limitation is explicit. Scripted SDK
fixtures prove composition only, not provider quality or injection immunity.

## Remaining goal actions after technical acceptance

Both identified fixture gaps are closed; full validation and final matrices PASS.
Technical acceptance is complete. Execute appropriate release
command following release safety, verify published release, post author-addressed
closeout explaining concrete consumer migration, and close issue as completed.
These release/closeout actions are not counted as missing technical AC at this stage.

## Remediation audit

Initial review scored 70%; AC4/AC7 coverage gaps were fixed without shrinking scope.
Both final logs include PASS for revised judge/privacy fixtures and new invalid-sidecar
and foreign-identity regressions. Registry validates semantic sidecar before hashing/
recording; sink rejects identity substitutions before any delivery while allowing
privacy removal. No unresolved completeness findings remain.
