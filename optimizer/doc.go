// Package optimizer evaluates immutable typed host candidates in bounded proposal
// rounds. Objectives, feasibility constraints, codecs and split validation belong
// to the host. Static enumeration uses the same protocol and accounting path.
// Best measured quality is distinct from a feasible winner; holdout runs after
// selection and cannot feed further rounds. Search v4 restoration rejects older
// formats without invoking proposal or evaluation callbacks.
package optimizer
