package limiter

import (
	"context"
	"sync"

	"golang.org/x/time/rate"

	port "github.com/davidmovas/postulator/internal/application/llm"
	"github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const secondsPerMinute = 60

type catalogReader interface {
	Lookup(ctx context.Context, ref llm.ModelRef) (llm.ModelInfo, error)
}

type Client struct {
	next     port.Client
	catalog  catalogReader
	limiters map[string]*rate.Limiter
	mu       sync.Mutex
}

func New(next port.Client, catalog catalogReader) *Client {
	return &Client{next: next, catalog: catalog, limiters: make(map[string]*rate.Limiter)}
}

func (c *Client) Complete(ctx context.Context, req port.Request) (port.Response, error) {
	if err := c.wait(ctx, req.Ref); err != nil {
		return port.Response{}, err
	}
	return c.next.Complete(ctx, req)
}

func (c *Client) Stream(ctx context.Context, req port.Request) (<-chan port.Delta, error) {
	if err := c.wait(ctx, req.Ref); err != nil {
		return nil, err
	}
	return c.next.Stream(ctx, req)
}

func (c *Client) wait(ctx context.Context, ref llm.ModelRef) error {
	info, err := c.catalog.Lookup(ctx, ref)
	if err != nil {
		return err
	}

	if err = c.limiterFor(ref, info.RPM).Wait(ctx); err != nil {
		return errors.New(errors.Cancelled, "the model call was cancelled while waiting for the provider rate limit").
			WithDetail("model", ref.String()).
			WithInternal(err)
	}
	return nil
}

func (c *Client) limiterFor(ref llm.ModelRef, rpm int) *rate.Limiter {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := ref.String()
	if existing, ok := c.limiters[key]; ok {
		return existing
	}

	if rpm <= 0 {
		rpm = 1
	}
	created := rate.NewLimiter(rate.Limit(float64(rpm)/secondsPerMinute), rpm)
	c.limiters[key] = created
	return created
}
