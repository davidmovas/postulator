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
