package dto

import (
	appErrors "Postulator/pkg/errors"
	"errors"
)

type Response[T any] struct {
	Success bool   `json:"success"`
	Data    T      `json:"data,omitempty"`
	Error   *Error `json:"error,omitempty"`
}

type PaginatedResponse[T any] struct {
	Success bool   `json:"success"`
	Items   []T    `json:"items"`
	Total   int    `json:"total"`
	Limit   int    `json:"limit"`
	Offset  int    `json:"offset"`
	HasMore bool   `json:"hasMore"`
	Error   *Error `json:"error,omitempty"`
}

type Error struct {
	Code         string         `json:"code"`
	Message      string         `json:"message"`
	UserMessage  string         `json:"userMessage,omitempty"`
	Context      map[string]any `json:"context,omitempty"`
	IsUserFacing bool           `json:"isUserFacing"`
}

func NewResponse[T any](data T) *Response[T] {
	return &Response[T]{
		Success: true,
		Data:    data,
	}
}

func NewErrorResponse[T any](err error) *Response[T] {
	return &Response[T]{
		Success: false,
		Error:   toError(err),
	}
}

func NewPaginatedResponse[T any](items []T, total, limit, offset int) *PaginatedResponse[T] {
	return &PaginatedResponse[T]{
		Success: true,
		Items:   items,
		Total:   total,
		Limit:   limit,
		Offset:  offset,
		HasMore: offset+len(items) < total,
	}
}

func NewPaginatedError[T any](err error) *PaginatedResponse[T] {
	return &PaginatedResponse[T]{
		Success: false,
		Error:   toError(err),
	}
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
	return err.Message
}
