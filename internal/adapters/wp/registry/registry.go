package registry

import (
	"context"
	"sync"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/wp"
	"github.com/davidmovas/postulator/internal/domain/site"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

type siteReader interface {
	Get(ctx context.Context, id string) (site.Site, error)
}

type secretStore interface {
	Get(ctx context.Context, ref string) (string, error)
}

type entry struct {
	client  *wp.Client
	stamp   time.Time
	baseURL string
}

type Registry struct {
	sites   siteReader
	secrets secretStore
	options []wp.Option
	cached  map[string]entry
	mu      sync.Mutex
}

func New(sites siteReader, secrets secretStore, options ...wp.Option) *Registry {
	return &Registry{sites: sites, secrets: secrets, options: options, cached: make(map[string]entry)}
}

func (r *Registry) Client(ctx context.Context, siteID string) (*wp.Client, error) {
	record, err := r.sites.Get(ctx, siteID)
	if err != nil {
		return nil, err
	}

	r.mu.Lock()
	held, ok := r.cached[siteID]
	r.mu.Unlock()
	if ok && held.stamp.Equal(record.UpdatedAt) && held.baseURL == record.BaseURL {
		return held.client, nil
	}

	password, err := r.secrets.Get(ctx, record.SecretRef)
	if err != nil {
		if errors.IsCode(err, errors.NotFound) {
			return nil, errors.New(errors.Unauthorized, "the site has no stored application password").
				WithDetail("siteId", siteID)
		}
		return nil, err
	}

	client, err := wp.New(wp.Config{
		BaseURL:       record.BaseURL,
		Username:      record.Username,
		AppPassword:   password,
		AllowInsecure: record.AllowInsecure,
	}, r.options...)
	if err != nil {
		return nil, err
	}

	r.mu.Lock()
	r.cached[siteID] = entry{client: client, stamp: record.UpdatedAt, baseURL: record.BaseURL}
	r.mu.Unlock()
	return client, nil
}

func (r *Registry) Probe(ctx context.Context, record site.Site) (site.PluginState, error) {
	client, err := r.Client(ctx, record.ID)
	if err != nil {
		return site.PluginState{}, err
	}

	client.InvalidateManifest()

	capabilities, err := client.Capabilities(ctx)
	if err != nil {
		if wp.IsPluginMissing(err) {
			return site.PluginState{Capabilities: []string{}}, nil
		}
		return site.PluginState{}, err
	}

	return site.PluginState{
		Installed:    true,
		Version:      capabilities.Version,
		Capabilities: capabilities.Names,
		SEOPlugin:    capabilities.SEOPlugin,
	}, nil
}
