// Package observation grades already performed observations on a bounded worker.
// It cannot call production targets. Partial assessment retains completed paid
// grades and explicit skipped outcomes. Offline re-score creates parent-linked
// assessment revisions from saved permitted views without execution capabilities.
// Host owns sampling, retention and storage.
package observation
