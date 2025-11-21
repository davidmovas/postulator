package fault

import (
	"errors"
	"fmt"
)

type ErrorType string

const (
	ErrorTypeFatal       ErrorType = "fatal"
	ErrorTypeRetryable   ErrorType = "retryable"
	ErrorTypeRecoverable ErrorType = "recoverable"
	ErrorTypeValidation  ErrorType = "validation"
)

type ErrorCode string

const (
	ErrCodeUnknown ErrorCode = "unknown"

	ErrCodeInvalidJob      ErrorCode = "invalid_job"
	ErrCodeInactiveSite    ErrorCode = "inactive_site"
	ErrCodeMissingConfig   ErrorCode = "missing_config"
	ErrCodeInvalidStrategy ErrorCode = "invalid_strategy"

	ErrCodeNoTopics     ErrorCode = "no_topics"
	ErrCodeNoCategories ErrorCode = "no_categories"
	ErrCodeNoProvider   ErrorCode = "no_provider"

	ErrCodePromptRenderFailed ErrorCode = "prompt_render_failed"
	ErrCodeAIGenerationFailed ErrorCode = "ai_generation_failed"
	ErrCodeEmptyContent       ErrorCode = "empty_content"
	ErrCodeInvalidOutput      ErrorCode = "invalid_output"

	ErrCodePublishFailed    ErrorCode = "publish_failed"
	ErrCodeWPClientError    ErrorCode = "wp_client_error"
	ErrCodeArticleSaveError ErrorCode = "article_save_error"

	ErrCodeDatabaseError  ErrorCode = "database_error"
	ErrCodeRecordNotFound ErrorCode = "record_not_found"
	ErrCodeUpdateFailed   ErrorCode = "update_failed"

	ErrCodeNetworkError ErrorCode = "network_error"
	ErrCodeTimeout      ErrorCode = "timeout"
)

type PipelineError struct {
	Type      ErrorType
	Code      ErrorCode
	Message   string
	Step      string
	State     string
	Cause     error
	Context   map[string]any
	Retryable bool
}

func (e *PipelineError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s at step '%s': %s (caused by: %v)",
			e.Type, e.Code, e.Step, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s] %s at step '%s': %s",
		e.Type, e.Code, e.Step, e.Message)
}

func (e *PipelineError) Unwrap() error {
	return e.Cause
}

func (e *PipelineError) WithContext(key string, value interface{}) *PipelineError {
	if e.Context == nil {
		e.Context = make(map[string]interface{})
	}
	e.Context[key] = value
	return e
}

func (e *PipelineError) IsFatal() bool {
	return e.Type == ErrorTypeFatal
}

func (e *PipelineError) IsRetryable() bool {
	return e.Retryable || e.Type == ErrorTypeRetryable
}

func (e *PipelineError) IsRecoverable() bool {
	return e.Type == ErrorTypeRecoverable
}

func NewFatalError(code ErrorCode, step string, message string) *PipelineError {
	return &PipelineError{
		Type:      ErrorTypeFatal,
		Code:      code,
		Step:      step,
		Message:   message,
		Retryable: false,
	}
}

func NewRetryableError(code ErrorCode, step string, message string) *PipelineError {
	return &PipelineError{
		Type:      ErrorTypeRetryable,
		Code:      code,
		Step:      step,
		Message:   message,
		Retryable: true,
	}
}

func NewRecoverableError(code ErrorCode, step string, message string) *PipelineError {
	return &PipelineError{
		Type:      ErrorTypeRecoverable,
		Code:      code,
		Step:      step,
		Message:   message,
		Retryable: false,
	}
}

func NewValidationError(code ErrorCode, step string, message string) *PipelineError {
	return &PipelineError{
		Type:      ErrorTypeValidation,
		Code:      code,
		Step:      step,
		Message:   message,
		Retryable: false,
	}
}

func WrapError(err error, code ErrorCode, step string, message string) *PipelineError {
	var pErr *PipelineError
	if errors.As(err, &pErr) {
		return pErr
	}

	errType := ErrorTypeFatal
	retryable := false

	switch code {
	case ErrCodeNetworkError, ErrCodeTimeout:
		errType = ErrorTypeRetryable
		retryable = true
	case ErrCodeNoTopics, ErrCodeNoCategories:
		errType = ErrorTypeRecoverable
	case ErrCodeInvalidJob, ErrCodeInactiveSite, ErrCodeMissingConfig:
		errType = ErrorTypeValidation
	}

	return &PipelineError{
		Type:      errType,
		Code:      code,
		Step:      step,
		Message:   message,
		Cause:     err,
		Retryable: retryable,
	}
}

type ErrorHandler interface {
	Handle(err *PipelineError) ErrorAction
}

type ErrorAction string

const (
	ActionRetry    ErrorAction = "retry"
	ActionFail     ErrorAction = "fail"
	ActionPause    ErrorAction = "pause"
	ActionContinue ErrorAction = "continue"
	ActionRecover  ErrorAction = "recover"
)

type DefaultErrorHandler struct{}

func (h *DefaultErrorHandler) Handle(err *PipelineError) ErrorAction {
	switch err.Type {
	case ErrorTypeFatal:
		return ActionFail
	case ErrorTypeRetryable:
		return ActionRetry
	case ErrorTypeRecoverable:
		return ActionRecover
	case ErrorTypeValidation:
		return ActionPause
	default:
		return ActionFail
	}
}
