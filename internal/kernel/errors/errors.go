package errors

import (
	stderrors "errors"
	"maps"
	"strings"
	"time"
)

const internalMessage = "unexpected internal error"

type RetryInfo struct {
	After time.Duration
}

type Error struct {
	internal error
	Details  map[string]any
	Retry    *RetryInfo
	Code     Code
	Message  string
}

func New(code Code, message string) *Error {
	return &Error{Code: code, Message: message}
}

func Wrap(err error, code Code, message string) error {
	if err == nil {
		return nil
	}
	return &Error{Code: code, Message: message, internal: err}
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
