package paging

import (
	"encoding/json"
	"time"
)

type sortKind uint8

const (
	sortText sortKind = iota
	sortInt
	sortTime
)

type SortKey[T any] struct {
	Value  func(T) any
	Field  string
	Column string
	kind   sortKind
}

func newSortKey[T any](kind sortKind, field, column string, value func(T) any) SortKey[T] {
	return SortKey[T]{kind: kind, Field: field, Column: column, Value: value}
}

func TextKey[T any](field, column string, value func(T) any) SortKey[T] {
	return newSortKey(sortText, field, column, value)
}

func IntKey[T any](field, column string, value func(T) any) SortKey[T] {
	return newSortKey(sortInt, field, column, value)
}

func TimeKey[T any](field, column string, value func(T) any) SortKey[T] {
	return newSortKey(sortTime, field, column, value)
}

func (s SortKey[T]) literal(raw any) (any, bool) {
	switch s.kind {
	case sortText:
		return coerceString(raw)
	case sortInt:
		return coerceInt(raw)
	case sortTime:
		return coerceTime(raw)
	default:
		return nil, false
	}
}

func coerceString(raw any) (any, bool) {
	switch value := raw.(type) {
	case string:
		return value, true
	case []byte:
		return string(value), true
	default:
		return nil, false
	}
}

func coerceInt(raw any) (any, bool) {
	switch value := raw.(type) {
	case int:
		return int64(value), true
	case int32:
		return int64(value), true
	case int64:
		return value, true
	case float64:
		return int64(value), true
	case json.Number:
		parsed, err := value.Int64()
		if err != nil {
			return nil, false
		}
		return parsed, true
	default:
		return nil, false
	}
}

func coerceTime(raw any) (any, bool) {
	switch value := raw.(type) {
	case time.Time:
		return value.UTC().Format(time.RFC3339), true
	case string:
		parsed, err := time.Parse(time.RFC3339, value)
		if err != nil {
			return nil, false
		}
		return parsed.UTC().Format(time.RFC3339), true
	default:
		return nil, false
	}
}
