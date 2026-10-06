# Structural preflight and grading boundaries

Required capabilities are validated before identity, decode or external dispatch.
ValidatePort rejects nil interfaces and typed nil pointers/functions/maps/slices/
channels; an optional StructuralValidator can reject local setup without probing
paid capabilities. Validations, identities and callback behavior must remain stable
during use. Arbitrary host panics are not caught by a global recovery mechanism.

Dataset, scenario, saved-view and candidate restore reject absent/structurally invalid
codecs with ErrInvalid or the validator's explicit error before Identity/Decode.
All peer codec structures are checked before any peer Identity is used in root
dataset/scenario/view helpers. Malformed supplied identity returns ErrInvalid;
valid incompatible identity returns ErrUnsupported. Existing record version/state/
integrity checks may run first, so invalid record and invalid port combined do not
have a universal precedence. Search result restore also checks its supplied codec
structure and identity before decoding candidate values.

SaveExperiment, LoadExperiment, SaveSavedView and LoadSavedView validate the store
before Put/Get. LoadSavedView also validates all supplied codecs before Get.
Structural validation errors are returned explicitly. Store host implementations
own authentication, durability and error behavior after dispatch; these helpers
do not introduce implicit retries or refunds. Passing an adapter directly to its
own methods remains subject to that adapter's declared setup contract.

## Absence constructor migration (F07)

AbsenceGrader now returns `Absence[I,O,R]`, a stock grader with private configuration
and a structural validator. Its zero value, nil predicate, missing revision, empty
kind and invalid UTF-8 kind are invalid independently of evidence contents.
Kinds remain open host strings; the grader does not introduce an event registry or
infer that an unmatched channel means a valid absence proof.

The function's arguments and Grader interface are unchanged. Use an inferred local
variable or `Grader[I,O,R]`; code explicitly assigning the result to GraderFunc or
mutating its Evaluate/Identity fields must switch to the constructor and methods.
ValidatePort and direct Grade reject invalid setup. Assess returns a nondispatched
GraderError for an invalid plan without calling its view factory. Run returns a
preflight error before lifecycle/target/projection effects; observation.Start returns
an error and no worker before starting its work. Constructor validation is local;
predicate semantics and concurrent access remain host responsibilities.

## Recorded decisions

- D08 — replace the exhaustive no-op reflect switch with a finite nil-kind list;
  relocate ValidateCodecIdentity GoDoc to its function. This keeps typed-nil
  detection without enumerating irrelevant scalar kinds or weakening lint rules.
- D20 — retain the fresh-view host promise. Run/SavedView/paired helpers decode
  owned snapshots; an arbitrary view factory must return fresh permitted data for
  each grader. Reflection cloning arbitrary domain types would violate BYOT and
  would not sandbox closures. Tests `TestAssessPreflightsWholePlanAndFreshFactories`
  and `TestSavedViewIsolatesMutatingGraders` retain these boundaries.
- D21 — retain distinct grading results. Dispatched GraderError is a result,
  and grading completion does not mean product success. Consumers inspect status
  plus assertions/metrics and apply an explicit gate; no fallback pass is introduced.
- D27 — retain channel completeness independently of event count. A complete known
  channel with zero events can prove absence when the predicate is valid. Incomplete
  coverage with zero or matching events returns InsufficientEvidence, never pass.
- D28 — align public Save/Load and restore capability preflight. Missing ports are
  configuration errors, not unsupported schema or identity conflicts. Migration:
  invalid SavedView/Candidate codec configuration now reports ErrInvalid/validator
  error, valid incompatible Candidate codec reports ErrUnsupported; malformed record
  revisions still fail integrity checks. No panic-catching adapter wrapper is added.

AAA verification covers nil/typed nil for both dataset/scenario codecs, rejected
validators without Identity/Decode, compatible/incompatible identity and roundtrip;
all store helpers, three view codec positions before I/O, and invalid absence setup
across ValidatePort/direct Grade/Assess/Run/observation. F06/F07 baseline repro uses
`python3 scripts/preflight_baseline_repro.py` with read-only Go overlays and asserts
both original typed-nil panics and nil-predicate empty-pass/matching-event panic.
