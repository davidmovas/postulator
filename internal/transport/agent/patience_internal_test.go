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
			held := newPatience(countingCatalog{rpm: 600}, tc.retries, time.Millisecond)
			handler := held.middleware(domainllm.ModelRef{Provider: "openai", Model: "chat"}, nil)(
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

	held := newPatience(countingCatalog{rpm: 0}, 0, time.Millisecond)
	ref := domainllm.ModelRef{Provider: "openai", Model: "chat"}

	deadline, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()

	for range 5 {
		if err := held.wait(deadline, ref); err != nil {
			t.Fatalf("a model with no declared rate was held back: %v", err)
		}
	}
}

func TestARateLimitedRoundSaysHowLongItIsWaiting(t *testing.T) {
	t.Parallel()

	type notice struct {
		reason  errors.Code
		attempt int
		delay   time.Duration
	}

	told := make([]notice, 0, 2)
	calls := 0
	held := newPatience(countingCatalog{rpm: 600}, 3, time.Millisecond)
	handler := held.middleware(domainllm.ModelRef{Provider: "openai", Model: "chat"},
		func(_ context.Context, cause error, attempt int, delay time.Duration) {
			told = append(told, notice{reason: errors.CodeOf(cause), attempt: attempt, delay: delay})
		})(
		func(context.Context, *gollem.ContentRequest) (<-chan *gollem.ContentResponse, error) {
			calls++
			if calls <= 2 {
				return nil, refused(errors.RateLimited, 4*time.Millisecond)
			}
			out := make(chan *gollem.ContentResponse)
			close(out)
			return out, nil
		})

	if _, err := handler(t.Context(), &gollem.ContentRequest{}); err != nil {
		t.Fatalf("the call answered %v", err)
	}
	if len(told) != 2 {
		t.Fatalf("the stream was told about %d waits, want one per retry: %+v", len(told), told)
	}
	for index, said := range told {
		if said.attempt != index+1 || said.reason != errors.RateLimited {
			t.Fatalf("wait %d reads %+v", index+1, said)
		}
		if said.delay != 4*time.Millisecond {
			t.Fatalf("wait %d held for %s, want the delay the provider named", index+1, said.delay)
		}
	}
}
