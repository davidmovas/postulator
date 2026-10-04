package tuning

import (
	"context"

	port "github.com/davidmovas/postulator/internal/application/llm"
)

type Client struct {
	next   port.Client
	policy *Policy
}

func New(next port.Client, policy *Policy) *Client {
	return &Client{next: next, policy: policy}
}

func (c *Client) Complete(ctx context.Context, req port.Request) (port.Response, error) {
	return c.next.Complete(ctx, c.tuned(req))
}

func (c *Client) Stream(ctx context.Context, req port.Request) (<-chan port.Delta, error) {
	return c.next.Stream(ctx, c.tuned(req))
}

func (c *Client) tuned(req port.Request) port.Request {
	if req.Effort == "" {
		req.Effort = c.policy.Effort(req.Meta.Role)
	}
	if req.Tier == "" {
		req.Tier = c.policy.Tier(req.Meta.Role)
	}
	return req
}
