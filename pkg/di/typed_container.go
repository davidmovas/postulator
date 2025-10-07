package di

import (
	"fmt"
	"reflect"
)

type typedContainer[T any] struct {
	container Container
}

func (t *typedContainer[T]) Get() (T, error) {
	var zero T
	targetType := reflect.TypeOf(zero)

	instance, err := t.container.(*container).resolveType(targetType)
	if err != nil {
		return zero, err
	}

	return instance.(T), nil
}

func (t *typedContainer[T]) MustGet() T {
	instance, err := t.Get()
	if err != nil {
		panic(fmt.Sprintf("di: failed to get: %v", err))
	}
	return instance
}

func (t *typedContainer[T]) Resolve() (T, error) {
	return t.Get()
}

func (t *typedContainer[T]) MustResolve() T {
	return t.MustGet()
}
