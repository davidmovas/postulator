package openai_test

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/llm/openai"
	"github.com/davidmovas/postulator/internal/adapters/llm/openai/openaitest"
	port "github.com/davidmovas/postulator/internal/application/llm"
	"github.com/davidmovas/postulator/internal/kernel/clock"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func fault(kind, code, message string) openaitest.Fault {
	return openaitest.Fault{Type: kind, Code: code, Message: message}
}

func TestEveryRefusalBecomesAKernelError(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		reply     openaitest.Reply
		want      errors.Code
		message   string
		details   map[string]any
		retryable bool
		after     time.Duration
	}{
		{
			name:    "a rejected key",
			reply:   openaitest.Failure(http.StatusUnauthorized, fault("invalid_request_error", "invalid_api_key", "Incorrect API key provided.")),
			want:    errors.Unauthorized,
			message: "the model provider rejected the api key",
			details: map[string]any{"status": http.StatusUnauthorized, "code": "invalid_api_key", "providerMessage": "Incorrect API key provided."},
		},
		{
			name:    "a key without access",
			reply:   openaitest.Failure(http.StatusForbidden, fault("invalid_request_error", "", "You are not allowed to sample from this model.")),
			want:    errors.Unauthorized,
			message: "the key has no access to this model",
		},
		{
			name:    "an unknown model",
			reply:   openaitest.Failure(http.StatusNotFound, fault("invalid_request_error", "model_not_found", "The model `gpt-nope-1` does not exist or you do not have access to it.")),
			want:    errors.NotFound,
			message: "the model provider has no such model",
			details: map[string]any{"code": "model_not_found", "model": "openai:gpt-5.6-terra"},
		},
		{
			name:    "something else not found",
			reply:   openaitest.Failure(http.StatusNotFound, fault("invalid_request_error", "", "Item with id 'rs_1' not found.")),
			want:    errors.NotFound,
			message: "the model provider could not find what the request named",
		},
		{
			name: "a refused parameter",
			reply: openaitest.Failure(http.StatusBadRequest, openaitest.Fault{
				Type: "invalid_request_error", Code: "unsupported_value", Param: "reasoning.effort",
				Message: "Unsupported value: 'minimal' is not supported with the 'gpt-5.6-luna' model.",
			}),
			want:    errors.Invalid,
			message: "the model provider rejected the request",
			details: map[string]any{"status": http.StatusBadRequest, "param": "reasoning.effort", "code": "unsupported_value"},
		},
		{
			name:    "a prompt over the window",
			reply:   openaitest.Failure(http.StatusBadRequest, fault("invalid_request_error", "context_length_exceeded", "Your input exceeds the context window of this model.")),
			want:    errors.Invalid,
			message: "the prompt does not fit the model context window",
		},
		{
			name:    "an unprocessable request",
			reply:   openaitest.Failure(http.StatusUnprocessableEntity, fault("invalid_request_error", "", "no")),
			want:    errors.Invalid,
			message: "the model provider rejected the request",
		},
		{
			name:      "a request the provider timed out",
			reply:     openaitest.Failure(http.StatusRequestTimeout, fault("invalid_request_error", "", "Request timed out.")),
			want:      errors.External,
			retryable: true,
		},
		{
			name:      "a conflict the provider asks to retry",
			reply:     openaitest.Failure(http.StatusConflict, fault("invalid_request_error", "", "The request conflicted with another.")),
			want:      errors.External,
			retryable: true,
		},
		{
			name:    "the probed credit refusal",
			reply:   openaitest.QuotaExhausted(),
			want:    errors.NeedsHuman,
			message: quotaMessage,
			details: map[string]any{"status": http.StatusTooManyRequests, "code": "credit_balance_exhausted"},
		},
		{
			name:    "the legacy quota refusal",
			reply:   openaitest.Failure(http.StatusTooManyRequests, fault("insufficient_quota", "insufficient_quota", "You exceeded your current quota.")),
			want:    errors.NeedsHuman,
			message: quotaMessage,
		},
		{
			name:    "an organization spend limit",
			reply:   openaitest.Failure(http.StatusTooManyRequests, fault("insufficient_quota", "organization_spend_limit_exceeded", "limit")),
			want:    errors.NeedsHuman,
			message: quotaMessage,
		},
		{
			name:    "a project spend limit",
			reply:   openaitest.Failure(http.StatusTooManyRequests, fault("insufficient_quota", "project_spend_limit_exceeded", "limit")),
			want:    errors.NeedsHuman,
			message: quotaMessage,
		},
		{
			name:    "an organization usage limit",
			reply:   openaitest.Failure(http.StatusTooManyRequests, fault("insufficient_quota", "organization_usage_limit_exceeded", "limit")),
			want:    errors.NeedsHuman,
			message: quotaMessage,
		},
		{
			name:      "a rate limit that names its wait",
			reply:     openaitest.Failure(http.StatusTooManyRequests, openaitest.RateLimit()),
			want:      errors.RateLimited,
			message:   "the model provider is rate limiting this key",
			retryable: true, after: 1982 * time.Millisecond,
		},
		{
			name:      "a rate limit with a header in milliseconds",
			reply:     openaitest.Failure(http.StatusTooManyRequests, fault("rate_limit_error", "rate_limit_exceeded", "slow down")).WithHeader("retry-after-ms", "1500"),
			want:      errors.RateLimited,
			retryable: true, after: 1500 * time.Millisecond,
		},
		{
			name:      "a rate limit with a header in seconds",
			reply:     openaitest.Failure(http.StatusTooManyRequests, openaitest.RateLimit()).WithHeader("Retry-After", "7"),
			want:      errors.RateLimited,
			retryable: true, after: 7 * time.Second,
		},
		{
			name: "a rate limit with an emptied bucket",
			reply: openaitest.Failure(http.StatusTooManyRequests, fault("rate_limit_error", "rate_limit_exceeded", "slow down")).
				WithHeader("x-ratelimit-remaining-tokens", "0").WithHeader("x-ratelimit-reset-tokens", "20s").
				WithHeader("x-ratelimit-remaining-requests", "59").WithHeader("x-ratelimit-reset-requests", "6m0s"),
			want:      errors.RateLimited,
			retryable: true, after: 20 * time.Second,
		},
		{
			name:      "a flex capacity refusal off flex is a rate limit",
			reply:     openaitest.FlexCapacity(),
			want:      errors.RateLimited,
			retryable: true,
		},
		{
			name:      "a server error",
			reply:     openaitest.Failure(http.StatusInternalServerError, openaitest.ServerError()),
			want:      errors.External,
			message:   "the model provider returned a server error",
			retryable: true,
		},
		{
			name:      "an overloaded server that names its wait",
			reply:     openaitest.Failure(http.StatusServiceUnavailable, fault("service_unavailable_error", "", "Overloaded.")).WithHeader("Retry-After", "12"),
			want:      errors.External,
			retryable: true, after: 12 * time.Second,
		},
		{
			name:      "a gateway that answers in html",
			reply:     openaitest.Reply{Status: http.StatusBadGateway, Body: "<html>bad gateway</html>"},
			want:      errors.External,
			retryable: true,
			details:   map[string]any{"status": http.StatusBadGateway},
		},
		{
			name:    "a status the provider never sends",
			reply:   openaitest.Reply{Status: http.StatusAccepted, Body: `{}`},
			want:    errors.External,
			message: "the model provider returned an unexpected status",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			server := openaitest.New(t)
			server.Enqueue(tc.reply)

			_, err := newClient(server).Complete(t.Context(), write("hello"))
			if !errors.IsCode(err, tc.want) {
				t.Fatalf("Complete = %v (%s), want %s", err, errors.CodeOf(err), tc.want)
			}
			kernel := kernelOf(t, err)
			if tc.message != "" && kernel.Message != tc.message {
				t.Errorf("message = %q, want %q", kernel.Message, tc.message)
			}
			for key, want := range tc.details {
				if kernel.Details[key] != want {
					t.Errorf("details[%s] = %v, want %v (all: %v)", key, kernel.Details[key], want, kernel.Details)
				}
			}
			if tc.retryable != (kernel.Retry != nil) {
				t.Fatalf("retry = %+v, want retryable %t", kernel.Retry, tc.retryable)
			}
			if tc.retryable && kernel.Retry.After != tc.after {
				t.Errorf("retry after = %s, want %s", kernel.Retry.After, tc.after)
			}
			if len(server.Requests()) != 1 {
				t.Errorf("the client sent %d requests, want one: retrying is the decorator's work", len(server.Requests()))
			}
		})
	}
}

