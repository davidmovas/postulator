package paging

import "encoding/json"

const (
	DefaultLimit = 50
	MaxLimit     = 500
)

type Slice[T any] []T

func (s Slice[T]) MarshalJSON() ([]byte, error) {
	if s == nil {
		return []byte("[]"), nil
	}
	return json.Marshal([]T(s))
}

type Cursors struct {
	Next Cursor `json:"nextCursor,omitempty"`
	Prev Cursor `json:"prevCursor,omitempty"`
}

type List[T any] struct {
	Cursors
	Items   Slice[T] `json:"items"`
	HasMore bool     `json:"hasMore"`
}

type Request struct {
	After  Cursor
	Before Cursor
	Limit  int
}

func (r Request) Normalize() Request {
	switch {
	case r.Limit <= 0:
		r.Limit = DefaultLimit
	case r.Limit > MaxLimit:
		r.Limit = MaxLimit
	}
	return r
}

func (r Request) backward() bool {
	return r.After == "" && r.Before != ""
}

func (r Request) cursor() Cursor {
	if r.After != "" {
		return r.After
	}
	return r.Before
}
