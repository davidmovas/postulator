package di

import (
	"fmt"
	"reflect"
)

// Lifecycle describes how instances are created and cached by the container.
//
// Transient: a new instance is created every time the service is resolved.
// Singleton: a single instance is created once and reused across the root container and all scopes.
// Scoped: a single instance per scope; reused for the same scope, isolated between scopes.
//
// Use Scoped for per-request or per-operation lifetimes.
// Use Singleton for shared stateless/stateful services caching resources.
// Use Transient when you always want a fresh instance.
type Lifecycle uint8

const (
	// Transient creates a new instance for every resolution.
	Transient Lifecycle = iota
	// Singleton creates a single instance for the root container (shared with scopes).
	Singleton
	// Scoped creates a single instance per scope (child container).
	Scoped
)

// ProviderFunc is a factory function used to construct a service of type T.
// Signature must be: func(c di.Container) (T, error)
// The container parameter can be used to resolve dependencies of T.
type ProviderFunc[T any] func(container Container) (T, error)

// CloseFunc is a function that will be executed when Container.Close is called.
// Functions are executed in LIFO order (last added, first called).
type CloseFunc func()

// Close represents a resource that can be disposed.
// If a scoped instance implements Close, it will be closed when the scope is closed.
type Close interface {
	Close()
}

// IRegistration describes a registration entry created by helper functions
// (e.g., Provide) capturing the provider, lifecycle, and interface type.
// It is typically passed to Container.Register.
type IRegistration[T any] struct {
	// Provider constructs an instance of T using the container for dependencies.
	Provider ProviderFunc[T]
	// Lifecycle determines caching behavior (Transient/Singleton/Scoped).
	Lifecycle Lifecycle
	// Interface is the reflect.Type of the interface being registered.
	Interface reflect.Type
}

// Container is a minimalistic dependency injection container.
//
// Typical usage:
//
//	c := di.New()
//	c.MustRegister(di.Provide[Svc](newSvc, di.Singleton))
//	var svc Svc
//	if err := c.Resolve(&svc); err != nil { /* handle */ }
//	svc.Do()
type Container interface {
	// Scope creates a child container that shares registrations and singletons
	// with the parent, but maintains its own scoped instances.
	// Use this for request/operation scope where Scoped lifecycles should be unique.
	//
	// Example:
	//  root := di.New()
	//  scoped := root.Scope()
	//  defer scoped.Close() // ensures scoped instances are disposed
	Scope() Container

	// Register adds one or more service registrations to the container.
	// Each registration should be created via a helper (e.g., di.Provide) that
	// captures the interface type, provider function, and lifecycle.
	//
	// Provider signature must be: func(c di.Container) (T, error)
	// where T matches the registered interface type.
	//
	// Example:
	//  // Assume type Service is an interface and newService is a constructor.
	//  // di.Provide is a helper producing *Registration.
	//  err := c.Register(
	//      di.Provide[Service](newService, di.Singleton),
	//  )
	//  if err != nil { /* handle */ }
	Register(registrations ...any) error

	// MustRegister is like Register but panics on error.
	// Prefer Register in libraries; use MustRegister in main/test setup for brevity.
	//
	// Example:
	//  c.MustRegister(di.Provide[Service](newService, di.Singleton))
	MustRegister(registrations ...any)

	// Resolve populates the given pointer with an instance of its element type.
	// The type must have a registration; lifecycle rules (Singleton/Scoped/Transient)
	// are applied when creating/caching the instance.
	//
	// Example:
	//  var svc Service
	//  if err := c.Resolve(&svc); err != nil { return err }
	//  svc.DoWork()
	Resolve(target any) error

	// MustResolve is like Resolve but panics on error.
	// Useful in bootstrap code where failures are unrecoverable.
	//
	// Example:
	//  var svc Service
	//  c.MustResolve(&svc)
	MustResolve(target any)

	// AddCloseFunc registers a cleanup function to be called when Close() is invoked.
	// Functions are executed in LIFO order (last added, first called).
	//
	// Example:
	//  c.AddCloseFunc(func(){ _ = logger.Sync() })
	AddCloseFunc(cleanupFunc CloseFunc)

	// Close runs all registered cleanup functions (LIFO order) and disposes
	// scoped instances that implement Close. It is safe to call Close multiple times;
	// subsequent calls are no-ops.
	//
	// Example:
	//  scoped := root.Scope()
	//  defer scoped.Close() // ensures per-scope resources are freed
	Close()
}

// TypedContainer is a helper wrapper that enables type-safe operations
// for a specific generic type without repeatedly passing type info.
type TypedContainer[T any] struct {
	container Container
}

// For creates a generic resolver bound to the underlying container.
// Example:
//
//	t := di.Typed[MyService](c)
//	r := t.For()
//	svc := r.MustGet()
func (t TypedContainer[T]) For() *TypedResolver[T] {
	return &TypedResolver[T]{
		container: t.container,
	}
}

// TypedResolver provides typed helpers to resolve T from the container.
type TypedResolver[T any] struct {
	container Container
}

// Get resolves T and returns it, or an error if resolution fails.
func (t *TypedResolver[T]) Get() (T, error) {
	var zero T
	target := new(T)
	if err := t.container.Resolve(target); err != nil {
		return zero, err
	}
	return *target, nil
}

// MustGet resolves T and panics on error. Useful for bootstrap/tests.
func (t *TypedResolver[T]) MustGet() T {
	instance, err := t.Get()
	if err != nil {
		panic(fmt.Sprintf("di: failed to get: %v", err))
	}
	return instance
}
