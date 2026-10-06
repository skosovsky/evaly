# Budget accounting and recovery

Budget units remain finite nonnegative float64 host receipts; no shared currency
quantum is imposed. MemoryBudget interprets every accepted float as its exact
binary rational value, including zero, subnormal values and MaxFloat64. Admission
compares exact total liability plus the request to exact capacity. Reconciliation
replaces a reservation with exact known actual usage; it never computes a rounded
refund before subtracting. Release removes the exact unclaimed reservation.

`Used()` returns the smallest float64 at least as large as the exact total. That
display may be conservative by less than one float spacing; admission uses the
exact total rather than the display. The accepted total is bounded by finite
capacity, so reporting remains finite. No epsilon overspend or clamp-to-zero is
used. Two different scales cannot erase a small accepted liability, including
after release or settlement of a large one. Mutex protects the exact sum and entry
transitions together; this is a process-local adapter, not a distributed ledger.
Exact arithmetic has more cost than a float accumulator; no throughput claim is
made. The adapter retains reservation IDs to enforce dispatch identity semantics.

Migration: the public Budget, Usage and Reservation types and wire numeric fields
are unchanged. Hosts keep choosing their units and bounds. MemoryBudget now rejects
requests that previously slipped past capacity through rounding, and Used may round
up where the old sum lost a small liability. Host budget implementations remain
responsible for atomicity, precision and recovery; this change does not certify
their arithmetic or billing data.

## Dispatch liability decisions

- D02 — preserve liability after Claim. Claim authorizes risk even if cancellation
  or projection failure prevents a callback from running. Releasing a claimed
  reservation requires a separate host protocol that proves no external dispatch;
  evaly does not infer that proof from a callback count.
- D03 — preserve liability after an ambiguous Claim error. A distributed host may
  commit a claim before losing its reply. The caller must correlate by dispatch ID
  and reconcile with the host's authoritative ledger; automatic refund/retry would
  risk a duplicate external effect. No distributed retry engine is added to core.
- D46 — preserve explicit unknown optimizer overhead. EvaluationUnits reserves the
  overhead for the Evaluate callback, while target/grader usage is accounted
  inside evaluation separately. Without an overhead receipt a dispatched positive
  reservation stays unknown, not known zero. Zero overhead may settle known zero.

Identical active Reserve calls return the same reservation; changed units or a
released ID conflict. Claim succeeds once. Reconcile with unknown usage retains
the full reservation and marks it dispatched; a known receipt cannot exceed the
bound. Repeated identical known settlement is idempotent; changed settlement
conflicts. Release is idempotent before dispatch and conflicts after claim or
settlement. Unknown usage is not evidence of zero cost.

Verification includes both F03 repros against the reviewed baseline through a
read-only Go overlay (`python3 scripts/budget_baseline_repro.py`), AAA regressions,
mixed-scale operation orders, finite extremes and concurrent accounting/claim.
`FuzzMemoryBudgetSettlement` varies binary float bounds and receipts and checks
that full capacity never admits positive extra units and known actual survives.
