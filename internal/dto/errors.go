package dto

import (
	"errors"
	"fmt"
	"strings"
)

// ErrorResponse represents a structured error response for the frontend
type ErrorResponse struct {
	Code      string            `json:"code"`
	Message   string            `json:"message"`
	Details   string            `json:"details,omitempty"`
	Fields    map[string]string `json:"fields,omitempty"`
	Technical string            `json:"technical,omitempty"`
}

// Error implements the error interface
func (e *ErrorResponse) Error() string {
	return e.Message
}

// Common error codes
const (
	ErrorCodeValidation      = "VALIDATION_ERROR"
	ErrorCodeDuplicate       = "DUPLICATE_ERROR"
	ErrorCodeNotFound        = "NOT_FOUND"
	ErrorCodeDatabaseError   = "DATABASE_ERROR"
	ErrorCodeConnectionError = "CONNECTION_ERROR"
	ErrorCodeUnknownError    = "UNKNOWN_ERROR"
)

// NewValidationError creates a validation error with field-specific messages
func NewValidationError(message string, fields map[string]string) *ErrorResponse {
	return &ErrorResponse{
		Code:    ErrorCodeValidation,
		Message: message,
		Fields:  fields,
	}
}

// NewDuplicateError creates a duplicate error (e.g., unique constraint violation)
func NewDuplicateError(resource, field, value string) *ErrorResponse {
	return &ErrorResponse{
		Code:    ErrorCodeDuplicate,
		Message: fmt.Sprintf("%s with this %s already exists", resource, field),
		Details: fmt.Sprintf("Value '%s' is already in use", value),
	}
}

// NewNotFoundError creates a not found error
func NewNotFoundError(resource string) *ErrorResponse {
	return &ErrorResponse{
		Code:    ErrorCodeNotFound,
		Message: fmt.Sprintf("%s not found", resource),
	}
}

// NewDatabaseError creates a database error with user-friendly message
func NewDatabaseError(message string, technical string) *ErrorResponse {
	return &ErrorResponse{
		Code:      ErrorCodeDatabaseError,
		Message:   message,
		Technical: technical,
	}
}

// NewConnectionError creates a connection error
func NewConnectionError(service string) *ErrorResponse {
	return &ErrorResponse{
		Code:    ErrorCodeConnectionError,
		Message: fmt.Sprintf("Failed to connect to %s", service),
	}
}

// NewUnknownError creates an unknown error
func NewUnknownError(technical string) *ErrorResponse {
	return &ErrorResponse{
		Code:      ErrorCodeUnknownError,
		Message:   "An unexpected error occurred",
		Technical: technical,
	}
}

// TranslateError converts common Go/database errors to user-friendly ErrorResponse
func TranslateError(err error) *ErrorResponse {
	if err == nil {
		return nil
	}

	errStr := err.Error()
	errStrLower := strings.ToLower(errStr)

	// Check for unique constraint violations
	if strings.Contains(errStrLower, "unique constraint") ||
		strings.Contains(errStrLower, "duplicate") ||
		strings.Contains(errStrLower, "already exists") {

		// Try to extract field name from error
		if strings.Contains(errStrLower, "url") {
			return NewDuplicateError("Site", "URL", extractValue(errStr, "url"))
		}
		if strings.Contains(errStrLower, "name") {
			return NewDuplicateError("Site", "name", extractValue(errStr, "name"))
		}
		return NewDuplicateError("Record", "field", "")
	}

	// Check for not found errors
	if strings.Contains(errStrLower, "not found") ||
		strings.Contains(errStrLower, "no rows") {
		if strings.Contains(errStrLower, "site") {
			return NewNotFoundError("Site")
		}
		return NewNotFoundError("Record")
	}

	// Check for connection errors
	if strings.Contains(errStrLower, "connection") ||
		strings.Contains(errStrLower, "timeout") ||
		strings.Contains(errStrLower, "network") {
		return NewConnectionError("database")
	}

	// Check for validation errors
	if strings.Contains(errStrLower, "invalid") ||
		strings.Contains(errStrLower, "required") ||
		strings.Contains(errStrLower, "validate") {
		return &ErrorResponse{
			Code:      ErrorCodeValidation,
			Message:   "Invalid data",
			Details:   err.Error(),
			Technical: err.Error(),
		}
	}

	// Default to unknown error
	return NewUnknownError(err.Error())
}

// extractValue tries to extract a value from error message for a given field
func extractValue(errStr, field string) string {
	// Simple extraction - could be improved with regex
	parts := strings.Split(errStr, "'")
	for i, part := range parts {
		if strings.Contains(strings.ToLower(part), field) && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return ""
}

// WrapError wraps an existing error with additional context
func WrapError(err error, message string) *ErrorResponse {
	if err == nil {
		return nil
	}

	// If it's already an ErrorResponse, preserve it
	var errResp *ErrorResponse
	if errors.As(err, &errResp) {
		return errResp
	}

	// Translate the error and add context
	translated := TranslateError(err)
	if message != "" {
		translated.Details = message + ": " + translated.Details
	}
	return translated
}
