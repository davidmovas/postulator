package wp

import (
	"context"
	"sync"

	"github.com/davidmovas/postulator/internal/domain/entities"
)

var _ Client = (*client)(nil)

type client struct {
	mu     sync.RWMutex
	client Client
}

func NewClient() Client {
	return &client{
		client: NewRestyClient(),
	}
}

func (c *client) UseRestyClient() Client {
	resty := NewRestyClient()
	c.mu.Lock()
	c.client = resty
	c.mu.Unlock()
	return resty
}

func (c *client) UseCustomClient(client Client) {
	c.mu.Lock()
	c.client = client
	c.mu.Unlock()
}

func (c *client) GetCurrentClient() Client {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.client
}

func (c *client) EnableProxy(proxyURL string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if rc, ok := c.client.(*restyClient); ok {
		rc.WithProxy(proxyURL)
	}
}

func (c *client) DisableProxy() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if rc, ok := c.client.(*restyClient); ok {
		rc.WithoutProxy()
	}
}

func (c *client) CheckHealth(ctx context.Context, site *entities.Site) (*entities.HealthCheck, error) {
	return c.GetCurrentClient().CheckHealth(ctx, site)
}

func (c *client) GetCategories(ctx context.Context, s *entities.Site) ([]*entities.Category, error) {
	return c.GetCurrentClient().GetCategories(ctx, s)
}

func (c *client) CreateCategory(ctx context.Context, s *entities.Site, category *entities.Category) error {
	return c.GetCurrentClient().CreateCategory(ctx, s, category)
}

func (c *client) UpdateCategory(ctx context.Context, s *entities.Site, category *entities.Category) error {
	return c.GetCurrentClient().UpdateCategory(ctx, s, category)
}

func (c *client) DeleteCategory(ctx context.Context, s *entities.Site, wpCategoryID int) error {
	return c.GetCurrentClient().DeleteCategory(ctx, s, wpCategoryID)
}

func (c *client) GetPost(ctx context.Context, s *entities.Site, postID int) (*entities.Article, error) {
	return c.GetCurrentClient().GetPost(ctx, s, postID)
}

func (c *client) GetPosts(ctx context.Context, s *entities.Site) ([]*entities.Article, error) {
	return c.GetCurrentClient().GetPosts(ctx, s)
}

func (c *client) CreatePost(ctx context.Context, s *entities.Site, article *entities.Article, opts *PostOptions) (int, error) {
	return c.GetCurrentClient().CreatePost(ctx, s, article, opts)
}

func (c *client) UpdatePost(ctx context.Context, s *entities.Site, article *entities.Article) error {
	return c.GetCurrentClient().UpdatePost(ctx, s, article)
}

func (c *client) DeletePost(ctx context.Context, s *entities.Site, postID int) error {
	return c.GetCurrentClient().DeletePost(ctx, s, postID)
}

func (c *client) UploadMedia(ctx context.Context, s *entities.Site, filename string, data []byte, altText string) (*MediaResult, error) {
	return c.GetCurrentClient().UploadMedia(ctx, s, filename, data, altText)
}

func (c *client) UploadMediaFromURL(ctx context.Context, s *entities.Site, imageURL, filename, altText string) (*MediaResult, error) {
	return c.GetCurrentClient().UploadMediaFromURL(ctx, s, imageURL, filename, altText)
}

func (c *client) GetMedia(ctx context.Context, s *entities.Site, mediaID int) (*MediaResult, error) {
	return c.GetCurrentClient().GetMedia(ctx, s, mediaID)
}

func (c *client) DeleteMedia(ctx context.Context, s *entities.Site, mediaID int) error {
	return c.GetCurrentClient().DeleteMedia(ctx, s, mediaID)
}

func (c *client) GetPage(ctx context.Context, s *entities.Site, pageID int) (*WPPage, error) {
	return c.GetCurrentClient().GetPage(ctx, s, pageID)
}

func (c *client) GetPages(ctx context.Context, s *entities.Site, opts *PageListOptions) ([]*WPPage, error) {
	return c.GetCurrentClient().GetPages(ctx, s, opts)
}

func (c *client) GetAllPages(ctx context.Context, s *entities.Site) ([]*WPPage, error) {
	return c.GetCurrentClient().GetAllPages(ctx, s)
}

func (c *client) CreatePage(ctx context.Context, s *entities.Site, page *WPPage, opts *PageCreateOptions) (int, error) {
	return c.GetCurrentClient().CreatePage(ctx, s, page, opts)
}

func (c *client) UpdatePage(ctx context.Context, s *entities.Site, page *WPPage) error {
	return c.GetCurrentClient().UpdatePage(ctx, s, page)
}

func (c *client) DeletePage(ctx context.Context, s *entities.Site, pageID int, force bool) error {
	return c.GetCurrentClient().DeletePage(ctx, s, pageID, force)
}
