package wp_test

import (
	"context"
	stderrors "errors"
	"net/http"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/wp"
	"github.com/davidmovas/postulator/internal/adapters/wp/wptest"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func newClient(t *testing.T, server *wptest.Server, opts ...wp.Option) *wp.Client {
	t.Helper()

	options := append([]wp.Option{
		wp.WithRateLimit(0),
		wp.WithBackoff(func(int) time.Duration { return 0 }),
	}, opts...)

	client, err := wp.New(wp.Config{
		BaseURL:       server.URL(),
		Username:      wptest.DefaultUser,
		AppPassword:   wptest.DefaultPassword,
		AllowInsecure: true,
	}, options...)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return client
}

func TestEveryRequestCarriesBasicAuthAndOurHeaders(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	if _, err := newClient(t, server).Probe(t.Context()); err != nil {
		t.Fatalf("Probe: %v", err)
	}

	recorded, ok := server.LastRequest()
	if !ok {
		t.Fatal("no request reached the site")
	}

	user, password, present := parseBasicAuth(t, recorded.Header.Get("Authorization"))
	if !present || user != wptest.DefaultUser || password != wptest.DefaultPassword {
		t.Errorf("basic auth = %q %q %t", user, password, present)
	}
	if got := recorded.Header.Get("Accept"); got != "application/json" {
		t.Errorf("Accept = %q", got)
	}
	if got := recorded.Header.Get("User-Agent"); got == "" {
		t.Error("the request must identify Postulator")
	}
}

func parseBasicAuth(t *testing.T, header string) (user, password string, present bool) {
	t.Helper()

	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "https://example.com", http.NoBody)
	if err != nil {
		t.Fatalf("build the request: %v", err)
	}
	request.Header.Set("Authorization", header)
	return request.BasicAuth()
}

func TestATransientFailureIsRetriedUntilItSucceeds(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	server.FailNext(http.StatusServiceUnavailable, 2)

	if _, err := newClient(t, server).Probe(t.Context()); err != nil {
		t.Fatalf("Probe: %v", err)
	}
	if got := len(server.Requests()); got != 3 {
		t.Errorf("the site saw %d requests, want 3", got)
	}
}

func TestTheRetryBudgetIsFinite(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	server.FailNext(http.StatusServiceUnavailable, 10)

	_, err := newClient(t, server, wp.WithRetries(2)).Probe(t.Context())
	if !errors.IsCode(err, errors.External) {
		t.Fatalf("code = %q, want %q", errors.CodeOf(err), errors.External)
	}
	if got := len(server.Requests()); got != 3 {
		t.Errorf("the site saw %d requests, want 1 attempt and 2 retries", got)
	}
}

func TestANonTransientFailureIsNotRetried(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	server.FailNext(http.StatusForbidden, 3)

	_, err := newClient(t, server).Probe(t.Context())
	if !errors.IsCode(err, errors.Unauthorized) {
		t.Fatalf("code = %q, want %q", errors.CodeOf(err), errors.Unauthorized)
	}
	if got := len(server.Requests()); got != 1 {
		t.Errorf("the site saw %d requests, want 1", got)
	}
}

func TestRetryAfterIsReportedOnTheError(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	server.RateLimitNext(3 * time.Second)

	_, err := newClient(t, server, wp.WithRetries(0)).Probe(t.Context())
	if !errors.IsCode(err, errors.RateLimited) {
		t.Fatalf("code = %q, want %q", errors.CodeOf(err), errors.RateLimited)
	}

	var kernel *errors.Error
	if !stderrors.As(err, &kernel) || kernel.Retry == nil || kernel.Retry.After != 3*time.Second {
		t.Errorf("retry = %+v, want 3s", kernel)
	}
}

func TestRetryAfterIsWaitedForRatherThanTheBackoff(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	server.RateLimitNext(5 * time.Second)

	ctx, cancel := context.WithTimeout(t.Context(), 300*time.Millisecond)
	defer cancel()

	_, err := newClient(t, server, wp.WithRetries(1)).Probe(ctx)
	if !errors.IsCode(err, errors.Cancelled) {
		t.Fatalf("code = %q, want %q; the client did not wait out Retry-After", errors.CodeOf(err), errors.Cancelled)
	}
	if got := len(server.Requests()); got != 1 {
		t.Errorf("the site saw %d requests, want 1", got)
	}
}

func TestACancelledContextStopsTheCall(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := newClient(t, server).Probe(ctx)
	if !errors.IsCode(err, errors.Cancelled) {
		t.Errorf("code = %q, want %q", errors.CodeOf(err), errors.Cancelled)
	}
}
