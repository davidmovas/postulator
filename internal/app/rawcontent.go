package app

import (
	"context"

	"github.com/davidmovas/postulator/internal/adapters/wp"
)

type siteClients interface {
	Client(ctx context.Context, siteID string) (*wp.Client, error)
}

type rawContent struct {
	clients siteClients
}

func (r rawContent) RawContent(ctx context.Context, siteID string, wpID int64) (string, error) {
	client, err := r.clients.Client(ctx, siteID)
	if err != nil {
		return "", err
	}

	raw, err := client.GetRaw(ctx, wpID)
	if err != nil {
		return "", err
	}
	return raw.Content, nil
}
