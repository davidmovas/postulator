package agent

import (
	"context"
	stderrors "errors"
	"sync"
	"time"

	"github.com/gollem-dev/gollem"
	"golang.org/x/time/rate"

	"github.com/davidmovas/postulator/internal/adapters/llm/gollemclient"
	"github.com/davidmovas/postulator/internal/adapters/llm/retry"
	domainllm "github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	secondsPerMinute = 60
	maxBackoff       = 30 * time.Second
)

type patience struct {
	mu       sync.Mutex
	catalog  catalogReader
	limiters map[string]*rate.Limiter
	retries  int
	backoff  time.Duration
}

func newPatience(catalog catalogReader, retries int, backoff time.Duration) *patience {
	if retries < 0 {
		retries = 0
	}
	if retries > retry.MaxRetries {
		retries = retry.MaxRetries
	}
	if backoff <= 0 {
		backoff = retry.DefaultBackoff
	}
	return &patience{catalog: catalog, limiters: make(map[string]*rate.Limiter), retries: retries, backoff: backoff}
}

func (p *patience) middleware(ref domainllm.ModelRef) gollem.ContentStreamMiddleware {
	return func(next gollem.ContentStreamHandler) gollem.ContentStreamHandler {
		return func(ctx context.Context, req *gollem.ContentRequest) (<-chan *gollem.ContentResponse, error) {
			var last error
			for attempt := 0; attempt <= p.retries; attempt++ {
				if attempt > 0 {
					if err := p.pause(ctx, last, attempt); err != nil {
						return nil, err
					}
				}
				if err := p.wait(ctx, ref); err != nil {
					return nil, err
				}

				chunks, err := next(ctx, req)
				if err == nil {
					return chunks, nil
				}
				if !worthAnotherTry(err) {
					return nil, err
				}
				last = err
			}
			return nil, last
		}
	}
}

func (p *patience) wait(ctx context.Context, ref domainllm.ModelRef) error {
	if p.catalog == nil {
		return nil
	}
	info, err := p.catalog.Lookup(ctx, ref)
	if err != nil {
		return nil
	}

	held := p.limiterFor(ref, info.RPM)
	if held == nil {
		return nil
	}
	if waitErr := held.Wait(ctx); waitErr != nil {
		return errors.New(errors.Cancelled, "the agent turn was stopped while waiting for the provider rate limit").
			WithDetail("model", ref.String()).WithInternal(waitErr)
	}
	return nil
}

func (p *patience) limiterFor(ref domainllm.ModelRef, rpm int) *rate.Limiter {
	if rpm <= 0 {
		return nil
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	key := ref.String()
	if existing, ok := p.limiters[key]; ok {
		return existing
	}

	created := rate.NewLimiter(rate.Limit(float64(rpm)/secondsPerMinute), rpm)
	p.limiters[key] = created
	return created
}

func (p *patience) pause(ctx context.Context, last error, attempt int) error {
	timer := time.NewTimer(delayFor(last, p.backoffFor(attempt-1)))
	defer timer.Stop()

	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return errors.New(errors.Cancelled, "the agent turn was stopped while waiting to try the model again").
			WithInternal(ctx.Err())
	}
}

func (p *patience) backoffFor(step int) time.Duration {
	delay := p.backoff
	for range step {
		delay *= 2
		if delay >= maxBackoff {
			return maxBackoff
		}
	}
	return delay
}

func worthAnotherTry(err error) bool {
	if provider, ok := gollemclient.Provider(err); ok {
		err = provider
	}
	code := errors.CodeOf(err)
	return code == errors.RateLimited || code == errors.External
}

func delayFor(err error, fallback time.Duration) time.Duration {
	if provider, ok := gollemclient.Provider(err); ok {
		err = provider
	}

	var kernel *errors.Error
	if stderrors.As(err, &kernel) && kernel != nil && kernel.Retry != nil && kernel.Retry.After > 0 {
		return kernel.Retry.After
	}
	return fallback
}
