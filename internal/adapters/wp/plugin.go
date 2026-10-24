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
	codePluginMissing      = "plugin_missing"
	codePluginOutdated     = "plugin_outdated"
	CapabilityPreview      = "preview"
	CapabilitySEOMetaRead  = "seo_meta_read"
	CapabilityRaw          = "raw"
	CapabilityMetaWrite    = "meta_write"
	CapabilityGuardedWrite = "guarded_write"

	fieldSEOTitle         = "title"
	fieldSEODescription   = "description"
	fieldSEOCanonical     = "canonical"
	fieldSEOOGTitle       = "ogTitle"
	fieldSEOOGDescription = "ogDescription"

	defaultContentLimit = 100
	maxContentLimit     = 500

	PluginVersion = "1.5.2"
	PluginMinimum = "1.5.1"
)

var requiredCapabilities = []string{
	"bulk", "seo_meta", CapabilitySEOMetaRead, "content_hash", CapabilityRaw, CapabilityPreview, CapabilityRendered, CapabilityMeta, CapabilityTerms, CapabilitySettings,
	CapabilityMetaWrite, CapabilityGuardedWrite,
}

type PreviewLink struct {
	ExpiresAt time.Time
	URL       string
}

type Manifest struct {
	Version        string
	SEOPlugin      string
	WPVersion      string
	Site           string
	Capabilities   []string
	PostTypes      []string
	Taxonomies     []string
	UnfilteredHTML bool
}

type Capabilities struct {
	Version        string
	SEOPlugin      string
	WPVersion      string
	Site           string
	Names          []string
	PostTypes      []string
	Taxonomies     []string
	UnfilteredHTML bool
}

func (c Capabilities) Has(name string) bool {
	return slices.Contains(c.Names, name)
}

func (c Capabilities) Missing() []string {
	var missing []string
	for _, name := range requiredCapabilities {
		if !c.Has(name) {
			missing = append(missing, name)
		}
	}
	return missing
}

func (c Capabilities) Outdated() bool {
	return versionBelow(c.Version, PluginMinimum) || len(c.Missing()) > 0
}

func versionBelow(version, floor string) bool {
	have, readable := versionNumbers(version)
	if !readable {
		return true
	}
	need, _ := versionNumbers(floor)
	return slices.Compare(have, need) < 0
}

func versionNumbers(version string) ([]int, bool) {
	parts := strings.Split(strings.TrimSpace(version), ".")
	if len(parts) != 3 {
		return nil, false
	}
	numbers := make([]int, 0, len(parts))
	for _, part := range parts {
		number, err := strconv.Atoi(part)
		if err != nil || number < 0 {
			return nil, false
		}
		numbers = append(numbers, number)
	}
	return numbers, true
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
	Excerpt     string
	Meta        ContentMeta
	Links       []ContentLink
	Terms       []Term
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
	Hashes    map[string]string
}

type RawContent struct {
	Type        ItemType
	Content     string
	ContentHash string
	ID          int64
	SaveStable  bool
}

type RawWrite struct {
	Content string
	Hash    string
}

type manifestPayload struct {
	Version        string   `json:"version"`
	SEOPlugin      string   `json:"seoPlugin"`
	WPVersion      string   `json:"wpVersion"`
	Site           string   `json:"site"`
	Capabilities   []string `json:"capabilities"`
	PostTypes      []string `json:"postTypes"`
	Taxonomies     []string `json:"taxonomies"`
	UnfilteredHTML bool     `json:"unfilteredHtml"`
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
	Excerpt     string               `json:"excerpt"`
	Meta        contentMetaPayload   `json:"meta"`
	Links       []contentLinkPayload `json:"links"`
	Terms       []termPayload        `json:"terms"`
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
		Excerpt:     p.Excerpt,
		Meta:        ContentMeta{Title: p.Meta.Title, Description: p.Meta.Description, Canonical: p.Meta.Canonical},
		Links:       links,
		Terms:       termsOf(p.Terms),
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
	SaveStable  bool   `json:"saveStable"`
}

type seoResultPayload struct {
	SEOPlugin string            `json:"seoPlugin"`
	Applied   []string          `json:"applied"`
	Hashes    map[string]string `json:"hashes"`
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
	case fieldSEOTitle:
		return m.Title, true
	case fieldSEODescription:
		return m.Description, true
	case fieldSEOCanonical:
		return m.Canonical, true
	case fieldSEOOGTitle:
		return m.OGTitle, true
	case fieldSEOOGDescription:
		return m.OGDescription, true
	default:
		return "", false
	}
}

func pluginMissing() error {
	return errors.New(errors.Invalid, "the Postulator companion plugin is not installed on this site").
		WithDetail("code", codePluginMissing)
}

func IsPluginMissing(err error) bool {
	return errors.IsCode(err, errors.Invalid) && detailString(err, "code") == codePluginMissing
}

func pluginOutdated(capability string, found Capabilities) error {
	return errors.New(errors.Invalid,
		"the companion plugin on this site is too old for this: upload the new zip over the old one, and WordPress offers to replace the current version with the uploaded one").
		WithDetail("code", codePluginOutdated).
		WithDetail("capability", capability).
		WithDetail("version", found.Version).
		WithDetail("required", PluginVersion)
}

func IsPluginOutdated(err error) bool {
	return errors.IsCode(err, errors.Invalid) && detailString(err, "code") == codePluginOutdated
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
	c.manifest = nil
	c.manifestGone = false
	c.manifestMu.Unlock()

	c.forgetTypes()
}

