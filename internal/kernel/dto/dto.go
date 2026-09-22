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
	Field string `json:"field" description:"The field to order by; a cursor is issued for one order and refuses another"`
	Desc  bool   `json:"desc" description:"Order from the largest value down rather than up"`
}

type ListRequest struct {
	Cursor string `json:"cursor,omitempty" description:"The nextCursor a previous page returned; leave it out for the first page"`
	Limit  int    `json:"limit,omitempty" description:"How many rows to return; leave it out for the default"`
	Sort   *Sort  `json:"sort,omitempty" description:"How to order the rows; leave it out for the default order"`
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
