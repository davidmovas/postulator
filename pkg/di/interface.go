package di

import (
	"fmt"
	"reflect"
)

type Lifecycle uint8

const (
	Transient Lifecycle = iota
	Singleton
	Scoped
)

type ProviderFunc[T any] func(container Container) (T, error)

type CloseFunc func()

type Close interface {
	Close()
}

type IRegistration[T any] struct {
	Provider  ProviderFunc[T]
	Lifecycle Lifecycle
	Interface reflect.Type
}

type Container interface {
	Scope() Container
	Register(registrations ...any) error
	MustRegister(registrations ...any)
	Resolve(target any) error
	MustResolve(target any)
	AddCloseFunc(cleanupFunc CloseFunc)
	Close()
}

type TypedContainer[T any] struct {
	container Container
}

func (t TypedContainer[T]) For() *TypedResolver[T] {
	return &TypedResolver[T]{
		container: t.container,
	}
}

type TypedResolver[T any] struct {
	container Container
}

func (t *TypedResolver[T]) Get() (T, error) {
	var zero T
	target := new(T)
	if err := t.container.Resolve(target); err != nil {
		return zero, err
	}
	return *target, nil
}

func (t *TypedResolver[T]) MustGet() T {
	instance, err := t.Get()
	if err != nil {
		panic(fmt.Sprintf("di: failed to get: %v", err))
	}
	return instance
}
