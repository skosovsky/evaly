// Package httpjson provides an optional bounded JSON v2 target bridge using
// caller-owned codecs and explicit host evidence-delivery declarations. Delivered
// effects and known usage survive target errors; transport loss marks evidence
// incomplete. It neither runs an agent nor supplies a vendor SDK.
package httpjson
