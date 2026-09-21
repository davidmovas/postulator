package gollemclient

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/gollem-dev/gollem"
	"github.com/sashabaranov/go-openai"

	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func classify(ctx context.Context, err error) error {
	if err == nil {
		return nil
	}
	if ctx.Err() != nil {
		return errors.New(errors.Cancelled, "the model call was cancelled").WithInternal(err)
	}
	if stderrors.Is(err, context.DeadlineExceeded) {
		return errors.New(errors.External, "the model did not answer before the timeout").WithInternal(err).WithRetry(0)
	}
	if stderrors.Is(err, gollem.ErrTokenSizeExceeded) {
		return errors.New(errors.Invalid, "the prompt does not fit the model context window").WithInternal(err)
	}

	status, after, ok := statusOf(err)
	if !ok {
		return errors.New(errors.External, "the model provider could not be reached").WithInternal(err).WithRetry(0)
	}
	return fromStatus(status, after, err)
}

func fromStatus(status int, after time.Duration, err error) error {
	base := func(code errors.Code, message string) *errors.Error {
		built := errors.New(code, message).WithDetail("status", status).WithInternal(err)
		if told := maskKeys(messageOf(err)); told != "" {
			built = built.WithDetail("providerMessage", told)
		}
		return built
	}

	switch {
	case status == http.StatusUnauthorized:
		return base(errors.Unauthorized, "the model provider rejected the api key")
	case status == http.StatusForbidden:
		return base(errors.Unauthorized, "the key has no access to this model")
	case status == http.StatusNotFound:
		return base(errors.NotFound, "the model provider has no such model")
	case status == http.StatusTooManyRequests:
		return base(errors.RateLimited, "the model provider is rate limiting this key").WithRetry(after)
	case status >= http.StatusInternalServerError:
		return base(errors.External, "the model provider returned a server error").WithRetry(after)
	case status >= http.StatusBadRequest:
		return base(errors.Invalid, "the model provider rejected the request")
	default:
		return base(errors.External, "the model provider returned an unexpected status")
	}
}

const ellipsis = "…"

var keyPattern = regexp.MustCompile(`sk-[A-Za-z0-9_-]{8,}|AIza[A-Za-z0-9_-]{10,}`)

func maskKeys(text string) string {
	return keyPattern.ReplaceAllStringFunc(text, func(token string) string {
		prefix := "sk-"
		if strings.HasPrefix(token, "AIza") {
			prefix = "AIza"
		}
		return prefix + ellipsis + token[len(token)-4:]
	})
}

func messageOf(err error) string {
	var apiErr *openai.APIError
	if stderrors.As(err, &apiErr) {
		return strings.TrimSpace(apiErr.Message)
	}

	var requestErr *openai.RequestError
	if stderrors.As(err, &requestErr) {
		return strings.TrimSpace(envelopeMessage(requestErr.Body))
	}

	var anthropicErr *anthropic.Error
	if stderrors.As(err, &anthropicErr) {
		return strings.TrimSpace(envelopeMessage([]byte(anthropicErr.RawJSON())))
	}
	return ""
}

func envelopeMessage(body []byte) string {
	var envelope struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return ""
	}
	return envelope.Error.Message
}

func statusOf(err error) (status int, after time.Duration, ok bool) {
	var apiErr *openai.APIError
	if stderrors.As(err, &apiErr) {
		return apiErr.HTTPStatusCode, 0, true
	}

	var requestErr *openai.RequestError
	if stderrors.As(err, &requestErr) {
		return requestErr.HTTPStatusCode, 0, true
	}

	var anthropicErr *anthropic.Error
	if stderrors.As(err, &anthropicErr) {
		return anthropicErr.StatusCode, retryAfter(anthropicErr.Response), true
	}
	return 0, 0, false
}

func retryAfter(resp *http.Response) time.Duration {
	if resp == nil {
		return 0
	}

	trimmed := strings.TrimSpace(resp.Header.Get("Retry-After"))
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
	if delay := time.Until(when); delay > 0 {
		return delay
	}
	return 0
}
