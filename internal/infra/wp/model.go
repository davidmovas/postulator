package wp

import (
	"context"
	"time"

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

// WPPage represents a WordPress page
type WPPage struct {
	ID            int       // WordPress page ID
	ParentID      int       // Parent page ID (0 if top-level)
	Title         string    // Page title (rendered)
	Slug          string    // URL slug
	Status        string    // publish, draft, pending, private
	Link          string    // Full URL to the page
	Content       string    // Page content (rendered HTML)
	Excerpt       string    // Page excerpt (rendered)
	Author        int       // Author user ID
	FeaturedMedia int       // Featured image media ID
	MenuOrder     int       // Menu order for sorting
	Template      string    // Page template
	Date          time.Time // Publication date
	Modified      time.Time // Last modified date
}

// PageListOptions configures page listing request
type PageListOptions struct {
	Page       int    // Page number (1-based)
	PerPage    int    // Items per page (max 100)
	Status     string // Filter by status: publish, draft, pending, private, any
	Parent     *int   // Filter by parent ID (nil = all, 0 = top-level only)
	OrderBy    string // Order by: date, id, title, slug, menu_order, parent
	Order      string // Order direction: asc, desc
	Search     string // Search term
	Exclude    []int  // Exclude specific page IDs
	Include    []int  // Include only specific page IDs
}

// PageCreateOptions configures page creation
type PageCreateOptions struct {
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

	// Page methods
	GetPage(ctx context.Context, s *entities.Site, pageID int) (*WPPage, error)
	GetPages(ctx context.Context, s *entities.Site, opts *PageListOptions) ([]*WPPage, error)
	GetAllPages(ctx context.Context, s *entities.Site) ([]*WPPage, error)
	CreatePage(ctx context.Context, s *entities.Site, page *WPPage, opts *PageCreateOptions) (int, error)
	UpdatePage(ctx context.Context, s *entities.Site, page *WPPage) error
	DeletePage(ctx context.Context, s *entities.Site, pageID int, force bool) error

	// Media methods
	UploadMedia(ctx context.Context, s *entities.Site, filename string, data []byte, altText string) (*MediaResult, error)
	UploadMediaFromURL(ctx context.Context, s *entities.Site, imageURL, filename, altText string) (*MediaResult, error)
	GetMedia(ctx context.Context, s *entities.Site, mediaID int) (*MediaResult, error)
	DeleteMedia(ctx context.Context, s *entities.Site, mediaID int) error

	EnableProxy(proxyURL string)
	DisableProxy()
}
