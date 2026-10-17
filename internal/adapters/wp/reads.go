package wp

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	CapabilityRendered = "rendered"
	CapabilityMeta     = "meta"
	CapabilityTerms    = "terms"
	CapabilitySettings = "settings"

	maxMetaKeys = 50
)

type Rendered struct {
	Type        ItemType
	Content     string
	ContentHash string
	ID          int64
}

type renderedPayload struct {
	Type        string `json:"type"`
	Content     string `json:"content"`
	ContentHash string `json:"contentHash"`
	ID          int64  `json:"id"`
}

func (c *Client) GetRendered(ctx context.Context, itemType ItemType, id int64) (Rendered, error) {
	if err := postRoute(itemType, id); err != nil {
		return Rendered{}, err
	}
	if err := c.requireCapability(ctx, CapabilityRendered); err != nil {
		return Rendered{}, err
	}

	_, body, err := c.do(ctx, request{method: http.MethodGet, namespace: pluginNamespace, path: resourcePath("/content", id) + "/rendered"})
	if err != nil {
		return Rendered{}, err
	}

	var payload renderedPayload
	if err := decodeJSON(body, &payload); err != nil {
		return Rendered{}, err
	}
	if ItemType(payload.Type) != itemType {
		return Rendered{}, errors.New(errors.NotFound, "the site holds a "+payload.Type+" under that number, not a "+string(itemType)).
			WithDetail("id", id).
			WithDetail("type", string(itemType)).
			WithDetail("found", payload.Type)
	}
	if payload.ContentHash != ContentHash(payload.Content) {
		return Rendered{}, errors.New(errors.External, "the site reported a hash that does not match the filtered content it returned").
			WithDetail("id", payload.ID)
	}
	return Rendered{ID: payload.ID, Type: ItemType(payload.Type), Content: payload.Content, ContentHash: payload.ContentHash}, nil
}

type TechnicalPages struct {
	Front    int64
	Posts    int64
	Privacy  int64
	Shop     int64
	Cart     int64
	Checkout int64
	Account  int64
	Terms    int64
}

type PluginSettings struct {
	ShowOnFront string
	Pages       TechnicalPages
}

type pluginSettingsPayload struct {
	ShowOnFront string `json:"showOnFront"`
	Pages       struct {
		Front    int64 `json:"front"`
		Posts    int64 `json:"posts"`
		Privacy  int64 `json:"privacy"`
		Shop     int64 `json:"shop"`
		Cart     int64 `json:"cart"`
		Checkout int64 `json:"checkout"`
		Account  int64 `json:"account"`
		Terms    int64 `json:"terms"`
	} `json:"pages"`
}

func (c *Client) PluginSettings(ctx context.Context) (PluginSettings, error) {
	if err := c.requireCapability(ctx, CapabilitySettings); err != nil {
		return PluginSettings{}, err
	}

	_, body, err := c.do(ctx, request{method: http.MethodGet, namespace: pluginNamespace, path: "/settings"})
	if err != nil {
		return PluginSettings{}, err
	}

	var payload pluginSettingsPayload
	if err := decodeJSON(body, &payload); err != nil {
		return PluginSettings{}, err
	}
	return PluginSettings{ShowOnFront: payload.ShowOnFront, Pages: TechnicalPages(payload.Pages)}, nil
}

type termPayload struct {
	Taxonomy string `json:"taxonomy"`
	Name     string `json:"name"`
	Slug     string `json:"slug"`
	ID       int64  `json:"id"`
	Parent   int64  `json:"parent"`
}

func termsOf(payload []termPayload) []Term {
	terms := make([]Term, 0, len(payload))
	for _, held := range payload {
		terms = append(terms, Term(held))
	}
	return terms
}

func (c *Client) GetTerms(ctx context.Context, itemType ItemType, id int64) ([]Term, error) {
	if err := postRoute(itemType, id); err != nil {
		return nil, err
	}
	if err := c.requireCapability(ctx, CapabilityTerms); err != nil {
		return nil, err
	}

	_, body, err := c.do(ctx, request{method: http.MethodGet, namespace: pluginNamespace, path: resourcePath("/terms", id)})
	if err != nil {
		return nil, err
	}

	var payload struct {
		Terms []termPayload `json:"terms"`
	}
	if err := decodeJSON(body, &payload); err != nil {
		return nil, err
	}
	return termsOf(payload.Terms), nil
}

func metaKeys(keys []string) ([]string, error) {
	var asked []string
	for _, key := range keys {
		trimmed := strings.TrimSpace(key)
		switch {
		case trimmed == "":
			continue
		case strings.Contains(trimmed, ","):
			return nil, errors.New(errors.Invalid, "a meta key holding a comma cannot be asked for").WithDetail("key", trimmed)
		case !slices.Contains(asked, trimmed):
			asked = append(asked, trimmed)
		}
	}
	if len(asked) == 0 {
		return nil, errors.New(errors.Invalid, "a meta read names at least one key")
	}
	if len(asked) > maxMetaKeys {
		return nil, errors.New(errors.Invalid, "a meta read names at most "+strconv.Itoa(maxMetaKeys)+" keys").
			WithDetail("keys", len(asked))
	}
	return asked, nil
}

func (c *Client) GetMeta(ctx context.Context, itemType ItemType, id int64, keys []string) (map[string]json.RawMessage, error) {
	if err := postRoute(itemType, id); err != nil {
		return nil, err
	}
	asked, err := metaKeys(keys)
	if err != nil {
		return nil, err
	}
	if err := c.requireCapability(ctx, CapabilityMeta); err != nil {
		return nil, err
	}

	_, body, err := c.do(ctx, request{
		method:    http.MethodGet,
		namespace: pluginNamespace,
		path:      resourcePath("/meta", id),
		query:     url.Values{"keys": {strings.Join(asked, ",")}},
	})
	if err != nil {
		return nil, err
	}

	var payload struct {
		Meta map[string]json.RawMessage `json:"meta"`
	}
	if err := decodeJSON(body, &payload); err != nil {
		return nil, err
	}
	values := make(map[string]json.RawMessage, len(payload.Meta))
	for key, value := range payload.Meta {
		if len(bytes.TrimSpace(value)) == 0 || bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			continue
		}
		values[key] = value
	}
	return values, nil
}
