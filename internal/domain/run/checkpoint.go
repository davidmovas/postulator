package run

import (
	"encoding/json"
	"maps"

	"github.com/davidmovas/postulator/internal/kernel/errors"
)

type Checkpoint map[string]json.RawMessage

func NewCheckpoint() Checkpoint {
	return Checkpoint{}
}

func (c Checkpoint) Clone() Checkpoint {
	clone := make(Checkpoint, len(c))
	maps.Copy(clone, c)
	return clone
}

func (c Checkpoint) MergedWith(other Checkpoint) Checkpoint {
	merged := c.Clone()
	maps.Copy(merged, other)
	return merged
}

func (c Checkpoint) Encode() (string, error) {
	if c == nil {
		return "{}", nil
	}
	encoded, err := json.Marshal(map[string]json.RawMessage(c))
	if err != nil {
		return "", errors.Wrap(err, errors.Internal, "encode the checkpoint")
	}
	return string(encoded), nil
}

func DecodeCheckpoint(raw string) (Checkpoint, error) {
	if raw == "" {
		return NewCheckpoint(), nil
	}
	decoded := Checkpoint{}
	if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
		return nil, errors.Wrap(err, errors.Internal, "decode the checkpoint")
	}
	return decoded, nil
}

func Get[T any](c Checkpoint, key string) (value T, found bool, err error) {
	raw, ok := c[key]
	if !ok || len(raw) == 0 {
		return value, false, nil
	}
	if unmarshalErr := json.Unmarshal(raw, &value); unmarshalErr != nil {
		return value, false, errors.Wrap(unmarshalErr, errors.Internal, "decode the checkpoint key "+key)
	}
	return value, true, nil
}

func Set[T any](c Checkpoint, key string, value T) error {
	if c == nil {
		return errors.New(errors.Internal, "the checkpoint is not allocated, so the write would be lost").
			WithDetail("key", key)
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return errors.Wrap(err, errors.Internal, "encode the checkpoint key "+key)
	}
	c[key] = encoded
	return nil
}
