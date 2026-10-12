package evaly

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"net/url"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"
)

type Event struct {
	Version       int             `json:"version"`
	Sequence      int             `json:"sequence"`
	Kind          string          `json:"kind"`
	CorrelationID string          `json:"correlation_id"`
	Payload       json.RawMessage `json:"payload,omitempty"`
	References    []string        `json:"references,omitempty"`
}
type EvidenceRecord struct {
	Version         int             `json:"version"`
	State           string          `json:"state"`
	Revision        string          `json:"revision"`
	Policy          string          `json:"policy"`
	Events          []Event         `json:"events"`
	Coverage        map[string]bool `json:"coverage"`
	Redactions      int             `json:"redactions"`
	Truncated       bool            `json:"truncated"`
	Gaps            []int           `json:"gaps,omitempty"`
	Errors          []string        `json:"errors,omitempty"`
	ReplayAvailable bool            `json:"replay_available"`
}

// CapturePolicy MUST project before retention. Implementations must not retain raw input.
type CapturePolicy interface {
	Revision() string
	Project(context.Context, Event) (Event, error)
}

// FieldPolicy keeps only explicitly allowed top-level JSON fields. Nested values
// in allowed fields are host-classified. References default to being removed.
// Configure before use: Allowed and its slices must remain immutable while the policy
// is used. Copying this value does not copy its map or the allowed-field slices.
type FieldPolicy struct {
	ID             string              `json:"ID"`
	Allowed        map[string][]string `json:"Allowed"`
	KeepReferences bool                `json:"KeepReferences"`
}

