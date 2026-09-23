package retry_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/llm/retry"
	port "github.com/davidmovas/postulator/internal/application/llm"
	"github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/settings"
)

type flaky struct {
	failures []error
	calls    int
}

func (f *flaky) next() error {
	f.calls++
	if f.calls <= len(f.failures) {
		return f.failures[f.calls-1]
	}
	return nil
}

func (f *flaky) Complete(_ context.Context, _ port.Request) (port.Response, error) {
	if err := f.next(); err != nil {
		return port.Response{}, err
	}
	return port.Response{Text: "ok"}, nil
}

func (f *flaky) Stream(_ context.Context, _ port.Request) (<-chan port.Delta, error) {
	if err := f.next(); err != nil {
		return nil, err
	}
	out := make(chan port.Delta, 1)
	out <- port.Delta{Done: true}
	close(out)
	return out, nil
}

func request() port.Request {
	return port.Request{
		Ref:      llm.ModelRef{Provider: "openai", Model: "gpt-5.6-luna"},
		Messages: []port.Message{{Role: port.RoleUser, Text: "write"}},
	}
}

func repeat(err error, times int) []error {
	out := make([]error, 0, times)
	for range times {
		out = append(out, err)
	}
	return out
}

func TestRetries(t *testing.T) {
	t.Parallel()

	rateLimited := errors.New(errors.RateLimited, "slow down").WithRetry(time.Millisecond)
	external := errors.New(errors.External, "gateway")

	cases := []struct {
		name      string
		failures  []error
		retries   int
		wantCalls int
		wantErr   errors.Code
	}{
		{name: "a first success", retries: 3, wantCalls: 1},
		{name: "one rate limit", failures: []error{rateLimited}, retries: 3, wantCalls: 2},
		{name: "two upstream failures", failures: []error{external, external}, retries: 3, wantCalls: 3},
		{name: "the budget runs out", failures: repeat(external, 4), retries: 2, wantCalls: 3, wantErr: errors.External},
		{name: "no retries at all", failures: []error{external}, retries: 0, wantCalls: 1, wantErr: errors.External},
		{
			name:      "the ceiling is five",
			failures:  repeat(external, 9),
			retries:   50,
			wantCalls: retry.MaxRetries + 1,
			wantErr:   errors.External,
		},
		{name: "a rejected request", failures: []error{errors.New(errors.Invalid, "no")}, retries: 3, wantCalls: 1, wantErr: errors.Invalid},
		{name: "a rejected key", failures: []error{errors.New(errors.Unauthorized, "no")}, retries: 3, wantCalls: 1, wantErr: errors.Unauthorized},
		{name: "a cancelled call", failures: []error{errors.New(errors.Cancelled, "no")}, retries: 3, wantCalls: 1, wantErr: errors.Cancelled},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			for _, mode := range []string{"complete", "stream"} {
				inner := &flaky{failures: tc.failures}
				client := retry.New(inner, tc.retries, time.Millisecond)

				var err error
				if mode == "complete" {
					_, err = client.Complete(t.Context(), request())
				} else {
					_, err = client.Stream(t.Context(), request())
				}

				if tc.wantErr == "" && err != nil {
					t.Fatalf("%s: %v", mode, err)
				}
				if tc.wantErr != "" && !errors.IsCode(err, tc.wantErr) {
					t.Fatalf("%s error = %v, want %s", mode, err, tc.wantErr)
				}
				if inner.calls != tc.wantCalls {
					t.Errorf("%s calls = %d, want %d", mode, inner.calls, tc.wantCalls)
				}
			}
		})
	}
}

func TestRetryHonoursTheProviderHint(t *testing.T) {
	t.Parallel()

	hinted := errors.New(errors.RateLimited, "slow down").WithRetry(60 * time.Millisecond)
	inner := &flaky{failures: []error{hinted}}
	client := retry.New(inner, 3, time.Microsecond)

	started := time.Now()
	if _, err := client.Complete(t.Context(), request()); err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if elapsed := time.Since(started); elapsed < 50*time.Millisecond {
		t.Errorf("waited %s, want at least the provider hint", elapsed)
	}
}

func TestRetryStopsWhenTheCallerLeaves(t *testing.T) {
	t.Parallel()

	inner := &flaky{failures: repeat(errors.New(errors.External, "gateway"), 3)}
	client := retry.New(inner, 3, time.Hour)

	ctx, cancel := context.WithCancel(t.Context())
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	if _, err := client.Complete(ctx, request()); !errors.IsCode(err, errors.Cancelled) {
		t.Fatalf("Complete error = %v, want %s", err, errors.Cancelled)
	}
}

func TestBackoffGrowsAndIsBounded(t *testing.T) {
	t.Parallel()

	inner := &flaky{failures: repeat(errors.New(errors.External, "gateway"), 10)}
	client := retry.New(inner, -1, 0)

	if _, err := client.Complete(t.Context(), request()); !errors.IsCode(err, errors.External) {
		t.Fatalf("Complete error = %v, want %s", err, errors.External)
	}
	if inner.calls != 1 {
		t.Errorf("calls = %d, want a single attempt when retries are negative", inner.calls)
	}
}

func TestRetriesSetting(t *testing.T) {
	t.Parallel()

	values := settings.Default().NewValues()
	if got := retry.Retries(values); got != retry.MaxRetries {
		t.Fatalf("Retries = %d, want %d", got, retry.MaxRetries)
	}

	encoded, err := json.Marshal(2)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	if _, err = settings.Default().Apply(values, map[string]json.RawMessage{"llm.retries": encoded}); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if got := retry.Retries(values); got != 2 {
		t.Errorf("Retries = %d, want 2", got)
	}
}
