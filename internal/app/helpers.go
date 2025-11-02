package app

import (
	"errors"

	"github.com/davidmovas/postulator/internal/dto"
	appErrors "github.com/davidmovas/postulator/pkg/errors"
)

func dtoErr[T any](err *appErrors.AppError) *dto.Response[T] {
	if err == nil {
		return &dto.Response[T]{Success: true}
	}
	return &dto.Response[T]{
		Success: false,
		Error:   &dto.Error{Code: string(err.Code), Message: err.Message, Context: err.Context},
	}
}

func dtoPagErr[T any](err *appErrors.AppError) *dto.PaginatedResponse[T] {
	if err == nil {
		return &dto.PaginatedResponse[T]{Success: true}
	}
	return &dto.PaginatedResponse[T]{
		Success: false,
		Error:   &dto.Error{Code: string(err.Code), Message: err.Message, Context: err.Context},
	}
}

func asAppErr(err error) *appErrors.AppError {
	if err == nil {
		return nil
	}
	var ae *appErrors.AppError
	if errors.As(err, &ae) {
		return ae
	}
	return appErrors.Internal(err)
}
