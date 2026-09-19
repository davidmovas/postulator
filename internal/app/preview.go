package app

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/pages"
)

type previewIssuer struct {
	clients siteClients
}

func (p previewIssuer) IssuePreview(ctx context.Context, siteID string, wpID int64) (pages.IssuedPreview, error) {
	client, err := p.clients.Client(ctx, siteID)
	if err != nil {
		return pages.IssuedPreview{}, err
	}

	link, err := client.PreviewLink(ctx, wpID)
	if err != nil {
		return pages.IssuedPreview{}, err
	}
	return pages.IssuedPreview{URL: link.URL, ExpiresAt: link.ExpiresAt}, nil
}
