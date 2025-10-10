package ai

import "context"

type IClient interface {
	GenerateArticle(ctx context.Context, systemPrompt, userPrompt string) (string, error)
}

var _ IClient = (*Client)(nil)

type Client struct{}

func NewClient() *Client {
	return &Client{}
}

func (c *Client) GenerateArticle(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	return "", nil
}