func (p FieldPolicy) Revision() string { return p.ID }
func (p FieldPolicy) Project(ctx context.Context, e Event) (Event, error) {
	if err := ctx.Err(); err != nil {
		return Event{}, err
	}
	out := e
	out.Payload = nil
	out.References = nil
	if p.KeepReferences {
		out.References = append([]string(nil), e.References...)
		for _, ref := range out.References {
			if !safeReference(ref) {
				return Event{}, ErrInvalid
			}
		}
	}
	if len(e.Payload) == 0 {
		return out, nil
	}
	if _, err := CanonicalJSON(e.Payload); err != nil {
		return Event{}, err
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(e.Payload, &raw); err != nil {
		return Event{}, ErrInvalid
	}
	selected := map[string]json.RawMessage{}
	for _, key := range p.Allowed[e.Kind] {
		if v, ok := raw[key]; ok {
			selected[key] = v
		}
	}
	if len(selected) > 0 {
		b, err := canonical(selected)
		if err != nil {
			return Event{}, err
		}
		out.Payload = b
	}
	return out, nil
}
func safeReference(ref string) bool {
	if ref == "" || !utf8.ValidString(ref) || strings.Contains(ref, "#") || strings.IndexFunc(ref, func(r rune) bool {
		return unicode.IsSpace(r) || unicode.IsControl(r)
	}) >= 0 {
		return false
	}
	// url.Parse does not validate percent escapes inside an opaque URI.
	if _, err := url.PathUnescape(ref); err != nil {
		return false
	}
	u, e := url.Parse(ref)
	if e != nil || u.User != nil {
		return false
	}
	if (strings.EqualFold(u.Scheme, "http") || strings.EqualFold(u.Scheme, "https")) &&
		(u.Hostname() == "" || u.Opaque != "") {
		return false
	}
	query, err := url.ParseQuery(u.RawQuery)
	if err != nil {
		return false
	}
	for k := range query {
		key := strings.ToLower(k)
		if strings.Contains(key, "token") || strings.Contains(key, "secret") || strings.Contains(key, "key") ||
			strings.Contains(key, "password") || strings.Contains(key, "passwd") ||
			strings.Contains(key, "credential") || strings.Contains(key, "authorization") ||
			strings.Contains(key, "signature") || key == "auth" || key == "pwd" || key == "sig" {
			return false
		}
	}
	return true
}

// EvidenceSink retains permitted events and can conservatively invalidate channel coverage.
// MarkIncomplete records only host_incomplete; arbitrary host diagnostics stay outside artifacts.
type EvidenceSink interface {
	Record(context.Context, Event) error
	MarkIncomplete()
}
type CaptureConfig struct {
	Policy        CapturePolicy
	RequiredKinds []string
	KnownKinds    []string
	MaxEvents     int
	MaxBytes      int
}

// Capture bounds retained data and seals irreversibly. Record is thread-safe.
type Capture struct {
	mu     sync.Mutex
	config CaptureConfig
	record EvidenceRecord
	last   int
	bytes  int
	closed bool
}

func NewCapture(c CaptureConfig) (*Capture, error) {
	if err := ValidatePort(c.Policy); err != nil {
		return nil, err
	}
	if c.Policy == nil || c.Policy.Revision() == "" || c.MaxEvents <= 0 || c.MaxBytes <= 0 {
		return nil, ErrInvalid
	}
	c.RequiredKinds = append([]string(nil), c.RequiredKinds...)
	c.KnownKinds = append([]string(nil), c.KnownKinds...)
	r := EvidenceRecord{
		Version:         1,
		State:           "open",
		Policy:          c.Policy.Revision(),
		Coverage:        map[string]bool{},
		Events:          []Event{},
		ReplayAvailable: true, Revision: "", Redactions: 0, Truncated: false, Gaps: nil, Errors: nil,
	}
	for _, k := range c.KnownKinds {
		if k == "" {
			return nil, ErrInvalid
		}
		r.Coverage[k] = true
	}
	for _, k := range c.RequiredKinds {
		if _, ok := r.Coverage[k]; !ok {
			return nil, ErrUnsupported
		}
	}
	return &Capture{config: c, record: r, mu: sync.Mutex{}, last: 0, bytes: 0, closed: false}, nil
}
func (c *Capture) Record(ctx context.Context, e Event) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return ErrClosed
	}
	if err := ctx.Err(); err != nil {
		return c.failLocked(err)
	}
	if len(e.Payload) > c.config.MaxBytes {
		c.record.Truncated = true
		return c.failLocked(ErrIncomplete)
	}
	if e.Version != 1 {
		return c.failLocked(ErrUnsupported)
	}
	if _, ok := c.record.Coverage[e.Kind]; !ok {
		return c.failLocked(ErrUnsupported)
	}
	if e.Sequence != c.last+1 {
		if len(c.record.Gaps) < maxEvidenceDiagnostics {
			c.record.Gaps = append(c.record.Gaps, c.last+1)
		}
		for k := range c.record.Coverage {
			c.record.Coverage[k] = false
		}
	}
	if e.Sequence <= c.last {
		return c.failLocked(ErrConflict)
	}
	c.last = e.Sequence
	// A defensive copy prevents a policy mutating caller-owned raw data.
	raw, err := cloneCaptureInput(e)
	if err != nil {
		return c.failLocked(ErrInvalid)
	}
	out, err := c.config.Policy.Project(ctx, raw)
	if err != nil {
		return c.failLocked(err)
	}
	out, err = validateProjectedEvent(e, out)
	if err != nil {
		return c.failLocked(err)
	}
	// CanonicalJSON historically normalizes invalid metadata strings before
	// sizing. Marshal alone can use longer escaped replacements otherwise.
	out.Kind = normalizeJSONString(out.Kind)
	out.CorrelationID = normalizeJSONString(out.CorrelationID)
	// Deep payloads keep the original whole-Event boundary: nesting limits
	// include the enclosing Event object, even after payload canonicalization.
	deep := captureNeedsJSONBoundary(out.Payload)
	b, err := encodeProjectedEvent(out, deep)
	if err != nil {
		return c.failLocked(err)
	}
	if len(c.record.Events) >= c.config.MaxEvents {
		c.record.Truncated = true
		return c.failLocked(ErrIncomplete)
	}
	if c.bytes+len(b) > c.config.MaxBytes {
		c.record.Truncated = true
		return c.failLocked(ErrIncomplete)
	}
	if string(out.Payload) != string(e.Payload) || len(out.References) != len(e.References) {
		c.record.Redactions++
	}
	c.bytes += len(b)
	out, err = cloneProjectedEvent(out, deep)
	if err != nil {
		return c.failLocked(err)
	}
	c.record.Events = append(c.record.Events, out)
	return nil
}
func (c *Capture) MarkIncomplete() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return
	}
	if len(c.record.Errors) < maxEvidenceDiagnostics {
		c.record.Errors = append(c.record.Errors, "host_incomplete")
	}
	for k := range c.record.Coverage {
		c.record.Coverage[k] = false
	}
}
func (c *Capture) Seal() EvidenceRecord {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.closed {
		c.closed = true
		c.record.State = sealedState
		if len(c.record.Errors) > 0 || len(c.record.Gaps) > 0 || c.record.Truncated {
			c.record.State = incompleteState
		}
		b, err := canonical(c.record)
		if err != nil {
			panic(fmt.Sprintf("internal evidence encoding: %v", err))
		}
		c.record.Revision = digest(b)
	}
	// Config strings are host-owned and historically normalized by the JSON
	// round trip. Keep that boundary for non-UTF-8 policy IDs/coverage keys.
	if !utf8.ValidString(c.record.Policy) {
		return mustCloneJSON(c.record)
	}
	for kind := range c.record.Coverage {
		if !utf8.ValidString(kind) {
			return mustCloneJSON(c.record)
		}
	}
	for _, event := range c.record.Events {
		if captureNeedsJSONBoundary(event.Payload) {
			return mustCloneJSON(c.record)
		}
	}
	return cloneEvidenceRecord(c.record)
}
func ValidateEvidence(r EvidenceRecord) error {
	if r.Version != 1 {
		return ErrUnsupported
	}
	if r.State != sealedState && r.State != incompleteState {
		return ErrUnsealed
	}
	if r.Policy == "" || r.Coverage == nil || r.Redactions < 0 || len(r.Errors) > maxEvidenceDiagnostics ||
		len(r.Gaps) > maxEvidenceDiagnostics {
		return ErrInvalid
	}
	if r.State == sealedState && (len(r.Errors) > 0 || len(r.Gaps) > 0 || r.Truncated || !r.ReplayAvailable) {
		return ErrInvalid
	}
	hasGap, err := validateEvidenceEvents(r)
	if err != nil {
		return err
	}
	if hasGap && len(r.Gaps) == 0 {
		return ErrInvalid
	}
	if hasGap || len(r.Gaps) > 0 || len(r.Errors) > 0 || r.Truncated || !r.ReplayAvailable {
		for _, complete := range r.Coverage {
			if complete {
				return ErrInvalid
			}
		}
	}
	rev := r.Revision
	r.Revision = ""
	b, e := canonical(r)
	if e != nil {
		return ErrCorrupt
	}
	if digest(b) != rev {
		return ErrCorrupt
	}
	return nil
}
func CompleteFor(e EvidenceRecord, kind string) bool {
	return ValidateEvidence(e) == nil && e.Coverage[kind] && !e.Truncated
}