const quotaMessage = "the OpenAI account is out of credit or over its spending limit; add credits or raise the limit in the OpenAI billing settings, then try again"

func TestARetryDateIsReadAgainstTheClock(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 10, 3, 17, 43, 30, 0, time.UTC)
	server := openaitest.New(t)
	server.Enqueue(openaitest.Failure(http.StatusTooManyRequests, openaitest.RateLimit()).
		WithHeader("Retry-After", now.Add(45*time.Second).Format(http.TimeFormat)))

	_, err := newClient(server, openai.WithClock(clock.NewFake(now))).Complete(t.Context(), write("hello"))
	if after := kernelOf(t, err).Retry; after == nil || after.After != 45*time.Second {
		t.Fatalf("retry = %+v, want the 45 seconds the date names", after)
	}
}

func TestAKeyQuotedByTheProviderIsMasked(t *testing.T) {
	t.Parallel()

	const leaked = "sk-proj-AbCdEfGhIjKlMnOpQrStUvWx1234"
	server := openaitest.New(t)
	server.Enqueue(openaitest.Failure(http.StatusUnauthorized, fault("invalid_request_error", "invalid_api_key",
		"Incorrect API key provided: "+leaked+". You can find your API key at https://platform.openai.com/account/api-keys.")))

	_, err := newClient(server).Complete(t.Context(), write("hello"))
	kernel := kernelOf(t, err)
	told, ok := kernel.Details["providerMessage"].(string)
	if !ok {
		t.Fatalf("details = %v, want the provider's sentence", kernel.Details)
	}
	if strings.Contains(told, leaked) || strings.Contains(err.Error(), leaked) {
		t.Errorf("the key reached the error: %q / %q", told, err.Error())
	}
	if !strings.Contains(told, "sk-…1234") {
		t.Errorf("provider message = %q, want the key masked to its last four", told)
	}
}

