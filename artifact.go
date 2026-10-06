package evaly

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
)

type Envelope struct {
	Version    int                        `json:"version"`
	Kind       string                     `json:"kind"`
	ID         string                     `json:"id"`
	Data       json.RawMessage            `json:"data"`
	Checksum   string                     `json:"checksum"`
	Extensions map[string]json.RawMessage `json:"extensions,omitempty"`
}

func validArtifactID(id string) bool {
	ok, _ := regexp.MatchString(`^[a-zA-Z0-9_-]{1,128}$`, id)
	return ok
}
func NewEnvelope(kind, id string, data any) (Envelope, error) {
	b, e := canonical(data)
	if e != nil {
		return Envelope{}, e
	}
	env := Envelope{Version: 1, Kind: kind, ID: id, Data: b, Checksum: "", Extensions: nil}
	env.Checksum = checksumEnvelope(env)
	return env, ValidateEnvelope(env)
}
func checksumEnvelope(e Envelope) string {
	e.Checksum = ""
	b, err := canonical(e)
	if err != nil {
		return ""
	}
	return digest(b)
}
func ValidateEnvelope(e Envelope) error {
	if e.Version != 1 {
		return ErrUnsupported
	}
	switch e.Kind {
	case "dataset",
		"experiment",
		"comparison",
		"evidence", observationMode, searchMode, "view",
		"scenario",
		"assessment", candidateArtifactKind:
	default:
		return ErrUnsupported
	}
	if !validArtifactID(e.ID) {
		return ErrInvalid
	}
	data, err := CanonicalJSON(e.Data)
	if err != nil || len(data) == 0 || data[0] != '{' {
		return ErrInvalid
	}
	if e.Checksum == "" || e.Checksum != checksumEnvelope(e) {
		return ErrCorrupt
	}
	return nil
}

// ArtifactStore publishes immutable envelopes, idempotent by ID and bytes.
type ArtifactStore interface {
	Put(context.Context, Envelope) error
	Get(context.Context, string) (Envelope, error)
}
type StoreCapabilities struct {
	AtomicPublication bool `json:"AtomicPublication"`
	Deduplication     bool `json:"Deduplication"`
	Checksums         bool `json:"Checksums"`
	MultiHost         bool `json:"MultiHost"`
}

// FileStore is a local reference store. Configure MaxBytes and Fault before use
// and keep them immutable during concurrent operations. Filesystem durability
// does not imply network filesystem or multi-host correctness.
type FileStore struct {
	directory string
	MaxBytes  int
	Fault     func(string) error
}