func validateEvidenceEvents(r EvidenceRecord) (bool, error) {
	last := 0
	hasGap := false
	for _, event := range r.Events {
		if event.Version != 1 {
			return false, ErrUnsupported
		}
		if _, ok := r.Coverage[event.Kind]; !ok {
			return false, ErrUnsupported
		}
		if event.Sequence <= last {
			return false, ErrConflict
		}
		if event.Sequence != last+1 {
			hasGap = true
		}
		last = event.Sequence
		if len(event.Payload) > 0 {
			if _, err := CanonicalJSON(event.Payload); err != nil {
				return false, err
			}
		}
		for _, ref := range event.References {
			if !safeReference(ref) {
				return false, ErrInvalid
			}
		}
	}
	return hasGap, nil
}

func (c *Capture) failLocked(err error) error {
	if len(c.record.Errors) < maxEvidenceDiagnostics {
		c.record.Errors = append(c.record.Errors, "capture_error")
	}
	for k := range c.record.Coverage {
		c.record.Coverage[k] = false
	}
	return err
}

func validateProjectedEvent(e, out Event) (Event, error) {
	var err error
	if out.Version != e.Version || out.Sequence != e.Sequence || out.Kind != e.Kind ||
		out.CorrelationID != e.CorrelationID {
		return Event{}, ErrInvalid
	}
	for _, ref := range out.References {
		if !safeReference(ref) {
			return Event{}, ErrInvalid
		}
	}
	if len(out.Payload) > 0 {
		out.Payload, err = CanonicalJSON(out.Payload)
		if err != nil {
			return Event{}, err
		}
	}
	return out, nil
}

