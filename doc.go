// Package evaly measures typed targets using immutable datasets, isolated trials,
// permitted evidence and explicit grading outcomes. Domain types, credentials,
// production policy and environment provisioning belong to the caller.
//
// Current wire readers reject unsupported prior major formats; no compatibility
// layer is supplied. Optional HTTP, observation and optimizer packages compose
// these ports without becoming core dependencies. Scripted examples and conformance
// suites check infrastructure behavior; live model accuracy requires host evaluation.
//
// Callback ports must honor context cancellation. The package does not claim
// exactly-once external effects or statistically independent repeated trials.
package evaly
