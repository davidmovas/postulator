package retry

import (
	"context"
	stderrors "errors"
	"time"

	port "github.com/davidmovas/postulator/internal/application/llm"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/settings"
)

const (
	MaxRetries     = 5
	DefaultBackoff = 500 * time.Millisecond
	maxBackoff     = 30 * time.Second
)

var retriesSetting = settings.Int("llm.retries", MaxRetries, settings.IntRange(0, MaxRetries))

func Retries(values *settings.Values) int {
	return retriesSetting.Get(values)
}

type Client struct {
	next    port.Client
	backoff time.Duration
	retries int
}

func New(next port.Client, retries int, backoff time.Duration) *Client {
	if retries < 0 {
		retries = 0
	}
	if retries > MaxRetries {
		retries = MaxRetries
	}
	if backoff <= 0 {
		backoff = DefaultBackoff
	}
	return &Client{next: next, retries: retries, backoff: backoff}
}

func (c *Client) Complete(ctx context.Context, req port.Request) (port.Response, error) {
	var last error
	for attempt := 0; attempt <= c.retries; attempt++ {
		if attempt > 0 {
			if err := c.pause(ctx, last, attempt); err != nil {
				return port.Response{}, err
			}
		}

		resp, err := c.next.Complete(ctx, req)
		if err == nil {
			return resp, nil
		}
		if !retryable(err) {
			return port.Response{}, err
		}
		last = err
	}
	return port.Response{}, last
}

func (c *Client) Stream(ctx context.Context, req port.Request) (<-chan port.Delta, error) {
	var last error
	for attempt := 0; attempt <= c.retries; attempt++ {
		if attempt > 0 {
			if err := c.pause(ctx, last, attempt); err != nil {
				return nil, err
			}
		}

		deltas, err := c.next.Stream(ctx, req)
		if err == nil {
			return deltas, nil
		}
		if !retryable(err) {
			return nil, err
		}
		last = err
	}
	return nil, last
}

func (c *Client) pause(ctx context.Context, last error, attempt int) error {
	timer := time.NewTimer(delayFor(last, c.backoffFor(attempt-1)))
	defer timer.Stop()

	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return errors.New(errors.Cancelled, "the model call was cancelled while waiting to retry").WithInternal(ctx.Err())
	}
}

func (c *Client) backoffFor(step int) time.Duration {
	delay := c.backoff
	for range step {
		delay *= 2
		if delay >= maxBackoff {
			return maxBackoff
		}
	}
	return delay
}

func retryable(err error) bool {
	code := errors.CodeOf(err)
	return code == errors.RateLimited || code == errors.External
}

func delayFor(err error, fallback time.Duration) time.Duration {
	var kernel *errors.Error
	if stderrors.As(err, &kernel) && kernel != nil && kernel.Retry != nil && kernel.Retry.After > 0 {
		return kernel.Retry.After
	}
	return fallback
}
