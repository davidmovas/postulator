package wp

import (
	"net/http"
	"testing"

	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func TestRetryAllowed(t *testing.T) {
	t.Parallel()

	transient := errors.New(errors.External, "unreachable")
	limited := errors.New(errors.RateLimited, "slow down")
	refused := errors.New(errors.Unauthorized, "no")

	cases := []struct {
		name   string
		method string
		result attempt
		want   bool
	}{
		{name: "a read that got a server error", method: http.MethodGet, result: attempt{status: http.StatusBadGateway, err: transient}, want: true},
		{name: "a read that never reached the site", method: http.MethodGet, result: attempt{err: transient}, want: true},
		{name: "a read that was refused", method: http.MethodGet, result: attempt{status: http.StatusForbidden, err: refused}},
		{name: "a head that never reached the site", method: http.MethodHead, result: attempt{err: transient}, want: true},
		{name: "a write that got a server error", method: http.MethodPost, result: attempt{status: http.StatusInternalServerError, err: transient}},
		{name: "a write that was rate limited", method: http.MethodPost, result: attempt{status: http.StatusTooManyRequests, err: limited}},
		{name: "a write that never reached the site", method: http.MethodPost, result: attempt{err: transient}, want: true},
		{name: "a put that got a server error", method: http.MethodPut, result: attempt{status: http.StatusServiceUnavailable, err: transient}},
		{name: "a put that never reached the site", method: http.MethodPut, result: attempt{err: transient}, want: true},
		{name: "a delete that got a server error", method: http.MethodDelete, result: attempt{status: http.StatusInternalServerError, err: transient}},
		{name: "a delete that never reached the site", method: http.MethodDelete, result: attempt{err: transient}, want: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := retryAllowed(tc.method, tc.result); got != tc.want {
				t.Errorf("retryAllowed(%s, status %d) = %t, want %t", tc.method, tc.result.status, got, tc.want)
			}
		})
	}
}
