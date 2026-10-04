Initial Go library release for evaluating typed targets.

- Immutable datasets, versioned codecs, bounded scenario runs and generation lineage.
- Isolated execution, cleanup, cancellation and atomic budget reservations.
- Deterministic and scripted judge grading, evidence capture, privacy projections and offline re-scoring.
- Compatible experiment comparisons, case-level uncertainty, local artifact persistence and export delivery.
- Optional observation workers, typed optimizer, HTTP JSON adapter, CLI and runnable examples.

Requires Go 1.27.1. The runtime core has no third-party dependencies.
The contracttest module is development-only and is not separately released.

Validation: formatting, vet and race tests in both modules, real local HTTP tests,
CLI subprocesses and six runnable examples. The initial independent acceptance
covered 171/171 requirements and closed 14 reproduced defects; subsequent updates
were checked with the documented test commands.

Known limits: callbacks must cooperate with contexts; filesystem guarantees are
local; reference budgets and holdout ledger are process-local; real LLM accuracy
and production integrations are unverified. The restored strict linter reports
219 outstanding findings (215 root, 4 contracttest); this release does not claim
strict-lint acceptance.
