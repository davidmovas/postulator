package handlers

import (
	"github.com/davidmovas/postulator/internal/dto"
	"github.com/davidmovas/postulator/pkg/errors"
)

func ok[T any](data T) *dto.Response[T] {
	return dto.Success(data)
}

func fail[T any](err error) *dto.Response[T] {
	return dto.Fail[T](err)
}

func paginated[T any](items []T, total, limit, offset int) *dto.PaginatedResponse[T] {
	return dto.PaginatedSuccess(items, total, limit, offset)
}

func paginatedErr[T any](err error) *dto.PaginatedResponse[T] {
	return dto.PaginatedFail[T](err)
}

func validate(condition bool, message string) error {
	if !condition {
		return errors.Validation(message)
	}
	return nil
}
