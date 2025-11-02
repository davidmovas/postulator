package wp

import (
	"context"

	"github.com/davidmovas/postulator/internal/domain/entities"
)

type Client interface {
	// Categories

	GetCategories(ctx context.Context, s *entities.Site) ([]*entities.Category, error)
	CreateCategory(ctx context.Context, s *entities.Site, category *entities.Category) error
	UpdateCategory(ctx context.Context, s *entities.Site, category *entities.Category) error
	DeleteCategory(ctx context.Context, s *entities.Site, wpCategoryID int) error

	// Posts

	GetPost(ctx context.Context, s *entities.Site, postID int) (*entities.Article, error)
	GetPosts(ctx context.Context, s *entities.Site) ([]*entities.Article, error)
	CreatePost(ctx context.Context, s *entities.Site, article *entities.Article) (int, error)
	UpdatePost(ctx context.Context, s *entities.Site, article *entities.Article) error
	DeletePost(ctx context.Context, s *entities.Site, postID int) error
}
