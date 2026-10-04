// Package evaly measures typed targets using immutable datasets, isolated trials,
// permitted evidence and explicit grading outcomes. Domain types, credentials,
// production policy and environment provisioning belong to the caller.
//
// Callback ports must honor context cancellation. The package does not claim
// exactly-once external effects or statistically independent repeated trials.
package evaly
