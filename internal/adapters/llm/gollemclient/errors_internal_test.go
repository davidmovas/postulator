package gollemclient

import (
	"context"
	stderrors "errors"
	"net/http"
	"testing"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/gollem-dev/gollem"
	"github.com/sashabaranov/go-openai"

	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func anthropicError(status int, retryAfter string) error {
	header := http.Header{}
	if retryAfter != "" {
		header.Set("Retry-After", retryAfter)
	}
	return &anthropic.Error{StatusCode: status, Response: &http.Response{StatusCode: status, Header: header}}
}

func TestClassify(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		err       error
		want      errors.Code
		wantRetry time.Duration
		retryable bool
	}{
		{name: "nothing to classify"},
		{
			name: "an openai api error",
			err:  &openai.APIError{HTTPStatusCode: http.StatusUnauthorized},
			want: errors.Unauthorized,
		},
		{
			name: "an openai request error",
			err:  &openai.RequestError{HTTPStatusCode: http.StatusBadGateway},
			want: errors.External, retryable: true,
		},
		{name: "a missing model", err: &openai.APIError{HTTPStatusCode: http.StatusNotFound}, want: errors.NotFound},
		{name: "a rejected request", err: &openai.APIError{HTTPStatusCode: http.StatusUnprocessableEntity}, want: errors.Invalid},
		{name: "an unexpected status", err: &openai.APIError{HTTPStatusCode: http.StatusMovedPermanently}, want: errors.External},
		{
			name: "an anthropic rate limit with seconds",
			err:  anthropicError(http.StatusTooManyRequests, "7"),
			want: errors.RateLimited, wantRetry: 7 * time.Second, retryable: true,
		},
		{
			name: "an anthropic rate limit without a hint",
			err:  anthropicError(http.StatusTooManyRequests, ""),
			want: errors.RateLimited, retryable: true,
		},
		{
			name: "an anthropic rate limit with a past date",
			err:  anthropicError(http.StatusTooManyRequests, "Mon, 02 Jan 2006 15:04:05 GMT"),
			want: errors.RateLimited, retryable: true,
		},
		{
			name: "an anthropic rate limit with nonsense",
			err:  anthropicError(http.StatusTooManyRequests, "soon"),
			want: errors.RateLimited, retryable: true,
		},
		{
			name: "an anthropic rate limit with zero seconds",
			err:  anthropicError(http.StatusTooManyRequests, "0"),
			want: errors.RateLimited, retryable: true,
		},
		{name: "a transport failure", err: stderrors.New("dial tcp: refused"), want: errors.External, retryable: true},
		{name: "a deadline", err: context.DeadlineExceeded, want: errors.External, retryable: true},
		{name: "an oversized prompt", err: gollem.ErrTokenSizeExceeded, want: errors.Invalid},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := classify(context.Background(), tc.err)
			if tc.want == "" {
				if got != nil {
					t.Fatalf("classify(nil) = %v, want nil", got)
				}
				return
			}
			if !errors.IsCode(got, tc.want) {
				t.Fatalf("classify = %v (%s), want %s", got, errors.CodeOf(got), tc.want)
			}

			var kernel *errors.Error
			if !stderrors.As(got, &kernel) {
				t.Fatalf("classify returned %T, want a kernel error", got)
			}
			if tc.retryable != (kernel.Retry != nil) {
				t.Fatalf("retry hint = %v, want retryable %t", kernel.Retry, tc.retryable)
			}
			if tc.wantRetry > 0 && kernel.Retry.After != tc.wantRetry {
				t.Errorf("retry after = %s, want %s", kernel.Retry.After, tc.wantRetry)
			}
		})
	}
}

func TestClassifyCancellationWins(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if got := classify(ctx, &openai.APIError{HTTPStatusCode: http.StatusTooManyRequests}); !errors.IsCode(got, errors.Cancelled) {
		t.Fatalf("classify = %v (%s), want %s", got, errors.CodeOf(got), errors.Cancelled)
	}
}

func TestRetryAfterWithoutAResponse(t *testing.T) {
	t.Parallel()

	if got := retryAfter(nil); got != 0 {
		t.Fatalf("retryAfter(nil) = %s, want 0", got)
	}
	future := time.Now().Add(90 * time.Second).UTC().Format(http.TimeFormat)
	resp := &http.Response{Header: http.Header{"Retry-After": []string{future}}}
	if got := retryAfter(resp); got <= 0 || got > 90*time.Second {
		t.Fatalf("retryAfter(future) = %s, want a positive delay", got)
	}
}

