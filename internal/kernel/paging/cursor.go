package paging

import (
	"bytes"
	"encoding/base64"
	"encoding/json"

	"github.com/davidmovas/postulator/internal/kernel/errors"
)

type Order string

const (
	Asc  Order = "asc"
	Desc Order = "desc"
)

func (o Order) String() string {
	return string(o)
}

func (o Order) reversed() Order {
	if o == Desc {
		return Asc
	}
	return Desc
}

func (o Order) sql() string {
	if o == Desc {
		return "DESC"
	}
	return "ASC"
}

func (o Order) comparison() string {
	if o == Desc {
		return "<"
	}
	return ">"
}

type Cursor string

type Position struct {
	OrderBy string `json:"o"`
	Order   Order  `json:"d"`
	Values  []any  `json:"v"`
	ID      string `json:"i"`
}

func encodeCursor(p Position) (Cursor, error) {
	encoded, err := json.Marshal(p)
	if err != nil {
		return "", errors.Wrap(err, errors.Internal, "encode cursor")
	}
	return Cursor(base64.RawURLEncoding.EncodeToString(encoded)), nil
}

func decodeCursor(c Cursor) (Position, error) {
	raw, err := base64.RawURLEncoding.DecodeString(string(c))
	if err != nil {
		return Position{}, errors.Wrap(err, errors.Invalid, "cursor is not base64url")
	}

	var p Position
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err = decoder.Decode(&p); err != nil {
		return Position{}, errors.Wrap(err, errors.Invalid, "cursor is not a valid position")
	}
	return p, nil
}
