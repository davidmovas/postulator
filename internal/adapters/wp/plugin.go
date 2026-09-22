package wp

import (
	"context"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	CodePluginMissing     = "plugin_missing"
	CodePluginOutdated    = "plugin_outdated"
	CapabilityPreview     = "preview"
	CapabilitySEOMetaRead = "seo_meta_read"

	FieldSEOTitle         = "title"
	FieldSEODescription   = "description"
	FieldSEOCanonical     = "canonical"
	FieldSEOOGTitle       = "ogTitle"
	FieldSEOOGDescription = "ogDescription"

	defaultContentLimit = 100
	maxContentLimit     = 500
)

func SEOFields() []string {
	return []string{
		FieldSEOTitle, FieldSEODescription, FieldSEOCanonical, FieldSEOOGTitle, FieldSEOOGDescription,
	}
}

type PreviewLink struct {
	ExpiresAt time.Time
	URL       string
}

type Manifest struct {
	Version      string
	SEOPlugin    string
	WPVersion    string
	Site         string
	Capabilities []string
}

type Capabilities struct {
	Version   string
	SEOPlugin string
	WPVersion string
	Site      string
	Names     []string
}

func (c Capabilities) Has(name string) bool {
	return slices.Contains(c.Names, name)
}

type ContentQuery struct {
	Since  *time.Time
	Cursor string
	Types  []ItemType
	Limit  int
}

func (q ContentQuery) limitValue() int {
	switch {
	case q.Limit <= 0:
		return defaultContentLimit
	case q.Limit > maxContentLimit:
		return maxContentLimit
	default:
		return q.Limit
	}
}

func (q ContentQuery) values() url.Values {
	query := url.Values{}
	if q.Since != nil {
		query.Set("since", q.Since.UTC().Format(time.RFC3339))
	}
	if q.Cursor != "" {
		query.Set("cursor", q.Cursor)
	}
	if len(q.Types) > 0 {
		names := make([]string, 0, len(q.Types))
		for _, itemType := range q.Types {
			names = append(names, string(itemType))
		}
		query.Set("types", strings.Join(names, ","))
	}
	query.Set("limit", strconv.Itoa(q.limitValue()))
	return query
}

type ContentMeta struct {
	Title       string
	Description string
	Canonical   string
}

type ContentLink struct {
	Href   string
	Anchor string
}

type ContentItem struct {
	Modified    time.Time
	Type        ItemType
	Slug        string
	Path        string
	Status      string
	ContentHash string
	Title       string
	H1          string
	Meta        ContentMeta
	Links       []ContentLink
	ID          int64
	Parent      int64
}

type ContentPage struct {
	Items      []ContentItem
	NextCursor *string
}

type SEOMeta struct {
	Title         string `json:"title,omitempty"`
	Description   string `json:"description,omitempty"`
	Canonical     string `json:"canonical,omitempty"`
	OGTitle       string `json:"ogTitle,omitempty"`
	OGDescription string `json:"ogDescription,omitempty"`
}

type SEOResult struct {
	Applied   []string
	SEOPlugin string
}

type RawContent struct {
	Type        ItemType
	Content     string
	ContentHash string
	ID          int64
}

type manifestPayload struct {
	Version      string   `json:"version"`
	SEOPlugin    string   `json:"seoPlugin"`
	WPVersion    string   `json:"wpVersion"`
	Site         string   `json:"site"`
	Capabilities []string `json:"capabilities"`
}

type contentMetaPayload struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Canonical   string `json:"canonical"`
}

type contentLinkPayload struct {
	Href   string `json:"href"`
	Anchor string `json:"anchor"`
}

type contentItemPayload struct {
	Type        string               `json:"type"`
	Slug        string               `json:"slug"`
	Path        string               `json:"path"`
	Status      string               `json:"status"`
	Modified    string               `json:"modified"`
	ContentHash string               `json:"contentHash"`
	Title       string               `json:"title"`
	H1          string               `json:"h1"`
	Meta        contentMetaPayload   `json:"meta"`
	Links       []contentLinkPayload `json:"links"`
	ID          int64                `json:"id"`
	Parent      int64                `json:"parent"`
}

func (p contentItemPayload) contentItem() ContentItem {
	links := make([]ContentLink, 0, len(p.Links))
	for _, link := range p.Links {
		links = append(links, ContentLink(link))
	}

	return ContentItem{
		ID:          p.ID,
		Type:        ItemType(p.Type),
		Slug:        p.Slug,
		Path:        p.Path,
		Parent:      p.Parent,
		Status:      p.Status,
		Modified:    parseWPTime(p.Modified),
		ContentHash: p.ContentHash,
		Title:       p.Title,
		H1:          p.H1,
		Meta:        ContentMeta{Title: p.Meta.Title, Description: p.Meta.Description, Canonical: p.Meta.Canonical},
		Links:       links,
	}
}

type contentPagePayload struct {
	NextCursor *string              `json:"nextCursor"`
	Items      []contentItemPayload `json:"items"`
}