// mustValidatedCapture is used only after successful execution preflight. A
// subsequent failure means a host policy/config changed its validated behavior;
// do not ignore that error or disguise it as an ordinary empty capture.
func mustValidatedCapture(config CaptureConfig) *Capture {
	capture, err := NewCapture(config)
	if err != nil {
		panic("evaly: validated capture configuration changed")
	}
	return capture
}

// cloneCaptureInput mirrors the Event JSON round trip at the untrusted caller
// boundary. RawMessage marshaling validates and compacts JSON, including the
// default HTML escaping, without encoding and decoding the surrounding Event.
func cloneCaptureInput(e Event) (Event, error) {
	if captureNeedsJSONBoundary(e.Payload) {
		return cloneJSON(e)
	}
	if len(e.Payload) > 0 {
		payload, err := json.Marshal(e.Payload)
		if err != nil {
			return Event{}, err
		}
		e.Payload = payload
	} else {
		e.Payload = nil
	}
	e.Kind = normalizeJSONString(e.Kind)
	e.CorrelationID = normalizeJSONString(e.CorrelationID)
	if len(e.References) == 0 {
		e.References = nil
	} else {
		references := make([]string, len(e.References))
		for i, reference := range e.References {
			references[i] = normalizeJSONString(reference)
		}
		e.References = references
	}
	return e, nil
}

// encoding/json replaces each invalid UTF-8 decoding unit independently.
// [strings.ToValidUTF8] coalesces adjacent invalid bytes and is not equivalent.
func normalizeJSONString(s string) string {
	if utf8.ValidString(s) {
		return s
	}
	var out strings.Builder
	out.Grow(len(s))
	for _, r := range s {
		out.WriteRune(r)
	}
	return out.String()
}

// cloneEvidenceRecord copies only validated, privately owned wire records.
// It must not replace normalization or validation at a caller/BYOT boundary.
func cloneEvidenceRecord(r EvidenceRecord) EvidenceRecord {
	if r.Events != nil {
		events := make([]Event, len(r.Events))
		for i, event := range r.Events {
			event.Payload = append(json.RawMessage(nil), event.Payload...)
			event.References = append([]string(nil), event.References...)
			events[i] = event
		}
		r.Events = events
	}
	if r.Coverage != nil {
		r.Coverage = maps.Clone(r.Coverage)
	}
	r.Gaps = append([]int(nil), r.Gaps...)
	r.Errors = append([]string(nil), r.Errors...)
	return r
}

// Counting opening containers conservatively bounds nesting without a second
// parser. Brackets inside strings may trigger the legacy path unnecessarily,
// but cannot cause deeply nested JSON to miss the original boundary checks.
func captureNeedsJSONBoundary(payload []byte) bool {
	const fastPathContainerLimit = 128
	return bytes.Count(payload, []byte{'{'})+bytes.Count(payload, []byte{'['}) >= fastPathContainerLimit
}

// encodeProjectedEvent keeps the original wrapper validation for complex JSON.
// For shallow canonical payloads, sorting fixed Event keys cannot change size.
func encodeProjectedEvent(e Event, useJSONBoundary bool) ([]byte, error) {
	if useJSONBoundary {
		return canonical(e)
	}
	return json.Marshal(e)
}

// cloneProjectedEvent receives a validated canonical payload owned by evaly.
// Complex JSON retains the original whole-Event copy and depth validation.
func cloneProjectedEvent(e Event, useJSONBoundary bool) (Event, error) {
	if useJSONBoundary {
		return cloneJSON(e)
	}
	e.References = append([]string(nil), e.References...)
	return e, nil
}
