package dto

import (
	"encoding/json"
	"time"

	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	DefaultLimit = 50
	MaxLimit     = 500
)

type Time time.Time

func NewTime(t time.Time) Time {
	return Time(t.UTC().Truncate(time.Second))
}

func (t Time) Std() time.Time {
	return time.Time(t)
}

func (t Time) String() string {
	std := t.Std()
	if std.IsZero() {
		return ""
	}
	return std.UTC().Format(time.RFC3339)
}

func (t Time) MarshalJSON() ([]byte, error) {
	if t.Std().IsZero() {
		return []byte("null"), nil
	}
	return json.Marshal(t.String())
}

func (t *Time) UnmarshalJSON(raw []byte) error {
	if string(raw) == "null" {
		*t = Time(time.Time{})
		return nil
	}

	var text string
	if err := json.Unmarshal(raw, &text); err != nil {
		return errors.Wrap(err, errors.Invalid, "timestamp must be an RFC3339 string")
	}
	if text == "" {
		*t = Time(time.Time{})
		return nil
	}

	parsed, err := time.Parse(time.RFC3339, text)
	if err != nil {
		return errors.Wrap(err, errors.Invalid, "timestamp must be an RFC3339 string")
	}
	*t = NewTime(parsed)
	return nil
}

type Sort struct {
	Field string `json:"field"`
	Desc  bool   `json:"desc"`
}

type ListRequest struct {
	Cursor string `json:"cursor,omitempty"`
	Limit  int    `json:"limit"`
	Sort   *Sort  `json:"sort,omitempty"`
}

func (r ListRequest) Normalize() ListRequest {
	switch {
	case r.Limit <= 0:
		r.Limit = DefaultLimit
	case r.Limit > MaxLimit:
		r.Limit = MaxLimit
	}
	return r
}
