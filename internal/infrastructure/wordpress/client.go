package wordpress

import (
	"Postulator/internal/domain/site"
	"context"
)

type Client interface {
	CheckHealth(ctx context.Context, site *site.Site) error
	GetCategories(ctx context.Context, site *site.Site) ([]*site.Category, error)
	PublishPost(ctx context.Context, site *site.Site, title, content string, categoryID int) (postID int, postURL string, err error)
}
