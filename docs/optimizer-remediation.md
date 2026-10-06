# Optimizer artifact semantics and migration

Search v6 validates redundant audit fields against ordered attempts before every
successful return. ValidateResult uses the same contract; RestoreResult additionally
decodes/reseals candidates with the supplied codec. Content checksums establish
integrity, not producer truth or authenticity.

History is an ordered prefix of each round's Received, with deterministic dispatch
IDs derived from search, round and received position. Received revision is filled
only after sealing an attempted candidate. Candidates is exactly the accepted
revisions, excluding invalid attempts. Accepted IDs and revisions are unique;
ParentRevision refers to an earlier accepted revision. Invalid encoding may retain
unsealed lineage; duplicate/parent rejection may retain sealed diagnostics. Invalid
attempts have a safe reason and no authoritative experiment, comparison or quality.
Failed attempts require accepted lineage and a reason; bound partial measurements
may survive. Incomplete and evaluated attempts require experiment and comparison;
quality is present exactly when matched means are available. Evaluated additionally
requires sealed complete coverage and a conclusive gate verdict.

Calibration/holdout records are bound to derived experiment ID and concrete dataset
before canonical assignment, even alongside callback errors. Foreign/invalid records
are discarded; fixed safe failure categories stop subsequent dispatch. Valid bound
records with errors survive. Completed winners require holdout and comparison;
completed without winner requires no_selectable_candidate. States is exactly
proposing, optional selecting, terminal. Winner is the first evaluated feasible
GatePass in calibration ranking when selection was reached. Holdout failure retains
that calibration winner, without reselection or deploy approval (D39).
Candidate/round limit stops forbid holdout (D40).

ProposalUsage is the exact binary rational sum of recorded round units, rounded
upward once. Known is their conjunction, false without rounds; no-dispatch
unknown-zero rounds participate. A sum above MaxFloat64 returns ErrInvalid. Ranking
and BestMeasured are derived; retained audit copies cannot independently contradict
their source (D44).

D41: Parent is renamed ParentRevision on Proposal, CandidateRecord and
CandidateLineage. Candidate schema is v2, search is v6; candidate hashes change.
Reseal supported source descriptions deliberately; no implicit legacy reader or
field alias. Older search versions are ErrUnsupported. Update schema selectors and
callers using Parent. Descriptions remain BYOT codec-owned.

D42: empty Candidates always means normal exhaustion regardless of Exhausted;
Exhausted also ends after a last nonempty batch. D43: StaticProposer exposes
ValidateMaximum; Search calls it in preflight before budget authorization. Direct
Propose rejects oversized batches before decoding. No silent truncation.

D45: callback-facing JSON clones are checked; failure stops with
feedback_encoding_failure. Private core Record accessors use mustCloneJSON:
constructors validate/canonicalize and detach wire records before storage and expose
no mutable storage reference. Clone failure is an internal invariant breach and
panics explicitly instead of silently returning a zero copy. Host callbacks remain
outside this invariant and are not sandboxed or recovered.

D47: ledger claim still precedes holdout budget authorization. Contamination means
attempted claim, not proof of reading holdout. MemoryLedger is process local; hosts
must reconcile durable attempts after crashes. D48: Provenance records concrete
training/calibration/holdout dataset revisions and both baseline revisions separately
from the host Split label. Measurements/comparisons bind those revisions. ID
disjointness does not prove semantic independence; host split validation is opt-in.

D49: retain execution-order precedence. Later stop checks replace Reason:
proposal callback failure is overridden by cancellation, then reconciliation;
calibration protocol/budget/comparison failure by settlement, then cancellation,
then budget-exhausted trial/grade; holdout callback failure by settlement. Round
Reason records the terminal reason at its stop. Stop reasons additionally bind their execution phase: proposal/calibration/limit
stops cannot reach selection; holdout ledger/budget failures cannot retain holdout
artifacts; holdout evaluation failure cannot retain a comparison; holdout comparison
failure requires a bound holdout without comparison. Selection-stage stops require
a retained winner and completed proposal rounds. Failure categories are fixed,
never serialized raw errors. No secondary unbounded error list is introduced.

D50: ReceivedCount records the total returned count; ReceivedTruncated identifies
metadata capped at MaximumCandidates. Oversized batches reject in full, never
partially evaluate; only the first bounded metadata entries survive. This cap
bounds entry count, not string/description bytes. Hosts need byte limits for
untrusted descriptions/lineage; a global description limit would constrain BYOT.
Candidate count is not a complete memory sandbox.

Evidence: remediation_repro_test.go exercises five independently rehashed F08
mutations through both validators, four F09 phase/error combinations, and bound
measurements with operational errors. Existing tests cover cancellation, budget
stops, duplicates, feedback repair and isolation. remediation_edges_test.go covers
codec rejection, static mismatch, bounded metadata and overflow. usage_test.go and
semantic_fuzz_test.go exercise mixed scales and rehashed mutations. The disposable
baseline script reproduces F08/F09 on 76c224a without touching source refs.
