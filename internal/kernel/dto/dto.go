package dto

import (
	"encoding/json"
	"time"

	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	defaultLimit = 50
	maxLimit     = 500
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
	Field string `json:"field" description:"Order field, fixed across pages"`
	Desc  bool   `json:"desc,omitempty" description:"Descending"`
}

type ListRequest struct {
	Cursor string `json:"cursor,omitempty" description:"nextCursor of the previous page"`
	Limit  int    `json:"limit,omitempty" description:"Page size, default 50, max 500"`
	Sort   *Sort  `json:"sort,omitempty" description:"Row order"`
}

func PageSize(requested int) int {
	switch {
	case requested <= 0:
		return defaultLimit
	case requested > maxLimit:
		return maxLimit
	default:
		return requested
	}
}