func TestAnExhaustedOutputBudgetIsATruncatedAnswer(t *testing.T) {
	t.Parallel()

	server := openaitest.New(t)
	server.Enqueue(openaitest.Failure(http.StatusBadRequest, fault("invalid_request_error", "",
		"Could not finish the message because max_tokens or model output limit was reached. Please try again with higher max_tokens.")))

	req := write("hello")
	req.MaxTokens = 64
	resp, err := newClient(server).Complete(t.Context(), req)
	if err != nil {
		t.Fatalf("Complete = %v, want a truncated answer rather than a refusal", err)
	}
	if resp.FinishReason != port.FinishLength || resp.Text != "" {
		t.Errorf("response = %+v, want an empty answer that stopped for length", resp)
	}
}

func TestAnOutputCeilingTheProviderRefusesIsNotATruncation(t *testing.T) {
	t.Parallel()

	server := openaitest.New(t)
	server.Enqueue(openaitest.Failure(http.StatusBadRequest, openaitest.Fault{
		Type: "invalid_request_error", Code: "integer_below_min_value", Param: "max_output_tokens",
		Message: "Invalid 'max_output_tokens': integer below minimum value. Expected a value >= 16, but got 15 instead.",
	}))

	_, err := newClient(server).Complete(t.Context(), write("hello"))
	if !errors.IsCode(err, errors.Invalid) {
		t.Fatalf("Complete = %v, want %s", err, errors.Invalid)
	}
}
