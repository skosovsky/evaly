// Package optimizer evaluates immutable typed host candidates in bounded proposal
// rounds. Objectives, feasibility constraints, codecs and split validation belong
// to the host. Static enumeration uses the same protocol and accounting path.
// Best measured quality is distinct from a feasible winner; holdout runs after
// selection and cannot feed further rounds. Search v6 restoration rejects older
// formats without invoking proposal or evaluation callbacks.
// Config is executable host configuration; portable Result values are mutable
// caller-owned copies. Shared callbacks must be stable and concurrency safe.
// Evaluation errors are normally recorded as stopped histories; preflight and
// sealing/validation failures are return errors. A winner grants no deployment
// permission. Process-local reference ledger/budget do not establish durability.
package optimizer
