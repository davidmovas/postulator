package di

import "reflect"

type Registration[T any] struct {
	Provider      ProviderFunc[T]
	Lifecycle     Lifecycle
	InterfaceType reflect.Type
	CloseFn       func(T) error
}

func For[T any](provider ProviderFunc[T]) *Registration[T] {
	var zero T
	return &Registration[T]{
		Provider:      provider,
		Lifecycle:     Transient,
		InterfaceType: reflect.TypeOf(zero),
	}
}

func (r *Registration[T]) AsSingleton() *Registration[T] {
	r.Lifecycle = Singleton
	return r
}

func (r *Registration[T]) AsScoped() *Registration[T] {
	r.Lifecycle = Scoped
	return r
}

func (r *Registration[T]) AsTransient() *Registration[T] {
	r.Lifecycle = Transient
	return r
}

func Instance[T any](instance T) *Registration[T] {
	return For[T](func(Container) (T, error) {
		return instance, nil
	}).AsSingleton()
}

func Must[T any](instance T) ProviderFunc[T] {
	return func(Container) (T, error) {
		return instance, nil
	}
}

func (r *Registration[T]) WithClose(closeFn func(T) error) *Registration[T] {
	r.CloseFn = closeFn
	return r
}
