package wp

import (
	"context"

	"github.com/davidmovas/postulator/internal/domain/entities"
)

type PostOptions struct {
	Status string // "draft" or "publish"; default "publish" if empty
}

// MediaResult represents the result of uploading media to WordPress
type MediaResult struct {
	ID        int    // WordPress media ID
	SourceURL string // Full URL to the media file
	AltText   string // Alt text for the image
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

	// Media methods
	UploadMedia(ctx context.Context, s *entities.Site, filename string, data []byte, altText string) (*MediaResult, error)
	UploadMediaFromURL(ctx context.Context, s *entities.Site, imageURL, filename, altText string) (*MediaResult, error)
	GetMedia(ctx context.Context, s *entities.Site, mediaID int) (*MediaResult, error)
	DeleteMedia(ctx context.Context, s *entities.Site, mediaID int) error

	EnableProxy(proxyURL string)
	DisableProxy()
}
