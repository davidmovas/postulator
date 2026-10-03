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
		{name: "method not allowed", status: http.StatusMethodNotAllowed, body: `{"code":"rest_no_route"}`, want: errors.Invalid},
		{name: "gone", status: http.StatusGone, body: "", want: errors.Invalid},
		{name: "payload too large", status: http.StatusRequestEntityTooLarge, body: `{"code":"rest_upload_too_large"}`, want: errors.Invalid},
		{name: "unprocessable", status: http.StatusUnprocessableEntity, body: `{"code":"rest_invalid_param"}`, want: errors.Invalid},
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

func TestClassifyCarriesTheTermARefusalNames(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		status int
		body   string
		id     int64
		exists bool
	}{
		{
			name:   "core names the term it already has",
			status: http.StatusBadRequest,
			body:   `{"code":"term_exists","message":"A term with the name provided already exists with this parent.","data":{"status":400,"term_id":123}}`,
			id:     123,
			exists: true,
		},
		{
			name:   "woocommerce names it as the resource",
			status: http.StatusBadRequest,
			body:   `{"code":"term_exists","message":"A term with the name provided already exists with this parent.","data":{"status":400,"resource_id":77}}`,
			id:     77,
			exists: true,
		},
		{
			name:   "an id sent as text is still an id",
			status: http.StatusBadRequest,
			body:   `{"code":"term_exists","data":{"status":400,"term_id":"41"}}`,
			id:     41,
			exists: true,
		},
		{
			name:   "a refusal without data names no term",
			status: http.StatusBadRequest,
			body:   `{"code":"term_exists","message":"A term with the name provided already exists with this parent."}`,
		},
		{
			name:   "data that is not an object keeps the code",
			status: http.StatusBadRequest,
			body:   `{"code":"term_exists","message":"exists","data":"term 5"}`,
		},
		{
			name:   "another refusal is not a duplicate",
			status: http.StatusBadRequest,
			body:   `{"code":"rest_term_invalid","message":"Parent term does not exist.","data":{"status":400,"term_id":9}}`,
			id:     9,
		},
		{
			name:   "a duplicate is only ever a refused request",
			status: http.StatusInternalServerError,
			body:   `{"code":"term_exists","data":{"status":500,"term_id":9}}`,
			id:     9,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := classify(response(t, tc.status, nil), []byte(tc.body))
			if detailString(err, "code") == "" {
				t.Errorf("the WordPress code was lost from %s", tc.body)
			}

			value, carried := detailValue(err, "termId")
			if tc.id == 0 && carried {
				t.Errorf("termId detail = %v, want none", value)
			}
			if tc.id != 0 && value != tc.id {
				t.Errorf("termId detail = %v, want %d", value, tc.id)
			}

			id, exists := TermExists(err)
			if exists != tc.exists {
				t.Fatalf("TermExists = %t, want %t", exists, tc.exists)
			}
			if exists && id != tc.id {
				t.Errorf("TermExists id = %d, want %d", id, tc.id)
			}
		})
	}
}

func TestForbiddenTellsAMissingPermissionFromABadPassword(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		status int
		body   string
		want   bool
	}{
		{name: "a user who may not create terms", status: http.StatusForbidden, body: `{"code":"rest_cannot_create"}`, want: true},
		{name: "a user who may not edit products", status: http.StatusForbidden, body: `{"code":"woocommerce_rest_cannot_edit"}`, want: true},
		{name: "a refused password", status: http.StatusUnauthorized, body: `{"code":"rest_not_logged_in"}`},
		{name: "a refused request", status: http.StatusBadRequest, body: `{"code":"term_exists"}`},
		{name: "a missing resource", status: http.StatusNotFound, body: `{"code":"rest_term_invalid"}`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := classify(response(t, tc.status, nil), []byte(tc.body))
			if got := Forbidden(err); got != tc.want {
				t.Errorf("Forbidden = %t, want %t", got, tc.want)
			}
			if got := StoreForbidden(err); got != tc.want {
				t.Errorf("StoreForbidden = %t, want %t", got, tc.want)
			}
		})
	}

	if Forbidden(errors.New(errors.Unauthorized, "no status")) {
		t.Error("an unauthorized error without a status is not a refused permission")
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

	patient := classify(response(t, http.StatusServiceUnavailable, http.Header{"Retry-After": {"11"}}), nil)
	if got := delayFor(patient, 250*time.Millisecond); got != 11*time.Second {
		t.Errorf("retry = %s, want the 11s the site asked for", got)
	}

	dated := classify(response(t, http.StatusBadGateway, http.Header{"Retry-After": {time.Now().Add(time.Minute).UTC().Format(http.TimeFormat)}}), nil)
	if got := delayFor(dated, time.Millisecond); got <= time.Millisecond || got > time.Minute {
		t.Errorf("retry = %s, want a positive delay of at most a minute", got)
	}
}

func TestAnUnmappedClientErrorIsInvalidAndNeverRetried(t *testing.T) {
	t.Parallel()

	statuses := []int{
		http.StatusMethodNotAllowed,
		http.StatusGone,
		http.StatusRequestEntityTooLarge,
		http.StatusUnprocessableEntity,
	}

	for _, status := range statuses {
		err := classify(response(t, status, http.Header{"Retry-After": {"30"}}), []byte(`{"code":"rest_no_route","message":"No route was found"}`))
		if !errors.IsCode(err, errors.Invalid) {
			t.Errorf("status %d code = %q, want %q", status, errors.CodeOf(err), errors.Invalid)
		}
		if retryable(err) {
			t.Errorf("status %d must never be retried", status)
		}
		if got := detailString(err, "code"); got != "rest_no_route" {
			t.Errorf("status %d code detail = %q", status, got)
		}
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
