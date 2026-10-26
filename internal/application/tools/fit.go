package tools

import "encoding/json"

const (
	fitHeadroomDivisor = 16
	defaultPageLimit   = 50
)

func fitted[T any](deps Deps, limit int, page func(limit int) (T, error)) (T, error) {
	if limit <= 0 {
		limit = defaultPageLimit
	}
	room := 0
	if deps.ResultBytes != nil {
		room = deps.ResultBytes()
		room -= room / fitHeadroomDivisor
	}

	for {
		answer, err := page(limit)
		if err != nil || room <= 0 || limit <= 1 {
			return answer, err
		}
		encoded, marshalErr := json.Marshal(answer)
		if marshalErr != nil || len(encoded) <= room {
			return answer, nil
		}
		limit /= 2
	}
}
