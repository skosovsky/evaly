package evaly

// Snapshot seals a caller value through its explicit codec. It never copies
// arbitrary domain values implicitly, and each Value call decodes private bytes.
type Snapshot[T any] struct {
	codec    Codec[T]
	data     []byte
	revision string
	identity CodecIdentity
}

func SealSnapshot[T any](value T, codec Codec[T]) (Snapshot[T], error) {
	var s Snapshot[T]
	if ValidatePort(codec) != nil || ValidateCodecIdentity(codec.Identity()) != nil {
		return s, ErrInvalid
	}
	identity := codec.Identity()
	b, e := codec.Encode(value)
	if e != nil {
		return s, e
	}
	b, e = CanonicalJSON(b)
	if e != nil {
		return s, e
	}
	if codec.Identity() != identity {
		return s, ErrConflict
	}
	encoded, e := canonical(struct {
		Codec CodecIdentity
		Data  []byte
	}{identity, b})
	if e != nil {
		return s, e
	}
	return Snapshot[T]{
		codec:    codec,
		data:     append([]byte(nil), b...),
		revision: digest(encoded),
		identity: identity,
	}, nil
}
func (s Snapshot[T]) Revision() string { return s.revision }
func (s Snapshot[T]) Validate() error {
	if ValidatePort(s.codec) != nil || s.revision == "" {
		return ErrUnsealed
	}
	if ValidateCodecIdentity(s.identity) != nil || s.codec.Identity() != s.identity {
		return ErrInvalid
	}
	b, e := canonical(struct {
		Codec CodecIdentity
		Data  []byte
	}{s.identity, s.data})
	if e != nil || digest(b) != s.revision {
		return ErrCorrupt
	}
	return nil
}
func (s Snapshot[T]) Value() (T, error) {
	var zero T
	if e := s.Validate(); e != nil {
		return zero, e
	}
	return s.codec.Decode(append([]byte(nil), s.data...))
}