type rawPayload struct {
	Type        string `json:"type"`
	Content     string `json:"content"`
	ContentHash string `json:"contentHash"`
	ID          int64  `json:"id"`
}

type seoResultPayload struct {
	SEOPlugin string   `json:"seoPlugin"`
	Applied   []string `json:"applied"`
}

type seoStatePayload struct {
	Title         string `json:"title"`
	Description   string `json:"description"`
	Canonical     string `json:"canonical"`
	OGTitle       string `json:"ogTitle"`
	OGDescription string `json:"ogDescription"`
}

func (m SEOMeta) field(name string) (string, bool) {
	switch name {
	case FieldSEOTitle:
		return m.Title, true
	case FieldSEODescription:
		return m.Description, true
	case FieldSEOCanonical:
		return m.Canonical, true
	case FieldSEOOGTitle:
		return m.OGTitle, true
	case FieldSEOOGDescription:
		return m.OGDescription, true
	default:
		return "", false
	}
}

func pluginMissing() error {
	return errors.New(errors.Invalid, "the Postulator companion plugin is not installed on this site").
		WithDetail("code", CodePluginMissing)
}

func IsPluginMissing(err error) bool {
	return errors.IsCode(err, errors.Invalid) && detailString(err, "code") == CodePluginMissing
}

func pluginOutdated(capability string) error {
	return errors.New(errors.Invalid, "the Postulator companion plugin on this site is too old for this; update it").
		WithDetail("code", CodePluginOutdated).
		WithDetail("capability", capability)
}

func IsPluginOutdated(err error) bool {
	return errors.IsCode(err, errors.Invalid) && detailString(err, "code") == CodePluginOutdated
}

func (c *Client) Manifest(ctx context.Context) (Manifest, error) {
	c.manifestMu.Lock()
	defer c.manifestMu.Unlock()

	if c.manifestGone {
		return Manifest{}, pluginMissing()
	}
	if c.manifest != nil {
		return *c.manifest, nil
	}

	_, body, err := c.do(ctx, request{method: http.MethodGet, namespace: pluginNamespace, path: "/manifest"})
	if err != nil {
		if errors.IsCode(err, errors.NotFound) {
			c.manifestGone = true
			return Manifest{}, pluginMissing()
		}
		return Manifest{}, err
	}

	var payload manifestPayload
	if err := decodeJSON(body, &payload); err != nil {
		return Manifest{}, err
	}

	manifest := Manifest(payload)
	c.manifest = &manifest
	return manifest, nil
}

func (c *Client) InvalidateManifest() {
	c.manifestMu.Lock()
	defer c.manifestMu.Unlock()

	c.manifest = nil
	c.manifestGone = false
}

func (c *Client) Capabilities(ctx context.Context) (Capabilities, error) {
	manifest, err := c.Manifest(ctx)
	if err != nil {
		return Capabilities{}, err
	}

	return Capabilities{
		Version:   manifest.Version,
		SEOPlugin: manifest.SEOPlugin,
		WPVersion: manifest.WPVersion,
		Site:      manifest.Site,
		Names:     manifest.Capabilities,
	}, nil
}

func (c *Client) requirePlugin(ctx context.Context) error {
	_, err := c.Manifest(ctx)
	return err
}

func (c *Client) requireCapability(ctx context.Context, name string) error {
	capabilities, err := c.Capabilities(ctx)
	if err != nil {
		return err
	}
	if !capabilities.Has(name) {
		return pluginOutdated(name)
	}
	return nil
}

func (c *Client) ListContent(ctx context.Context, query ContentQuery) (ContentPage, error) {
	if err := c.requirePlugin(ctx); err != nil {
		return ContentPage{}, err
	}

	_, body, err := c.do(ctx, request{
		method:    http.MethodGet,
		namespace: pluginNamespace,
		path:      "/content",
		query:     query.values(),
	})
	if err != nil {
		return ContentPage{}, err
	}

	var payload contentPagePayload
	if err := decodeJSON(body, &payload); err != nil {
		return ContentPage{}, err
	}

	items := make([]ContentItem, 0, len(payload.Items))
	for index := range payload.Items {
		items = append(items, payload.Items[index].contentItem())
	}
	return ContentPage{Items: items, NextCursor: payload.NextCursor}, nil
}

func (c *Client) SetSEOMeta(ctx context.Context, id int64, meta SEOMeta) (SEOResult, error) {
	if err := c.requirePlugin(ctx); err != nil {
		return SEOResult{}, err
	}
	if meta == (SEOMeta{}) {
		return SEOResult{}, errors.New(errors.Invalid, "the SEO meta update carries no fields")
	}

	body, err := encodeJSON(meta)
	if err != nil {
		return SEOResult{}, err
	}

	_, raw, err := c.do(ctx, request{
		method:      http.MethodPut,
		namespace:   pluginNamespace,
		path:        resourcePath("/seo-meta", id),
		body:        body,
		contentType: contentTypeJSON,
	})
	if err != nil {
		return SEOResult{}, err
	}

	var payload seoResultPayload
	if err := decodeJSON(raw, &payload); err != nil {
		return SEOResult{}, err
	}
	return SEOResult{Applied: payload.Applied, SEOPlugin: payload.SEOPlugin}, nil
}

