package dbx

type Result[T any] struct {
	value T
	err   error
}

func Ok[T any](value T) Result[T] {
	return Result[T]{value: value}
}

func Err[T any](err error) Result[T] {
	return Result[T]{err: err}
}

func From[T any](value T, err error) Result[T] {
	return Result[T]{value: value, err: err}
}

func (r Result[T]) NotFound(replacement error) Result[T] {
	if r.err != nil && IsNotFound(r.err) {
		return Result[T]{err: replacement}
	}
	return r
}

func (r Result[T]) Conflict(replacement error) Result[T] {
	if r.err != nil && IsConflict(r.err) {
		return Result[T]{err: replacement}
	}
	return r
}

func (r Result[T]) WrapErr(wrapper func(error) error) Result[T] {
	if r.err == nil || isKernel(r.err) {
		return r
	}
	return Result[T]{err: wrapper(r.err)}
}

func (r Result[T]) Unwrap() (T, error) {
	return r.value, r.err
}
