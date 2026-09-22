package agent

import (
	"context"
	"testing"
	"time"

	"github.com/gollem-dev/gollem"

	domainllm "github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

type countingCatalog struct {
	rpm int
}

func (c countingCatalog) Lookup(context.Context, domainllm.ModelRef) (domainllm.ModelInfo, error) {
	return domainllm.ModelInfo{RPM: c.rpm}, nil
}

func refused(code errors.Code, after time.Duration) error {
	built := errors.New(code, "the provider refused the call")
	if after > 0 {
		return built.WithRetry(after)
	}
	return built
}

func TestATurnSurvivesAProviderThatRefusesOnce(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		failures  int
		with      error
		retries   int
		wantCalls int
		wantErr   bool
	}{
		{
			name: "a rate limit is waited out", failures: 1, with: refused(errors.RateLimited, time.Millisecond),
			retries: 3, wantCalls: 2,
		},
		{
			name: "a provider that is down is tried again", failures: 2, with: refused(errors.External, 0),
			retries: 3, wantCalls: 3,
		},
		{
			name: "a refusal the model caused is not retried", failures: 1, with: refused(errors.Invalid, 0),
			retries: 3, wantCalls: 1, wantErr: true,
		},
		{
			name: "the budget runs out", failures: 5, with: refused(errors.RateLimited, time.Millisecond),
			retries: 2, wantCalls: 3, wantErr: true,
		},
		{
			name: "a call that works is made once", failures: 0, with: nil, retries: 3, wantCalls: 1,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			calls := 0
			waiting := newPatience(countingCatalog{rpm: 600}, tc.retries, time.Millisecond)
			handler := waiting.middleware(domainllm.ModelRef{Provider: "openai", Model: "chat"})(
				func(context.Context, *gollem.ContentRequest) (<-chan *gollem.ContentResponse, error) {
					calls++
					if calls <= tc.failures {
						return nil, tc.with
					}
					out := make(chan *gollem.ContentResponse)
					close(out)
					return out, nil
				})

			_, err := handler(t.Context(), &gollem.ContentRequest{})
			if (err != nil) != tc.wantErr {
				t.Fatalf("the call answered %v, want an error %t", err, tc.wantErr)
			}
			if calls != tc.wantCalls {
				t.Errorf("the provider was called %d times, want %d", calls, tc.wantCalls)
			}
		})
	}
}

func TestAModelWithNoDeclaredRateIsNotHeldBack(t *testing.T) {
	t.Parallel()

	waiting := newPatience(countingCatalog{rpm: 0}, 0, time.Millisecond)
	ref := domainllm.ModelRef{Provider: "openai", Model: "chat"}

	deadline, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()

	for range 5 {
		if err := waiting.wait(deadline, ref); err != nil {
			t.Fatalf("a model with no declared rate was held back: %v", err)
		}
	}
}
