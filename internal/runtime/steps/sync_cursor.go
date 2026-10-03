package steps

import (
	"encoding/json"

	"github.com/davidmovas/postulator/internal/kernel/errors"
)

type coreCursor struct {
	Type string `json:"type"`
	Page int    `json:"page"`
}

func typeIndex(name string) int {
	for i := range coreTypes {
		if string(coreTypes[i]) == name {
			return i
		}
	}
	return 0
}

func decodeCore(cursor string) (coreCursor, error) {
	if cursor == "" {
		return coreCursor{Type: string(coreTypes[0]), Page: 1}, nil
	}

	var decoded coreCursor
	if err := json.Unmarshal([]byte(cursor), &decoded); err != nil {
		return coreCursor{}, errors.Wrap(err, errors.Invalid, "the stored sync cursor is not readable")
	}
	if decoded.Page < 1 {
		decoded.Page = 1
	}
	return decoded, nil
}

func encodeCore(position coreCursor) string {
	encoded, err := json.Marshal(position)
	if err != nil {
		return ""
	}
	return string(encoded)
}
