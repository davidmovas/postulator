package di

import (
	"fmt"
	"reflect"
	"sync"
)

type container struct {
	parent          *container
	registrations   *sync.Map // map[reflect.Type]*entry
	instances       *sync.Map // map[reflect.Type]any
	scopedInstances *sync.Map // map[reflect.Type]any
	mu              sync.RWMutex
	isScoped        bool
	cleanedUp       bool
	cleanupFuncs    []func()
}

type entry struct {
	provider     reflect.Value
	lifecycle    Lifecycle
	resolvedType reflect.Type
	mu           sync.Mutex
}

func New() Container {
	return &container{
		registrations:   &sync.Map{},
		instances:       &sync.Map{},
		scopedInstances: &sync.Map{},
	}
}

func Typed[T any](container Container) TypedContainer[T] {
	return TypedContainer[T]{container: container}
}

func (c *container) Scope() Container {
	return &container{
		parent:          c,
		registrations:   c.registrations,
		instances:       c.instances,
		scopedInstances: &sync.Map{},
		isScoped:        true,
	}
}

func (c *container) Register(registrations ...any) error {
	for _, reg := range registrations {
		if err := c.registerSingle(reg); err != nil {
			return err
		}
	}
	return nil
}

func (c *container) MustRegister(registrations ...any) {
	if err := c.Register(registrations...); err != nil {
		panic(fmt.Sprintf("di: failed to register: %v", err))
	}
}

func (c *container) registerSingle(registration any) error {
	regValue := reflect.ValueOf(registration)
	if regValue.Kind() != reflect.Ptr || regValue.IsNil() {
		return fmt.Errorf("di: registration must be non-nil pointer, got %T", registration)
	}

	regElem := regValue.Elem()
	if regElem.Type().Name() != "Registration" {
		return fmt.Errorf("di: expected *Registration, got %T", registration)
	}

	interfaceField := regElem.FieldByName("InterfaceType")
	if !interfaceField.IsValid() {
		return fmt.Errorf("di: invalid registration structure")
	}

	resolvedType := interfaceField.Interface().(reflect.Type)
	providerField := regElem.FieldByName("Provider")
	lifecycle := Lifecycle(regElem.FieldByName("Lifecycle").Int())

	c.registrations.Store(resolvedType, &entry{
		provider:     providerField,
		lifecycle:    lifecycle,
		resolvedType: resolvedType,
	})

	return nil
}

func (c *container) Resolve(target any) error {
	if target == nil {
		return fmt.Errorf("di: target cannot be nil")
	}

	targetValue := reflect.ValueOf(target)
	if targetValue.Kind() != reflect.Ptr || targetValue.IsNil() {
		return fmt.Errorf("di: target must be non-nil pointer, got %T", target)
	}

	targetType := targetValue.Elem().Type()
	instance, err := c.resolveType(targetType)
	if err != nil {
		return err
	}

	targetValue.Elem().Set(reflect.ValueOf(instance))
	return nil
}

func (c *container) MustResolve(target any) {
	if err := c.Resolve(target); err != nil {
		panic(fmt.Sprintf("di: failed to resolve: %v", err))
	}
}

func (c *container) AddCloseFunc(fn CloseFunc) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cleanupFuncs = append(c.cleanupFuncs, fn)
}

func (c *container) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.cleanedUp {
		return
	}

	for i := len(c.cleanupFuncs) - 1; i >= 0; i-- {
		c.cleanupFuncs[i]()
	}

	c.cleanupFuncs = nil
	c.cleanedUp = true

	if c.isScoped {
		c.scopedInstances.Range(func(key, value any) bool {
			if cleanupable, ok := value.(Close); ok {
				cleanupable.Close()
			}
			c.scopedInstances.Delete(key)
			return true
		})
	}
}

func (c *container) resolveType(targetType reflect.Type) (any, error) {
	if instance, ok := c.getInstanceFromCache(targetType); ok {
		return instance, nil
	}

	e, err := c.getRegistrationEntry(targetType)
	if err != nil {
		return nil, err
	}

	return c.createInstance(e)
}

func (c *container) getInstanceFromCache(targetType reflect.Type) (any, bool) {
	if c.isScoped {
		if instance, ok := c.scopedInstances.Load(targetType); ok {
			return instance, true
		}
	}

	if instance, ok := c.instances.Load(targetType); ok {
		return instance, true
	}

	return nil, false
}

func (c *container) getRegistrationEntry(targetType reflect.Type) (*entry, error) {
	if e, ok := c.registrations.Load(targetType); ok {
		return e.(*entry), nil
	}

	if c.parent != nil {
		return c.parent.getRegistrationEntry(targetType)
	}

	return nil, fmt.Errorf("di: no registration found for type %v", targetType)
}

func (c *container) createInstance(entry *entry) (any, error) {
	if entry.lifecycle == Singleton || entry.lifecycle == Scoped {
		entry.mu.Lock()
		defer entry.mu.Unlock()

		if instance, ok := c.getInstanceFromCache(entry.resolvedType); ok {
			return instance, nil
		}
	}

	results := entry.provider.Call([]reflect.Value{reflect.ValueOf(c)})
	if len(results) != 2 {
		return nil, fmt.Errorf("di: provider must return (T, error)")
	}

	instance := results[0].Interface()
	errVal := results[1].Interface()
	if errVal != nil {
		return nil, errVal.(error)
	}

	switch entry.lifecycle {
	case Singleton:
		c.instances.Store(entry.resolvedType, instance)
	case Scoped:
		if c.isScoped {
			c.scopedInstances.Store(entry.resolvedType, instance)
		} else {
			c.instances.Store(entry.resolvedType, instance)
		}
	default:
	}

	return instance, nil
}
