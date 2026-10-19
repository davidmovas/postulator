package paging

import (
	"cmp"
	"slices"
)

func (k Keyset[T]) Page(rows []T, request Request) (List[T], error) {
	if err := k.validate(); err != nil {
		return List[T]{}, err
	}
	request = request.Normalize()
	if request.cursor() == "" {
		return k.Cut(rows[:min(len(rows), request.Limit+1)], request)
	}
	position, err := k.Position(request.cursor())
	if err != nil {
		return List[T]{}, err
	}
	at := len(rows)
	for i := range rows {
		if k.after(rows[i], position) >= 0 {
			at = i
			break
		}
	}
	if request.backward() {
		before := slices.Clone(rows[max(0, at-request.Limit-1):at])
		slices.Reverse(before)
		return k.Cut(before, request)
	}
	if at < len(rows) && k.after(rows[at], position) == 0 {
		at++
	}
	return k.Cut(rows[at:min(len(rows), at+request.Limit+1)], request)
}

func (k Keyset[T]) after(item T, position Position) int {
	order := 1
	if k.Desc {
		order = -1
	}
	for i, key := range k.Keys {
		value, _ := key.literal(key.Value(item))
		if compared := compareLiteral(value, position.Values[i]); compared != 0 {
			return compared * order
		}
	}
	return cmp.Compare(k.ID(item), position.ID) * order
}

func compareLiteral(a, b any) int {
	switch left := a.(type) {
	case string:
		right, ok := b.(string)
		if !ok {
			return 0
		}
		return cmp.Compare(left, right)
	case int64:
		right, ok := b.(int64)
		if !ok {
			return 0
		}
		return cmp.Compare(left, right)
	default:
		return 0
	}
}
