# Task 2 implementation contracts

Status: Spec-First target for the ordered remediation plan. Existing implementation
is known to violate these contracts; compliance is established task by task.
The original review acceptance cases remain mandatory.

## Exact reference budget accounting (F03)

Keep finite nonnegative float64 host-facing units. Each accepted float denotes its
exact binary rational value. MemoryBudget stores capacity and the sum of active
liabilities with `math/big.Rat`, never a rounded float accumulator. This alternative
to integer quanta is selected because the current Budget port and all producer
usage receipts accept float64; it fixes the reference adapter without forcing hosts
to invent a common quantum or silently quantizing their existing receipts.

All finite float64 values, including subnormals and MaxFloat64, are supported.
Admission compares exact used + requested against exact capacity. Each live entry
contributes its reservation, or known settled actual; released entries contribute
zero. Reconciliation subtracts the exact reservation and adds the exact actual,
never subtracting a rounded `reserved-actual`. Unknown usage retains the reservation.
Used returns the smallest float64 greater than or equal to the exact total, computed
by rounding to float64 and advancing toward +Inf if that rounded below the total.
The total is bounded by finite capacity, so this reporting value remains finite.
No epsilon admission, clamp-to-zero, or approximate refund is permitted.

Keep reserve idempotency for identical active IDs, conflict for changed units or
released IDs, exactly-once Claim, Release only before claim/settlement, and known
actual ≤ reservation. All accounting remains mutex-atomic and process-local.
Host distributed recovery and billing accuracy are outside this adapter.

## JSON representation and ownership (F04/F05, D23/D24)

Generic interface-valued decode uses json.Number at every nesting level. Typed
ints/structs retain encoding/json decoding and overflow errors. Canonical identity
sorts object keys and preserves numeric lexemes: `1`, `1.0` and `1e0` are distinct
canonical representations even when numerically equal. Encode→Decode→Encode must
retain canonical bytes for supported sealed values. Domain dynamic Go types are
not promised across generic decode. Typed representations that normalize numeric
spellings must be rejected before successful sealing when reversibility fails.

Cycle detection must distinguish type/kind and slice extent, with an ancestor stack
rather than a global deduplication set. A struct pointer and first-field pointer,
empty aliases and repeated references in separate branches are legal. Actual
pointer/map/slice ancestor cycles fail. JSON-ignored fields are outside serialization
validation. UTF-8 checks follow the declared supported serialized field semantics;
custom MarshalJSON implementations own source-value validation, while emitted JSON
still undergoes canonical validation. Any unsupported field-selection corner must
be documented rather than falsely claiming complete encoding/json parity.

Codec outputs are caller-owned bytes stable after Encode returns; Decode returns
fresh owned mutable values and must not retain/mutate input. Public helpers copy
bytes at ownership boundaries. Host codecs/closures must keep identity, behavior
and validation stable during a call and be concurrency-safe when shared. Evaly
cannot sandbox a broken host codec or preserve bytes overwritten before return.

## Structural preflight (F06/F07, D28)

Every required port is checked through ValidatePort before Identity/Decode or
effectful dispatch. Nil interfaces, typed nil and failing StructuralValidator
return ErrInvalid (or the explicit validator error). Valid nonnil incompatible
codec identity returns ErrUnsupported at dataset/scenario restore. Validate stored
record semantics independently; corrupt inputs may fail before port validation.
Save/Load helpers follow the same structural port policy. Optional ports are
validated when supplied. No catch-all recovery of host callback panics.

AbsenceGrader has a structural validator for revision, nonempty valid event kind
and nonnil predicate; invalid setup fails independently of evidence contents.
ValidatePort/Assess/Run/observation reject setup before target/view/grader effects.
Complete empty coverage can prove absence; incomplete coverage is insufficient.

## Optimizer live/restore state contract (F08/F09)

Retain audit fields, validating redundant relationships rather than treating each
as an independent truth. Live Search with nil error must return a sealed Result
accepted by ValidateResult and RestoreResult with its supported codec. Validation
runs before successful return; host outputs are bound before assignment to any
canonical slot, even when the callback also returns an operational error.

| Artifact/state | Required relationships |
|---|---|
| Round | Contiguous index; dispatch `searchID/proposal/index`; stable proposal revision; terminal completed/stopped; stopped round only last in stopped search. Received records preserve returned lineage, including rejected/unattempted inputs. |
| Attempt | Ordered prefix of its round's Received; dispatch `searchID/evaluation/round/receivedIndex`; lineage matches received ID/parent/algorithm/codec. Never an unrelated dispatch string. |
| Accepted candidate | Sealed valid candidate, matching search algorithm and bound lineage. Exactly the round Candidates list in attempt order, excluding invalid attempts. Accepted IDs and revisions are each unique; parent references valid earlier accepted revisions. |
| Invalid attempt | Explicit rejection reason; no authoritative experiment/comparison/quality. Encoding rejection may have unsealed lineage; duplicate/lineage rejection may carry valid sealed candidate diagnostics without accepting it. |
| Failed attempt | Accepted candidate required; safe failure reason; only valid bound partial experiment allowed; measurement optional according to failure stage. |
| Incomplete attempt | Accepted candidate and bound experiment/comparison for a successfully measured incomplete result; quality present iff comparison makes matched means available. |
| Evaluated attempt | Accepted candidate, bound experiment and comparison required; quality iff matched means available; feasibility is recorded independently of measured quality. |
| Measurement | Experiment ID derived from search/phase/candidate revision; experiment restorable; comparison binds experiment revision and objective; quality agrees with matched means. |
| Aggregate usage | Units are the exact binary rational sum of recorded round units, rounded upward once to float64 as in budget reporting. If the sum exceeds MaxFloat64, Search returns ErrInvalid rather than a nil-error sealed result; ValidateResult rejects that aggregate. Known is conjunction of all recorded rounds, false when there are no rounds. A no-dispatch round retains unknown zero and participates in that conjunction. |
| Ranking/BestMeasured | Derived from all measured qualities in objective direction, revision tie-break; no duplicates; BestMeasured is first or empty. |
| Winner | First selectable evaluated feasible GatePass in ranking when selection was reached; stopped before selection has no winner. A winner retained during holdout failure is valid. |
| Completed with winner | Both valid bound holdout experiment and holdout comparison mandatory. Holdout failure never causes candidate reselection. |
| Completed without winner | No holdout artifacts; explicit no-selectable reason; no invented successful candidate. |
| Stopped | Nonempty safe reason; valid partial history. Winner/holdout allowed only when selection was reached and artifacts are bound; missing holdout is legal after holdout budget/ledger/protocol/cancellation failure. |
| States | Exact reachable path: proposing, optional selecting, terminal state. Selection is required for completed and for any retained winner/holdout; no arbitrary intermediate states. |

Wrong-ID or otherwise invalid host measurements never enter canonical Experiment
or Holdout. Persist a bounded safe diagnostic category instead of raw host output.
Stop further dispatch on protocol violation. Valid bound partial records returned
alongside errors remain available. Duplicate/invalid candidates, cancellation,
budget stop, and paid failed proposals retain legitimate audit history.

Result bytes have content integrity, not producer authenticity. Restore checks
internal consistency, not host truth or statistical independence. Changes to wire
fields/revisions are intentional and synchronized with schema and migration docs;
there are no implicit legacy readers.