func (c *Client) GetSEOMeta(ctx context.Context, id int64) (SEOMeta, error) {
	if err := c.requireCapability(ctx, CapabilitySEOMetaRead); err != nil {
		return SEOMeta{}, err
	}

	_, body, err := c.do(ctx, request{
		method:    http.MethodGet,
		namespace: pluginNamespace,
		path:      resourcePath("/seo-meta", id),
	})
	if err != nil {
		return SEOMeta{}, err
	}

	var payload seoStatePayload
	if err := decodeJSON(body, &payload); err != nil {
		return SEOMeta{}, err
	}
	return SEOMeta(payload), nil
}

func (c *Client) ReplaceSEOMeta(ctx context.Context, id int64, meta SEOMeta, fields []string) (SEOResult, error) {
	if err := c.requirePlugin(ctx); err != nil {
		return SEOResult{}, err
	}

	written := make(map[string]string, len(fields))
	for _, name := range fields {
		value, known := meta.field(name)
		if !known {
			return SEOResult{}, errors.New(errors.Invalid, "the SEO meta has no field called "+name).
				WithDetail("field", name)
		}
		written[name] = value
	}
	if len(written) == 0 {
		return SEOResult{}, errors.New(errors.Invalid, "the SEO meta replacement names no field")
	}

	body, err := encodeJSON(written)
	if err != nil {
		return SEOResult{}, err
	}

	_, raw, err := c.do(ctx, request{
		method:      http.MethodPut,
		namespace:   pluginNamespace,
		path:        resourcePath("/seo-meta", id),
		body:        body,
		contentType: contentTypeJSON,
	})
	if err != nil {
		return SEOResult{}, err
	}

	var payload seoResultPayload
	if err := decodeJSON(raw, &payload); err != nil {
		return SEOResult{}, err
	}
	return SEOResult{Applied: payload.Applied, SEOPlugin: payload.SEOPlugin}, nil
}

func (c *Client) GetRaw(ctx context.Context, id int64) (RawContent, error) {
	if err := c.requirePlugin(ctx); err != nil {
		return RawContent{}, err
	}

	_, body, err := c.do(ctx, request{method: http.MethodGet, namespace: pluginNamespace, path: rawPath(id)})
	if err != nil {
		return RawContent{}, err
	}

	var payload rawPayload
	if err := decodeJSON(body, &payload); err != nil {
		return RawContent{}, err
	}
	if payload.ContentHash != ContentHash(payload.Content) {
		return RawContent{}, errors.New(errors.External, "the site reported a content hash that does not match the content it returned").
			WithDetail("id", payload.ID)
	}

	return RawContent{
		ID:          payload.ID,
		Type:        ItemType(payload.Type),
		Content:     payload.Content,
		ContentHash: payload.ContentHash,
	}, nil
}

func (c *Client) PutRaw(ctx context.Context, id int64, content, expectedHash string) (string, error) {
	if err := c.requirePlugin(ctx); err != nil {
		return "", err
	}

	attributes := map[string]any{"content": content}
	if expectedHash != "" {
		attributes["expectedHash"] = expectedHash
	}

	body, err := encodeJSON(attributes)
	if err != nil {
		return "", err
	}

	_, raw, err := c.do(ctx, request{
		method:      http.MethodPut,
		namespace:   pluginNamespace,
		path:        rawPath(id),
		body:        body,
		contentType: contentTypeJSON,
	})
	if err != nil {
		return "", err
	}

	var payload struct {
		ContentHash string `json:"contentHash"`
	}
	if err := decodeJSON(raw, &payload); err != nil {
		return "", err
	}
	return payload.ContentHash, nil
}

func rawPath(id int64) string {
	return resourcePath("/content", id) + "/raw"
}

func previewPath(id int64) string {
	return resourcePath("/content", id) + "/preview"
}

type previewPayload struct {
	URL       string `json:"url"`
	ExpiresAt string `json:"expiresAt"`
}

func (c *Client) PreviewLink(ctx context.Context, id int64) (PreviewLink, error) {
	if err := c.requireCapability(ctx, CapabilityPreview); err != nil {
		return PreviewLink{}, err
	}

	_, body, err := c.do(ctx, request{method: http.MethodPost, namespace: pluginNamespace, path: previewPath(id)})
	if err != nil {
		return PreviewLink{}, err
	}

	var payload previewPayload
	if err := decodeJSON(body, &payload); err != nil {
		return PreviewLink{}, err
	}

	expires, err := time.Parse(time.RFC3339, payload.ExpiresAt)
	if err != nil {
		return PreviewLink{}, errors.New(errors.External, "the site reported a preview expiry that is not a timestamp").
			WithDetail("id", id)
	}
	parsed, err := url.Parse(payload.URL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return PreviewLink{}, errors.New(errors.External, "the site issued a preview link that is not an absolute address").
			WithDetail("id", id)
	}
	return PreviewLink{URL: payload.URL, ExpiresAt: expires.UTC()}, nil
}
