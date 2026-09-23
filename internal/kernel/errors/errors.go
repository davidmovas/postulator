package errors

import (
	stderrors "errors"
	"maps"
	"runtime"
	"strings"
	"time"
)

const (
	maxStackDepth   = 32
	stackSkipFrames = 3

	internalMessage = "unexpected internal error"
)

type RetryInfo struct {
	After time.Duration
}

type Frame struct {
	Function string
	File     string
	Line     int
}

type Error struct {
	internal error
	Details  map[string]any
	Retry    *RetryInfo
	Code     Code
	Message  string
	stack    []uintptr
}

func New(code Code, message string) *Error {
	return newError(code, message, nil)
}

func Wrap(err error, code Code, message string) error {
	if err == nil {
		return nil
	}
	return newError(code, message, err)
}

func newError(code Code, message string, internal error) *Error {
	var pcs [maxStackDepth]uintptr
	n := runtime.Callers(stackSkipFrames, pcs[:])

	return &Error{
		Code:     code,
		Message:  message,
		internal: internal,
		stack:    pcs[:n],
	}
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	if e.internal == nil {
		return e.Message
	}

	var sb strings.Builder
	sb.WriteString(e.Message)
	sb.WriteString(": ")
	sb.WriteString(e.internal.Error())
	return sb.String()
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.internal
}

func (e *Error) Is(target error) bool {
	if e == nil {
		return false
	}
	var other *Error
	if stderrors.As(target, &other) {
		return e.Code == other.Code
	}
	return false
}

func (e *Error) WithInternal(err error) *Error {
	next := e.clone()
	next.internal = err
	return next
}

func (e *Error) WithRetry(after time.Duration) *Error {
	next := e.clone()
	next.Retry = &RetryInfo{After: after}
	return next
}

func (e *Error) WithDetail(key string, value any) *Error {
	next := e.clone()
	next.Details = make(map[string]any, len(e.Details)+1)
	maps.Copy(next.Details, e.Details)
	next.Details[key] = value
	return next
}

func (e *Error) clone() *Error {
	copied := *e
	return &copied
}

func CodeOf(err error) Code {
	if err == nil {
		return ""
	}
	var kernel *Error
	if stderrors.As(err, &kernel) && kernel != nil {
		return kernel.Code
	}
	return Internal
}

func Describe(err error) (code Code, message string) {
	if err == nil {
		return "", ""
	}

	var kernel *Error
	if !stderrors.As(err, &kernel) || kernel == nil {
		return Internal, internalMessage
	}
	return kernel.Code, kernel.Message
}

func IsCode(err error, code Code) bool {
	if err == nil {
		return false
	}
	return CodeOf(err) == code
}

func Stack(err error) []Frame {
	var kernel *Error
	if !stderrors.As(err, &kernel) || kernel == nil || len(kernel.stack) == 0 {
		return nil
	}

	frames := runtime.CallersFrames(kernel.stack)
	out := make([]Frame, 0, len(kernel.stack))
	for {
		frame, more := frames.Next()
		out = append(out, Frame{Function: frame.Function, File: frame.File, Line: frame.Line})
		if !more {
			return out
		}
	}
}
