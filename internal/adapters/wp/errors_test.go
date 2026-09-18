package wp

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func response(t *testing.T, status int, header http.Header) *http.Response {
	t.Helper()

	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "https://example.com/wp-json/wp/v2/pages", http.NoBody)
	if err != nil {
		t.Fatalf("build the request: %v", err)
	}
	if header == nil {
		header = http.Header{}
	}
	return &http.Response{StatusCode: status, Header: header, Request: request}
}

func TestClassifyMapsWordPressStatusesToKernelCodes(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		status int
		header http.Header
		body   string
		want   errors.Code
	}{
		{name: "bad request", status: http.StatusBadRequest, body: `{"code":"rest_invalid_param","message":"Invalid parameter(s): slug"}`, want: errors.Invalid},
		{name: "unauthorized", status: http.StatusUnauthorized, body: `{"code":"rest_not_logged_in","message":"You are not currently logged in."}`, want: errors.Unauthorized},
		{name: "forbidden", status: http.StatusForbidden, body: `{"code":"rest_forbidden"}`, want: errors.Unauthorized},
		{name: "not found", status: http.StatusNotFound, body: `{"code":"rest_post_invalid_id"}`, want: errors.NotFound},
		{name: "conflict", status: http.StatusConflict, body: `{"code":"hash_mismatch","currentHash":"abc"}`, want: errors.Conflict},
		{name: "rate limited", status: http.StatusTooManyRequests, header: http.Header{"Retry-After": {"5"}}, body: `{"code":"too_many_requests"}`, want: errors.RateLimited},
		{name: "server error", status: http.StatusInternalServerError, body: "<html>fatal</html>", want: errors.External},
		{name: "gateway error", status: http.StatusBadGateway, body: "", want: errors.External},
		{name: "unexpected redirect", status: http.StatusFound, body: "", want: errors.External},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := classify(response(t, tc.status, tc.header), []byte(tc.body))
			if !errors.IsCode(err, tc.want) {
				t.Fatalf("code = %q, want %q", errors.CodeOf(err), tc.want)
			}
			if got, ok := detail(err, "status"); !ok || got != tc.status {
				t.Errorf("status detail = %v, %t", got, ok)
			}
			if got := detailString(err, "path"); got != "/wp-json/wp/v2/pages" {
				t.Errorf("path detail = %q", got)
			}
		})
	}
}

func detail(err error, key string) (int, bool) {
	value, ok := detailValue(err, key)
	if !ok {
		return 0, false
	}
	number, ok := value.(int)
	return number, ok
}

func TestClassifyCarriesTheWordPressCodeAndConflictHash(t *testing.T) {
	t.Parallel()

	invalid := classify(response(t, http.StatusBadRequest, nil), []byte(`{"code":"rest_invalid_param","message":"Invalid parameter(s): slug"}`))
	if got := detailString(invalid, "code"); got != "rest_invalid_param" {
		t.Errorf("code detail = %q", got)
	}
	if got := detailString(invalid, "wpMessage"); got != "Invalid parameter(s): slug" {
		t.Errorf("wpMessage detail = %q", got)
	}

	conflict := classify(response(t, http.StatusConflict, nil), []byte(`{"code":"hash_mismatch","currentHash":"deadbeef"}`))
	if got := detailString(conflict, "currentHash"); got != "deadbeef" {
		t.Errorf("currentHash detail = %q", got)
	}
}

func TestClassifyAttachesRetryInformation(t *testing.T) {
	t.Parallel()

	limited := classify(response(t, http.StatusTooManyRequests, http.Header{"Retry-After": {"5"}}), nil)
	if got := delayFor(limited, time.Minute); got != 5*time.Second {
		t.Errorf("retry = %s, want 5s", got)
	}

	failed := classify(response(t, http.StatusServiceUnavailable, nil), nil)
	if got := delayFor(failed, 250*time.Millisecond); got != 250*time.Millisecond {
		t.Errorf("retry = %s, want the fallback backoff", got)
	}
}

func TestRetryAfterUnderstandsSecondsAndDates(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		header string
		want   time.Duration
	}{
		{name: "absent", header: "", want: 0},
		{name: "seconds", header: "7", want: 7 * time.Second},
		{name: "padded seconds", header: " 7 ", want: 7 * time.Second},
		{name: "negative", header: "-7", want: 0},
		{name: "rubbish", header: "soon", want: 0},
		{name: "date in the past", header: "Wed, 21 Oct 2015 07:28:00 GMT", want: 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := retryAfter(tc.header); got != tc.want {
				t.Errorf("retryAfter(%q) = %s, want %s", tc.header, got, tc.want)
			}
		})
	}
}

func TestRetryAfterUnderstandsAFutureDate(t *testing.T) {
	t.Parallel()

	header := time.Now().Add(time.Minute).UTC().Format(http.TimeFormat)
	if got := retryAfter(header); got <= 0 || got > time.Minute {
		t.Errorf("retryAfter(%q) = %s, want a positive delay of at most a minute", header, got)
	}
}

func TestOnlyTransientCodesAreRetried(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		code errors.Code
		want bool
	}{
		{name: "rate limited", code: errors.RateLimited, want: true},
		{name: "external", code: errors.External, want: true},
		{name: "unauthorized", code: errors.Unauthorized},
		{name: "not found", code: errors.NotFound},
		{name: "conflict", code: errors.Conflict},
		{name: "invalid", code: errors.Invalid},
		{name: "cancelled", code: errors.Cancelled},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := retryable(errors.New(tc.code, "x")); got != tc.want {
				t.Errorf("retryable(%s) = %t, want %t", tc.code, got, tc.want)
			}
		})
	}
}

func TestATransportFailureRespectsTheCallersContext(t *testing.T) {
	t.Parallel()

	live := transportError(t.Context(), http.ErrHandlerTimeout)
	if !errors.IsCode(live, errors.External) {
		t.Errorf("code = %q, want %q", errors.CodeOf(live), errors.External)
	}
	if !retryable(live) {
		t.Error("our own timeout must be retryable")
	}

	cancelled, cancel := context.WithCancel(t.Context())
	cancel()
	if got := transportError(cancelled, http.ErrHandlerTimeout); !errors.IsCode(got, errors.Cancelled) {
		t.Errorf("code = %q, want %q", errors.CodeOf(got), errors.Cancelled)
	}
}