func OpenFileStore(directory string) (*FileStore, error) {
	if directory == "" {
		return nil, ErrInvalid
	}
	abs, e := filepath.Abs(directory)
	if e != nil {
		return nil, e
	}
	if e = os.MkdirAll(abs, 0700); e != nil {
		return nil, e
	}
	info, e := os.Stat(abs)
	if e != nil || !info.IsDir() {
		return nil, ErrInvalid
	}
	return &FileStore{directory: abs, MaxBytes: defaultArtifactMaxBytes, Fault: nil}, nil
}
func (s *FileStore) Capabilities() StoreCapabilities {
	return StoreCapabilities{true, true, true, false}
}
func (s *FileStore) fault(stage string) error {
	if s.Fault != nil {
		return s.Fault(stage)
	}
	return nil
}
func (s *FileStore) Put(ctx context.Context, e Envelope) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := ValidateEnvelope(e); err != nil {
		return err
	}
	b, err := canonical(e)
	if err != nil {
		return err
	}
	if len(b) > s.MaxBytes {
		return ErrInvalid
	}
	dest := filepath.Join(s.directory, e.ID+".json")
	if previous, errLocal := s.Get(ctx, e.ID); errLocal == nil {
		old, _ := canonical(previous)
		if string(old) == string(b) {
			return nil
		}
		return ErrConflict
	} else if !errors.Is(errLocal, os.ErrNotExist) {
		return errLocal
	}
	f, err := os.CreateTemp(s.directory, ".staged-*")
	if err != nil {
		return err
	}
	name := f.Name()
	defer os.Remove(name)
	defer f.Close()
	validated, err := s.validateStaged(f, name, b)
	if err != nil {
		return err
	}
	if err = ctx.Err(); err != nil {
		return err
	}
	linked, err := s.linkStaged(ctx, name, dest, validated, b)
	if err != nil || !linked {
		return err
	}
	if err = s.fault("committed"); err != nil {
		return err
	}
	dir, err := os.Open(s.directory)
	if err != nil {
		return err
	}
	defer dir.Close()
	return dir.Sync()
}
func (s *FileStore) Get(ctx context.Context, id string) (Envelope, error) {
	var out Envelope
	if err := ctx.Err(); err != nil {
		return out, err
	}
	if !validArtifactID(id) || s.MaxBytes <= 0 {
		return out, ErrInvalid
	}
	path := filepath.Join(s.directory, id+".json")
	info, err := os.Lstat(path)
	if err != nil {
		return out, err
	}
	if !info.Mode().IsRegular() {
		return out, ErrCorrupt
	}
	if info.Size() > int64(s.MaxBytes) {
		return out, ErrCorrupt
	}
	f, err := os.Open(path)
	if err != nil {
		return out, err
	}
	defer f.Close()
	b, err := io.ReadAll(io.LimitReader(f, int64(s.MaxBytes)+1))
	if err != nil {
		return out, err
	}
	if len(b) > s.MaxBytes {
		return out, ErrCorrupt
	}
	if err = ctx.Err(); err != nil {
		return out, err
	}
	if _, err = CanonicalJSON(b); err != nil {
		return out, ErrCorrupt
	}
	out, err = DecodeWire[Envelope](b)
	if err != nil {
		return out, fmt.Errorf("%w: envelope", ErrCorrupt)
	}
	if err = ValidateEnvelope(out); err != nil {
		return out, err
	}
	if out.ID != id {
		return out, ErrCorrupt
	}
	return out, nil
}
func SaveExperiment(ctx context.Context, s ArtifactStore, e Experiment) error {
	if err := ValidatePort(s); err != nil {
		return err
	}
	if _, err := RestoreExperiment(e.Record()); err != nil {
		return err
	}
	env, err := NewEnvelope("experiment", e.ID(), e.Record())
	if err != nil {
		return err
	}
	return s.Put(ctx, env)
}
func LoadExperiment(ctx context.Context, s ArtifactStore, id string) (Experiment, error) {
	if err := ValidatePort(s); err != nil {
		return Experiment{}, err
	}
	env, err := s.Get(ctx, id)
	if err != nil {
		return Experiment{}, err
	}
	if env.Kind != "experiment" {
		return Experiment{}, ErrUnsupported
	}
	r, err := DecodeWire[ExperimentRecord](env.Data)
	if err != nil {
		return Experiment{}, ErrCorrupt
	}
	e, err := RestoreExperiment(r)
	if err != nil {
		return Experiment{}, err
	}
	if e.ID() != id {
		return Experiment{}, ErrCorrupt
	}
	if r.Manifest.State != sealedState {
		return Experiment{}, ErrUnsealed
	}
	return e, nil
}

func (s *FileStore) validateStaged(f *os.File, name string, b []byte) (Envelope, error) {
	var err error
	if err = s.fault("staged"); err != nil {
		return Envelope{}, err
	}
	if _, err = f.Write(b); err != nil {
		return Envelope{}, err
	}
	if err = f.Sync(); err != nil {
		return Envelope{}, err
	}
	if err = f.Close(); err != nil {
		return Envelope{}, err
	}
	read, err := os.ReadFile(name)
	if err != nil {
		return Envelope{}, err
	}
	validated, err := DecodeWire[Envelope](read)
	if err != nil {
		return Envelope{}, err
	}
	if err = ValidateEnvelope(validated); err != nil {
		return Envelope{}, err
	}
	if err = s.fault("validated"); err != nil {
		return Envelope{}, err
	}
	return validated, nil
}

func (s *FileStore) linkStaged(ctx context.Context, name, dest string, validated Envelope, b []byte) (bool, error) {
	var err error
	if err = os.Link(name, dest); err == nil {
		return true, nil
	}
	if !errors.Is(err, os.ErrExist) {
		return false, err
	}
	previous, e := s.Get(ctx, validated.ID)
	if e != nil {
		return false, e
	}
	old, _ := canonical(previous)
	if string(old) == string(b) {
		return false, nil
	}
	return false, ErrConflict
}
