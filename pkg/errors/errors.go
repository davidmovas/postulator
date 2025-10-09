package errors

import (
	"fmt"
)

type ErrorCode string

const (
	ErrCodeInternal      ErrorCode = "INTERNAL"
	ErrCodeValidation    ErrorCode = "VALIDATION"
	ErrCodeNotFound      ErrorCode = "NOT_FOUND"
	ErrCodeAlreadyExists ErrorCode = "ALREADY_EXISTS"

	ErrCodeDatabase ErrorCode = "DATABASE"

	ErrCodeSiteUnreachable ErrorCode = "SITE_UNREACHABLE"
	ErrCodeSiteAuth        ErrorCode = "SITE_AUTH"

	ErrCodeWordPress ErrorCode = "WORDPRESS"

	ErrCodeAI          ErrorCode = "AI"
	ErrCodeAIRateLimit ErrorCode = "AI_RATE_LIMIT"

	ErrCodeImport ErrorCode = "IMPORT"

	ErrCodeJobExecution ErrorCode = "JOB_EXECUTION"
	ErrCodeScheduler    ErrorCode = "SCHEDULER"
)

type AppError struct {
	Code          ErrorCode
	Message       string
	InternalError error
	Context       map[string]any
}

func (e *AppError) Error() string {
	if e.InternalError != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.InternalError)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

func (e *AppError) Unwrap() error {
	return e.InternalError
}

func (e *AppError) IsUserFacing() bool {
	switch e.Code {
	case ErrCodeInternal, ErrCodeDatabase:
		return false
	default:
		return true
	}
}

func New(code ErrorCode, message string) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Context: make(map[string]any),
	}
}

func Wrap(code ErrorCode, message string, err error) *AppError {
	return &AppError{
		Code:          code,
		Message:       message,
		InternalError: err,
		Context:       make(map[string]any),
	}
}

func (e *AppError) WithContext(key string, value any) *AppError {
	e.Context[key] = value
	return e
}

func Internal(err error) *AppError {
	return Wrap(ErrCodeInternal, "Internal application error", err)
}

func Validation(message string) *AppError {
	return New(ErrCodeValidation, message)
}

func NotFound(entity string, identifier any) *AppError {
	return New(ErrCodeNotFound, fmt.Sprintf("%s with identifier %v not found", entity, identifier))
}

func AlreadyExists(entity string) *AppError {
	return New(ErrCodeAlreadyExists, fmt.Sprintf("%s already exists", entity))
}

func Database(err error) *AppError {
	return Wrap(ErrCodeDatabase, "Database error", err)
}

func SiteUnreachable(siteURL string, err error) *AppError {
	return Wrap(ErrCodeSiteUnreachable, fmt.Sprintf("Site %s is unreachable", siteURL), err).
		WithContext("url", siteURL)
}

func SiteAuth(siteURL string) *AppError {
	return New(ErrCodeSiteAuth, fmt.Sprintf("Authentication failed for site %s", siteURL)).
		WithContext("url", siteURL)
}

func WordPress(operation string, err error) *AppError {
	return Wrap(ErrCodeWordPress, fmt.Sprintf("WordPress error: %s", operation), err).
		WithContext("operation", operation)
}

func AI(provider string, err error) *AppError {
	return Wrap(ErrCodeAI, fmt.Sprintf("AI provider %s error", provider), err).
		WithContext("provider", provider)
}

func AIRateLimit(provider string) *AppError {
	return New(ErrCodeAIRateLimit, fmt.Sprintf("Rate limit exceeded for %s", provider)).
		WithContext("provider", provider)
}

func Import(format string, err error) *AppError {
	return Wrap(ErrCodeImport, fmt.Sprintf("Import error from format %s", format), err).
		WithContext("format", format)
}

func JobExecution(jobID int64, err error) *AppError {
	return Wrap(ErrCodeJobExecution, "Job execution error", err).
		WithContext("job_id", jobID)
}

func Scheduler(err error) *AppError {
	return Wrap(ErrCodeScheduler, "Task scheduler error", err)
}