func TestAnOpenAIRateLimitIsWaitedOutAsLongAsItAsks(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		err  error
		want time.Duration
	}{
		{
			name: "the seconds the provider named",
			err: &openai.APIError{HTTPStatusCode: http.StatusTooManyRequests, Message: "Rate limit reached for " +
				"gpt-5.6-terra in organization org-1 on tokens per min. Limit: 30000, Used: 29998. " +
				"Please try again in 1.982s. Visit the account page to raise it."},
			want: 1982 * time.Millisecond,
		},
		{
			name: "milliseconds",
			err: &openai.APIError{HTTPStatusCode: http.StatusTooManyRequests,
				Message: "Rate limit reached. Please try again in 20ms."},
			want: 20 * time.Millisecond,
		},
		{
			name: "a compound duration is read whole",
			err: &openai.APIError{HTTPStatusCode: http.StatusTooManyRequests,
				Message: "Rate limit reached. Please try again in 1m30s."},
			want: 90 * time.Second,
		},
		{
			name: "a wait longer than a turn should hold for is left to the backoff",
			err: &openai.APIError{HTTPStatusCode: http.StatusTooManyRequests,
				Message: "Rate limit reached. Please try again in 6m0s."},
		},
		{
			name: "a body the sdk hands back unparsed",
			err: &openai.RequestError{HTTPStatusCode: http.StatusTooManyRequests,
				Body: []byte(`{"error":{"message":"Rate limit reached. Please try again in 3s."}}`)},
			want: 3 * time.Second,
		},
		{
			name: "a refusal that names no delay",
			err: &openai.APIError{HTTPStatusCode: http.StatusTooManyRequests,
				Message: "Rate limit reached for requests"},
		},
		{
			name: "a delay longer than a turn would wait is left to the backoff",
			err: &openai.APIError{HTTPStatusCode: http.StatusTooManyRequests,
				Message: "Rate limit reached. Please try again in 9h."},
		},
		{
			name: "a server error naming a delay is waited out too",
			err: &openai.APIError{HTTPStatusCode: http.StatusServiceUnavailable,
				Message: "The engine is overloaded. Please try again in 12s."},
			want: 12 * time.Second,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := classify(context.Background(), tc.err)

			var kernel *errors.Error
			if !stderrors.As(got, &kernel) || kernel.Retry == nil {
				t.Fatalf("classify = %v, want a retryable kernel error", got)
			}
			if kernel.Retry.After != tc.want {
				t.Fatalf("retry after = %s, want %s", kernel.Retry.After, tc.want)
			}
		})
	}
}

func TestAnAnthropicHeaderBeatsTheSentence(t *testing.T) {
	t.Parallel()

	held := anthropicError(http.StatusTooManyRequests, "7")
	got := classify(context.Background(), held)

	var kernel *errors.Error
	if !stderrors.As(got, &kernel) || kernel.Retry == nil || kernel.Retry.After != 7*time.Second {
		t.Fatalf("classify = %v, want the seven seconds the header named", got)
	}
}

func TestClassifyNamesAnExhaustedOutputBudget(t *testing.T) {
	t.Parallel()

	param := "max_completion_tokens"
	const told = "Could not finish the message because max_tokens or model output limit was reached. " +
		"Please try again with higher max_tokens."

	cases := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "the provider names the parameter",
			err:  &openai.APIError{HTTPStatusCode: http.StatusBadRequest, Param: &param, Message: "no help"},
			want: exhaustedOutput,
		},
		{
			name: "the provider only says it in prose",
			err:  &openai.APIError{HTTPStatusCode: http.StatusBadRequest, Message: told},
			want: exhaustedOutput,
		},
		{
			name: "another bad request keeps the general sentence",
			err:  &openai.APIError{HTTPStatusCode: http.StatusBadRequest, Message: "unknown parameter: frequency_penalty"},
			want: rejectedRequest,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := classify(context.Background(), tc.err)
			if !errors.IsCode(got, errors.Invalid) {
				t.Fatalf("classify = %v (%s), want %s", got, errors.CodeOf(got), errors.Invalid)
			}

			var kernel *errors.Error
			if !stderrors.As(got, &kernel) {
				t.Fatalf("classify returned %T, want a kernel error", got)
			}
			if kernel.Message != tc.want {
				t.Errorf("message = %q, want %q", kernel.Message, tc.want)
			}
		})
	}
}
