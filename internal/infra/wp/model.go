package wp

import (
	"context"

	"github.com/davidmovas/postulator/internal/domain/entities"
)

type PostOptions struct {
	Status string // "draft" or "publish"; default "publish" if empty
}

type Client interface {
	CheckHealth(ctx context.Context, site *entities.Site) (*entities.HealthCheck, error)

	GetCategories(ctx context.Context, s *entities.Site) ([]*entities.Category, error)
	CreateCategory(ctx context.Context, s *entities.Site, category *entities.Category) error
	UpdateCategory(ctx context.Context, s *entities.Site, category *entities.Category) error
	DeleteCategory(ctx context.Context, s *entities.Site, wpCategoryID int) error

	GetPost(ctx context.Context, s *entities.Site, postID int) (*entities.Article, error)
	GetPosts(ctx context.Context, s *entities.Site) ([]*entities.Article, error)
	CreatePost(ctx context.Context, s *entities.Site, article *entities.Article, opts *PostOptions) (int, error)
	UpdatePost(ctx context.Context, s *entities.Site, article *entities.Article) error
	DeletePost(ctx context.Context, s *entities.Site, postID int) error

	EnableProxy(proxyURL string)
	DisableProxy()
}
