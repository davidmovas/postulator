package dto

import (
	"errors"
	"fmt"
	"time"

	appErrors "github.com/davidmovas/postulator/pkg/errors"
)

const timeFormat = "2006-01-02 15:04:05"

type Response[T any] struct {
	Success bool   `json:"success"`
	Data    T      `json:"data,omitempty"`
	Error   *Error `json:"error,omitempty"`
}

type PaginatedResponse[T any] struct {
	Success bool   `json:"success"`
	Items   []T    `json:"items,omitempty"`
	Total   int    `json:"total,omitempty"`
	Limit   int    `json:"limit,omitempty"`
	Offset  int    `json:"offset,omitempty"`
	HasMore bool   `json:"hasMore,omitempty"`
	Error   *Error `json:"error,omitempty"`
}

type Error struct {
	Code         string         `json:"code"`
	Message      string         `json:"message"`
	UserMessage  string         `json:"userMessage,omitempty"`
	Context      map[string]any `json:"context,omitempty"`
	IsUserFacing bool           `json:"isUserFacing"`
}

func Success[T any](data T) *Response[T] {
	return &Response[T]{Success: true, Data: data}
}

func Fail[T any](err error) *Response[T] {
	return &Response[T]{Success: false, Error: toError(err)}
}

func PaginatedSuccess[T any](items []T, total, limit, offset int) *PaginatedResponse[T] {
	return &PaginatedResponse[T]{
		Success: true,
		Items:   items,
		Total:   total,
		Limit:   limit,
		Offset:  offset,
		HasMore: offset+len(items) < total,
	}
}

func PaginatedFail[T any](err error) *PaginatedResponse[T] {
	return &PaginatedResponse[T]{Success: false, Error: toError(err)}
}

func toError(err error) *Error {
	if err == nil {
		return nil
	}

	var appErr *appErrors.AppError
	if errors.As(err, &appErr) {
		return &Error{
			Code:         string(appErr.Code),
			Message:      appErr.Error(),
			UserMessage:  getUserMessage(appErr),
			Context:      appErr.Context,
			IsUserFacing: appErr.IsUserFacing(),
		}
	}

	return &Error{
		Code:         string(appErrors.ErrCodeInternal),
		Message:      err.Error(),
		UserMessage:  "An unexpected error occurred",
		IsUserFacing: false,
	}
}

func getUserMessage(err *appErrors.AppError) string {
	if !err.IsUserFacing() {
		return "An unexpected error occurred. Please try again later."
	}
	// Include internal error details for user-facing errors (like AI errors)
	if err.InternalError != nil {
		return fmt.Sprintf("%s: %v", err.Message, err.InternalError)
	}
	return err.Message
}

func TimeToString(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(timeFormat)
}

func StringToTime(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, nil
	}

	t, err := time.Parse(timeFormat, s)
	if err != nil {
		return time.Time{}, appErrors.Validation(fmt.Sprintf("Invalid date format: %s", s))
	}

	return t, nil
}