func (c *Client) Capabilities(ctx context.Context) (Capabilities, error) {
	manifest, err := c.Manifest(ctx)
	if err != nil {
		return Capabilities{}, err
	}

	return Capabilities{
		Version:        manifest.Version,
		SEOPlugin:      manifest.SEOPlugin,
		WPVersion:      manifest.WPVersion,
		Site:           manifest.Site,
		Names:          manifest.Capabilities,
		PostTypes:      manifest.PostTypes,
		Taxonomies:     manifest.Taxonomies,
		UnfilteredHTML: manifest.UnfilteredHTML,
	}, nil
}

func (c *Client) GuardedWrites(ctx context.Context) error {
	return c.requireCapability(ctx, CapabilityGuardedWrite)
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
		return pluginOutdated(name, capabilities)
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

func termRefused(itemType ItemType, id int64) error {
	return errors.New(errors.Invalid,
		"a taxonomy term is no post, and the companion plugin's post routes would answer the post that shares its number").
		WithDetail("type", string(itemType)).
		WithDetail("id", id)
}

func (c *Client) contentPost(ctx context.Context, itemType ItemType, id int64) error {
	switch itemType {
	case TypePage, TypePost, TypeProduct:
		return nil
	case TypeProductCategory:
		return termRefused(itemType, id)
	}

	manifest, err := c.Manifest(ctx)
	if err != nil {
		return err
	}
	switch {
	case slices.Contains(manifest.Taxonomies, string(itemType)):
		return termRefused(itemType, id)
	case slices.Contains(manifest.PostTypes, string(itemType)):
		return nil
	default:
		return errors.New(errors.Invalid, "the site names no content post type of that name").WithDetail("type", string(itemType))
	}
}

func (c *Client) SetSEOMeta(ctx context.Context, itemType ItemType, id int64, meta SEOMeta, expected map[string]string) (SEOResult, error) {
	if err := c.contentPost(ctx, itemType, id); err != nil {
		return SEOResult{}, err
	}
	if err := c.requireCapability(ctx, CapabilityGuardedWrite); err != nil {
		return SEOResult{}, err
	}
	if meta == (SEOMeta{}) {
		return SEOResult{}, errors.New(errors.Invalid, "the SEO meta update carries no fields")
	}

	written := make(map[string]string, len(seoFields))
	for _, name := range seoFields {
		if value, _ := meta.field(name); value != "" {
			written[name] = value
		}
	}
	return c.putSEO(ctx, id, written, expected)
}

var seoFields = []string{fieldSEOTitle, fieldSEODescription, fieldSEOCanonical, fieldSEOOGTitle, fieldSEOOGDescription}

func (c *Client) putSEO(ctx context.Context, id int64, written, expected map[string]string) (SEOResult, error) {
	attributes := make(map[string]any, len(written)+1)
	for name, value := range written {
		attributes[name] = value
	}
	if len(expected) > 0 {
		attributes["expected"] = expected
	}
	body, err := encodeJSON(attributes)
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
	return SEOResult{Applied: payload.Applied, SEOPlugin: payload.SEOPlugin, Hashes: payload.Hashes}, nil
}

func (c *Client) GetSEOMeta(ctx context.Context, itemType ItemType, id int64) (SEOMeta, error) {
	if err := c.contentPost(ctx, itemType, id); err != nil {
		return SEOMeta{}, err
	}
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

func (c *Client) ReplaceSEOMeta(ctx context.Context, itemType ItemType, id int64, meta SEOMeta, fields []string,
	expected map[string]string) (SEOResult, error) {
	if err := c.contentPost(ctx, itemType, id); err != nil {
		return SEOResult{}, err
	}
	if err := c.requireCapability(ctx, CapabilityGuardedWrite); err != nil {
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
	return c.putSEO(ctx, id, written, expected)
}

func (c *Client) GetRaw(ctx context.Context, itemType ItemType, id int64) (RawContent, error) {
	if err := c.contentPost(ctx, itemType, id); err != nil {
		return RawContent{}, err
	}
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
	if ItemType(payload.Type) != itemType {
		return RawContent{}, errors.New(errors.NotFound, "the site holds a "+payload.Type+" under that number, not a "+string(itemType)).
			WithDetail("id", id).
			WithDetail("type", string(itemType)).
			WithDetail("found", payload.Type)
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
		SaveStable:  payload.SaveStable,
	}, nil
}

func (c *Client) PutRaw(ctx context.Context, itemType ItemType, id int64, content, expectedHash string) (RawWrite, error) {
	if err := c.contentPost(ctx, itemType, id); err != nil {
		return RawWrite{}, err
	}
	if err := c.requireCapability(ctx, CapabilityGuardedWrite); err != nil {
		return RawWrite{}, err
	}

	attributes := map[string]any{"content": content}
	if expectedHash != "" {
		attributes["expectedHash"] = expectedHash
	}

	body, err := encodeJSON(attributes)
	if err != nil {
		return RawWrite{}, err
	}

	_, raw, err := c.do(ctx, request{
		method:      http.MethodPut,
		namespace:   pluginNamespace,
		path:        rawPath(id),
		body:        body,
		contentType: contentTypeJSON,
	})
	if err != nil {
		return RawWrite{}, err
	}

	var payload struct {
		Content     string `json:"content"`
		ContentHash string `json:"contentHash"`
	}
	if err := decodeJSON(raw, &payload); err != nil {
		return RawWrite{}, err
	}
	if payload.ContentHash != ContentHash(payload.Content) {
		return RawWrite{}, errors.New(errors.External, "the site answered a stored content whose hash it did not report").
			WithDetail("id", id)
	}
	return RawWrite{Content: payload.Content, Hash: payload.ContentHash}, nil
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

func (c *Client) PreviewLink(ctx context.Context, itemType ItemType, id int64) (PreviewLink, error) {
	if err := c.contentPost(ctx, itemType, id); err != nil {
		return PreviewLink{}, err
	}
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
