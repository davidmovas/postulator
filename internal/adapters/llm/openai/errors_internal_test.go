package openai

import (
	"net/http"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/domain/llm"
)

func TestTheRetryDelayIsReadFromWhateverTheProviderGives(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 10, 3, 17, 43, 30, 0, time.UTC)
	header := func(pairs ...string) http.Header {
		built := http.Header{}
		for i := 0; i+1 < len(pairs); i += 2 {
			built.Set(pairs[i], pairs[i+1])
		}
		return built
	}

	cases := []struct {
		name    string
		header  http.Header
		message string
		want    time.Duration
	}{
		{name: "nothing said"},
		{name: "milliseconds win", header: header("retry-after-ms", "1500", "Retry-After", "7"), want: 1500 * time.Millisecond},
		{name: "fractional milliseconds", header: header("retry-after-ms", "250.5"), want: 250500 * time.Microsecond},
		{name: "seconds", header: header("Retry-After", "7"), want: 7 * time.Second},
		{name: "fractional seconds", header: header("Retry-After", "1.5"), want: 1500 * time.Millisecond},
		{name: "a date", header: header("Retry-After", now.Add(90*time.Second).Format(http.TimeFormat)), want: 90 * time.Second},
		{name: "a date in the past", header: header("Retry-After", now.Add(-time.Minute).Format(http.TimeFormat))},
		{name: "nonsense", header: header("Retry-After", "soon", "retry-after-ms", "later")},
		{name: "zero", header: header("Retry-After", "0")},
		{name: "negative", header: header("retry-after-ms", "-5")},
		{
			name:   "the emptied request bucket",
			header: header("x-ratelimit-remaining-requests", "0", "x-ratelimit-reset-requests", "1s", "x-ratelimit-reset-tokens", "6m0s"),
			want:   time.Second,
		},
		{
			name: "both buckets empty waits for the later",
			header: header("x-ratelimit-remaining-requests", "0", "x-ratelimit-reset-requests", "1s",
				"x-ratelimit-remaining-tokens", "0", "x-ratelimit-reset-tokens", "20ms"),
			want: time.Second,
		},
		{
			name:    "a bucket that is not empty says nothing",
			header:  header("x-ratelimit-remaining-tokens", "120", "x-ratelimit-reset-tokens", "6m0s"),
			message: "Please try again in 2s.",
			want:    2 * time.Second,
		},
		{name: "an unreadable reset", header: header("x-ratelimit-remaining-tokens", "0", "x-ratelimit-reset-tokens", "eventually")},
		{name: "the sentence", message: "Rate limit reached. Please try again in 1.982s. Visit the page.", want: 1982 * time.Millisecond},
		{name: "a compound sentence", message: "Please try again in 1m30s.", want: 90 * time.Second},
		{name: "milliseconds in the sentence", message: "Please try again in 20ms.", want: 20 * time.Millisecond},
		{name: "a header beats the sentence", header: header("Retry-After", "3"), message: "Please try again in 9s.", want: 3 * time.Second},
		{name: "a header past the cap falls to the sentence", header: header("Retry-After", "600"), message: "Please try again in 4s.", want: 4 * time.Second},
		{name: "a wait past two minutes is left to the backoff", message: "Please try again in 6m0s."},
		{name: "a wait too long to count", message: "Please try again in 99999999999999999999h."},
		{name: "a wait of hours is left to the backoff", header: header("x-ratelimit-remaining-tokens", "0", "x-ratelimit-reset-tokens", "9h")},
		{name: "exactly two minutes is honored", header: header("Retry-After", "120"), want: 2 * time.Minute},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := retryDelay(tc.header, tc.message, now); got != tc.want {
				t.Errorf("retryDelay = %s, want %s", got, tc.want)
			}
		})
	}
}

func TestKeysAreMaskedWhereverTheyAppear(t *testing.T) {
	t.Parallel()

	cases := []struct {
		text string
		want string
	}{
		{text: "", want: ""},
		{text: "no key here", want: "no key here"},
		{text: "key sk-abcdefgh1234 rejected", want: "key sk-…1234 rejected"},
		{text: "Incorrect API key provided: sk-proj-********************wxyz.", want: "Incorrect API key provided: sk-…wxyz."},
		{text: "two sk-aaaaaaaa1111 and sk-bbbbbbbb2222", want: "two sk-…1111 and sk-…2222"},
		{text: "too short sk-abc", want: "too short sk-abc"},
	}

	for _, tc := range cases {
		if got := maskKeys(tc.text); got != tc.want {
			t.Errorf("maskKeys(%q) = %q, want %q", tc.text, got, tc.want)
		}
	}
}

func TestARefusalNeverTellsMoreThanThatItRefused(t *testing.T) {
	t.Parallel()

	refused := &refusal{status: http.StatusUnauthorized, fault: wireFault{Message: "Incorrect API key provided: sk-abcdefgh1234."}}
	if got := refused.Error(); got != rejectedRequest {
		t.Errorf("Error() = %q, want the plain sentence", got)
	}
}

func TestTheServedTierFallsBackToTheOneSent(t *testing.T) {
	t.Parallel()

	cases := []struct {
		served string
		sent   string
		want   llm.ServiceTier
	}{
		{served: "flex", sent: "default", want: llm.TierFlex},
		{served: "default", sent: "flex", want: llm.TierDefault},
		{served: "auto", sent: "default", want: llm.TierDefault},
		{served: "priority", sent: "default", want: llm.TierDefault},
		{served: "", sent: "flex", want: llm.TierFlex},
		{served: "", sent: "default", want: llm.TierDefault},
	}
	for _, tc := range cases {
		if got := tierOf(tc.served, tc.sent); got != tc.want {
			t.Errorf("tierOf(%q, %q) = %s, want %s", tc.served, tc.sent, got, tc.want)
		}
	}
}

func TestAnInStreamFailureIsGivenTheStatusItWouldHaveHad(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		fault wireFault
		want  int
	}{
		{name: "the credit is gone", fault: wireFault{Type: "insufficient_quota", Code: "credit_balance_exhausted"}, want: http.StatusTooManyRequests},
		{name: "a rate limit by code", fault: wireFault{Code: "rate_limit_exceeded"}, want: http.StatusTooManyRequests},
		{name: "a rate limit by type", fault: wireFault{Type: "rate_limit_error"}, want: http.StatusTooManyRequests},
		{name: "a request the server refuses", fault: wireFault{Type: "invalid_request_error", Code: "invalid_prompt"}, want: http.StatusBadRequest},
		{name: "an unknown model", fault: wireFault{Type: "invalid_request_error", Code: "model_not_found"}, want: http.StatusNotFound},
		{name: "a rejected key", fault: wireFault{Code: "invalid_api_key"}, want: http.StatusUnauthorized},
		{name: "an authentication failure", fault: wireFault{Type: "authentication_error"}, want: http.StatusUnauthorized},
		{name: "a server error", fault: wireFault{Code: "server_error"}, want: http.StatusInternalServerError},
		{name: "anything else", fault: wireFault{Code: "vector_store_timeout"}, want: http.StatusInternalServerError},
	}

	for _, tc := range cases {
		if got := inferredStatus(tc.fault); got != tc.want {
			t.Errorf("%s: inferredStatus = %d, want %d", tc.name, got, tc.want)
		}
	}
}
