# Remediation verification

Baseline: `76c224a97e5f03a44e87ab7ea308216a7e10d139`.
This is a durable record of task acceptance, not a claim of live provider quality,
distributed durability or exhaustive absence of bugs.

For each task record: frozen reviewed diff/base, mandatory criterion count and
completeness percentage, independent reviewer reports, commands/outcomes/limits,
and commit mapping. A commit can be located by its unique subject below; later
records also record its SHA. Pending tasks have no acceptance claim.

## T01 — accepted

Changes: ordered checklist with complete F/D ownership; numeric/JSON/preflight/
optimizer target contracts. Implementation behavior is unchanged.

Checks: source task, current budget/codec/preflight/restore/search/release/Makefile
read; clean starting worktree on main. Documentation-only task; runtime tests are
not evidence of target-contract compliance and are deferred to implementation.

Initial independent review: completeness 2/3 (66.67%), missing D22 ownership;
correctness identified the same gap, missing accepted candidate ID uniqueness and
unspecified proposal usage overflow policy. Corrected by assigning D22 to T09,
requiring unique accepted IDs/revisions, and specifying exact rational aggregate
with upward finite reporting and ErrInvalid on overflow. Both reviewers must
recheck this corrected version.

Final independent acceptance: `/root/t01_completeness` — 3/3, 100%;
`/root/t01_correctness` — no errors found in the reviewed specification scope.
Both reviewed plan SHA256 `1e1a971c3c692f36530a9d76252be1e2b14534ac0112c0ac6f030935b23c84ed`
and contracts SHA256 `49932f6f010193183bffdca241f2e5dd69ee83fbd7b81c0f24d41c340ea06dbb`.
After review only acceptance metadata and the T01 checkbox were updated.
Whitespace checks on both new documents produced no diagnostics (`git diff
--no-index --check /dev/null PATH`; exit 1 indicates added content).
Commit subject: `docs: define remediation contracts`.
