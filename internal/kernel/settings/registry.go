package settings

import (
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/davidmovas/postulator/internal/kernel/errors"
)

type Values struct {
	current atomic.Pointer[[]any]
}

func (v *Values) at(slot int) any {
	entries := v.current.Load()
	if entries == nil || slot >= len(*entries) {
		return nil
	}
	return (*entries)[slot]
}

type Registry struct {
	byKey   map[string]*definition
	ordered []*definition
	mu      sync.RWMutex
	applyMu sync.Mutex
}

var defaultRegistry = New()

func New() *Registry {
	return &Registry{byKey: make(map[string]*definition)}
}

func Default() *Registry {
	return defaultRegistry
}

func (r *Registry) add(def *definition) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.byKey[def.key]; exists {
		panic(fmt.Errorf("settings: %q is already registered", def.key))
	}

	def.slot = len(r.ordered)
	r.byKey[def.key] = def
	r.ordered = append(r.ordered, def)
}

func (r *Registry) Keys() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return slices.Sorted(maps.Keys(r.byKey))
}

func (r *Registry) Has(key string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.byKey[key]
	return ok
}

func (r *Registry) NewValues() *Values {
	values := &Values{}
	values.current.Store(r.baseline())
	return values
}

func (r *Registry) baseline() *[]any {
	r.mu.RLock()
	defer r.mu.RUnlock()

	entries := make([]any, len(r.ordered))
	for i, def := range r.ordered {
		entries[i] = def.defaultValue
	}
	return &entries
}

func (r *Registry) Apply(values *Values, stored map[string]json.RawMessage) ([]string, error) {
	r.applyMu.Lock()
	defer r.applyMu.Unlock()

	next := r.baseline()

	r.mu.RLock()
	defer r.mu.RUnlock()

	var unknown []string
	for _, key := range slices.Sorted(maps.Keys(stored)) {
		def, known := r.byKey[key]
		if !known {
			unknown = append(unknown, key)
			continue
		}

		decoded, err := def.decode(stored[key])
		if err != nil {
			return unknown, errors.New(errors.Invalid, "setting "+key+" is not readable").WithDetail("key", key).WithInternal(err)
		}
		if err = def.validate(decoded); err != nil {
			return unknown, errors.New(errors.Invalid, "setting "+key+" is out of range").WithDetail("key", key).WithInternal(err)
		}
		(*next)[def.slot] = decoded
	}

	values.current.Store(next)
	return unknown, nil
}

func (r *Registry) Validate(key string, raw json.RawMessage) error {
	r.mu.RLock()
	def, known := r.byKey[key]
	r.mu.RUnlock()

	if !known {
		return errors.New(errors.NotFound, "setting "+key+" is not declared").WithDetail("key", key)
	}

	decoded, err := def.decode(raw)
	if err != nil {
		return errors.New(errors.Invalid, "setting "+key+" is not readable").WithDetail("key", key).WithInternal(err)
	}
	if err = def.validate(decoded); err != nil {
		return errors.New(errors.Invalid, "setting "+key+" is out of range").WithDetail("key", key).WithInternal(err)
	}
	return nil
}

type Descriptor struct {
	Default  json.RawMessage `json:"default"`
	Min      any             `json:"min,omitempty"`
	Max      any             `json:"max,omitempty"`
	Key      string          `json:"key"`
	Group    string          `json:"group"`
	Type     string          `json:"type"`
	Enum     []string        `json:"enum,omitempty"`
	NonEmpty bool            `json:"nonEmpty,omitempty"`
}

func (r *Registry) Schema() ([]Descriptor, error) {
	r.mu.RLock()
	defs := slices.Clone(r.ordered)
	r.mu.RUnlock()

	descriptors := make([]Descriptor, 0, len(defs))
	for _, def := range defs {
		encoded, err := def.encode(def.defaultValue)
		if err != nil {
			return nil, errors.Wrap(err, errors.Internal, "describe setting "+def.key)
		}

		descriptors = append(descriptors, Descriptor{
			Key:      def.key,
			Group:    string(GroupOf(def.key)),
			Type:     def.kind.String(),
			Default:  encoded,
			Min:      def.constraints.Min,
			Max:      def.constraints.Max,
			Enum:     def.constraints.Enum,
			NonEmpty: def.constraints.NonEmpty,
		})
	}

	slices.SortFunc(descriptors, func(a, b Descriptor) int { return strings.Compare(a.Key, b.Key) })
	return descriptors, nil
}

func (r *Registry) Bool(key string, defaultValue bool) *Setting[bool] {
	return register(r, key, KindBool, defaultValue, decodeBool, encodeBool, nil)
}

func (r *Registry) Int(key string, defaultValue int, validators ...Validator[int]) *Setting[int] {
	return register(r, key, KindInt, defaultValue, decodeInt, encodeInt, validators)
}

func (r *Registry) String(key, defaultValue string, validators ...Validator[string]) *Setting[string] {
	return register(r, key, KindString, defaultValue, decodeString, encodeString, validators)
}

func (r *Registry) Enum(key, defaultValue string, allowed []string) *Setting[string] {
	return register(r, key, KindEnum, defaultValue, decodeString, encodeString, []Validator[string]{OneOf(allowed)})
}

func (r *Registry) Duration(key string, defaultValue time.Duration, validators ...Validator[time.Duration]) *Setting[time.Duration] {
	return register(r, key, KindDuration, defaultValue, decodeDuration, encodeDuration, validators)
}

func Bool(key string, defaultValue bool) *Setting[bool] {
	return defaultRegistry.Bool(key, defaultValue)
}

func Int(key string, defaultValue int, validators ...Validator[int]) *Setting[int] {
	return defaultRegistry.Int(key, defaultValue, validators...)
}

func String(key, defaultValue string, validators ...Validator[string]) *Setting[string] {
	return defaultRegistry.String(key, defaultValue, validators...)
}

func Enum(key, defaultValue string, allowed []string) *Setting[string] {
	return defaultRegistry.Enum(key, defaultValue, allowed)
}

func Duration(key string, defaultValue time.Duration, validators ...Validator[time.Duration]) *Setting[time.Duration] {
	return defaultRegistry.Duration(key, defaultValue, validators...)
}
