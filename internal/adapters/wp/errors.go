package wp

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/davidmovas/postulator/internal/kernel/errors"
)

type wpError struct {
	Code        string `json:"code"`
	Message     string `json:"message"`
	CurrentHash string `json:"currentHash"`
}

func decodeError(body []byte) wpError {
	var failure wpError
	if err := json.Unmarshal(body, &failure); err != nil {
		return wpError{}
	}
	return failure
}

func classify(resp *http.Response, body []byte) error {
	failure := decodeError(body)

	base := errors.New(errors.External, "the WordPress site returned an unexpected status").
		WithDetail("status", resp.StatusCode).
		WithDetail("method", resp.Request.Method).
		WithDetail("path", resp.Request.URL.Path)
	if failure.Code != "" {
		base = base.WithDetail("code", failure.Code)
	}
	if failure.Message != "" {
		base = base.WithDetail("wpMessage", failure.Message)
	}

	switch {
	case resp.StatusCode == http.StatusUnauthorized, resp.StatusCode == http.StatusForbidden:
		return recode(base, errors.Unauthorized, "WordPress rejected the application password")
	case resp.StatusCode == http.StatusNotFound:
		return recode(base, errors.NotFound, "WordPress has no such resource")
	case resp.StatusCode == http.StatusConflict:
		conflict := recode(base, errors.Conflict, "the WordPress content changed since it was read")
		if failure.CurrentHash != "" {
			conflict = conflict.WithDetail("currentHash", failure.CurrentHash)
		}
		return conflict
	case resp.StatusCode == http.StatusTooManyRequests:
		return recode(base, errors.RateLimited, "WordPress is rate limiting this site").
			WithRetry(retryAfter(resp.Header.Get("Retry-After")))
	case resp.StatusCode == http.StatusBadRequest:
		return recode(base, errors.Invalid, "WordPress rejected the request")
	case resp.StatusCode >= http.StatusInternalServerError:
		return recode(base, errors.External, "the WordPress site returned a server error").WithRetry(0)
	default:
		return base
	}
}

func recode(base *errors.Error, code errors.Code, message string) *errors.Error {
	next := errors.New(code, message)
	for key, value := range base.Details {
		next = next.WithDetail(key, value)
	}
	return next
}

func transportError(ctx context.Context, err error) error {
	if ctx.Err() != nil {
		return errors.New(errors.Cancelled, "the WordPress request was cancelled").WithInternal(ctx.Err())
	}
	return errors.New(errors.External, "the WordPress site could not be reached").WithInternal(err).WithRetry(0)
}

func retryAfter(header string) time.Duration {
	trimmed := strings.TrimSpace(header)
	if trimmed == "" {
		return 0
	}

	if seconds, err := strconv.Atoi(trimmed); err == nil {
		if seconds <= 0 {
			return 0
		}
		return time.Duration(seconds) * time.Second
	}

	when, err := http.ParseTime(trimmed)
	if err != nil {
		return 0
	}
	delay := time.Until(when)
	if delay <= 0 {
		return 0
	}
	return delay
}

func retryable(err error) bool {
	code := errors.CodeOf(err)
	return code == errors.RateLimited || code == errors.External
}

func delayFor(err error, fallback time.Duration) time.Duration {
	var kernel *errors.Error
	if stderrors.As(err, &kernel) && kernel != nil && kernel.Retry != nil && kernel.Retry.After > 0 {
		return kernel.Retry.After
	}
	return fallback
}

func detailValue(err error, key string) (any, bool) {
	var kernel *errors.Error
	if !stderrors.As(err, &kernel) || kernel == nil {
		return nil, false
	}
	value, ok := kernel.Details[key]
	return value, ok
}

func detailString(err error, key string) string {
	value, ok := detailValue(err, key)
	if !ok {
		return ""
	}
	text, ok := value.(string)
	if !ok {
		return ""
	}
	return text
}
