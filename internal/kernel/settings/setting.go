package settings

import (
	"encoding/json"
	"fmt"
	"strings"
)

const maxKeyLength = 128

type Kind uint8

const (
	KindBool Kind = iota
	KindInt
	KindString
	KindEnum
	KindDuration
)

func (k Kind) String() string {
	switch k {
	case KindBool:
		return "bool"
	case KindInt:
		return "int"
	case KindString:
		return "string"
	case KindEnum:
		return "enum"
	case KindDuration:
		return "duration"
	default:
		return "unknown"
	}
}

type definition struct {
	decode       func(json.RawMessage) (any, error)
	encode       func(any) (json.RawMessage, error)
	validate     func(any) error
	defaultValue any
	key          string
	constraints  Constraints
	slot         int
	kind         Kind
}

type Setting[T any] struct {
	def *definition
}

func (s *Setting[T]) Key() string {
	return s.def.key
}

func (s *Setting[T]) Kind() Kind {
	return s.def.kind
}

func (s *Setting[T]) Default() T {
	typed, ok := s.def.defaultValue.(T)
	if !ok {
		var zero T
		return zero
	}
	return typed
}

func (s *Setting[T]) Get(values *Values) T {
	if values == nil {
		return s.Default()
	}
	typed, ok := values.at(s.def.slot).(T)
	if !ok {
		return s.Default()
	}
	return typed
}

func register[T any](
	reg *Registry,
	key string,
	kind Kind,
	defaultValue T,
	decode func(json.RawMessage) (T, error),
	encode func(T) (json.RawMessage, error),
	validators []Validator[T],
) *Setting[T] {
	if err := checkKey(key); err != nil {
		panic(err)
	}

	validate := func(value any) error {
		typed, ok := value.(T)
		if !ok {
			return fmt.Errorf("value has type %T, want %T", value, *new(T))
		}
		for _, validator := range validators {
			if err := validator.check(typed); err != nil {
				return err
			}
		}
		return nil
	}

	var constraints Constraints
	for _, validator := range validators {
		if validator.describe != nil {
			validator.describe(&constraints)
		}
	}

	if err := validate(defaultValue); err != nil {
		panic(fmt.Errorf("settings: default for %q is invalid: %w", key, err))
	}

	def := &definition{
		key:          key,
		kind:         kind,
		defaultValue: defaultValue,
		constraints:  constraints,
		validate:     validate,
		encode: func(value any) (json.RawMessage, error) {
			typed, ok := value.(T)
			if !ok {
				return nil, fmt.Errorf("value has type %T, want %T", value, *new(T))
			}
			return encode(typed)
		},
		decode: func(raw json.RawMessage) (any, error) {
			typed, err := decode(raw)
			if err != nil {
				return nil, err
			}
			return typed, nil
		},
	}

	reg.add(def)
	return &Setting[T]{def: def}
}

func checkKey(key string) error {
	if key == "" {
		return fmt.Errorf("settings: key must not be empty")
	}
	if len(key) > maxKeyLength {
		return fmt.Errorf("settings: key %q is longer than %d characters", key, maxKeyLength)
	}

	segments := strings.Split(key, ".")
	if len(segments) < 2 {
		return fmt.Errorf("settings: key %q must be grouped as <group>.<name>", key)
	}
	for _, segment := range segments {
		if segment == "" {
			return fmt.Errorf("settings: key %q has an empty segment", key)
		}
	}
	return nil
}
