package paging

import (
	"encoding/json"
	"time"
)

type SortKind uint8

const (
	Text SortKind = iota
	UUID
	Enum
	Int
	Float
	Bool
	Time
)

type SortKey[T any] struct {
	Value  func(T) any
	Field  string
	Column string
	Kind   SortKind
}

func newSortKey[T any](kind SortKind, field, column string, value func(T) any) SortKey[T] {
	return SortKey[T]{Kind: kind, Field: field, Column: column, Value: value}
}

func TextKey[T any](field, column string, value func(T) any) SortKey[T] {
	return newSortKey(Text, field, column, value)
}

func UUIDKey[T any](field, column string, value func(T) any) SortKey[T] {
	return newSortKey(UUID, field, column, value)
}

func EnumKey[T any](field, column string, value func(T) any) SortKey[T] {
	return newSortKey(Enum, field, column, value)
}

func IntKey[T any](field, column string, value func(T) any) SortKey[T] {
	return newSortKey(Int, field, column, value)
}

func FloatKey[T any](field, column string, value func(T) any) SortKey[T] {
	return newSortKey(Float, field, column, value)
}

func BoolKey[T any](field, column string, value func(T) any) SortKey[T] {
	return newSortKey(Bool, field, column, value)
}

func TimeKey[T any](field, column string, value func(T) any) SortKey[T] {
	return newSortKey(Time, field, column, value)
}

func (s SortKey[T]) literal(raw any) (any, bool) {
	switch s.Kind {
	case Text, UUID, Enum:
		return coerceString(raw)
	case Int:
		return coerceInt(raw)
	case Float:
		return coerceFloat(raw)
	case Bool:
		value, ok := raw.(bool)
		return value, ok
	case Time:
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

func coerceFloat(raw any) (any, bool) {
	switch value := raw.(type) {
	case float32:
		return float64(value), true
	case float64:
		return value, true
	case int:
		return float64(value), true
	case int32:
		return float64(value), true
	case int64:
		return float64(value), true
	case json.Number:
		parsed, err := value.Float64()
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
