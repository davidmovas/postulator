package wp

import (
	"cmp"
	"context"
	"net/http"
	"net/url"
	"slices"

	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const coreRestNamespace = "wp/v2"

type PostType struct {
	Slug          string
	Name          string
	RestBase      string
	RestNamespace string
	Taxonomies    []string
	Hierarchical  bool
	Viewable      bool
}

type Term struct {
	Taxonomy string
	Name     string
	Slug     string
	ID       int64
	Parent   int64
}

type SiteSettings struct {
	ShowOnFront string
	FrontPageID int64
	PostsPageID int64
}

type postTypePayload struct {
	Name          string   `json:"name"`
	Slug          string   `json:"slug"`
	RestBase      string   `json:"rest_base"`
	RestNamespace string   `json:"rest_namespace"`
	Taxonomies    []string `json:"taxonomies"`
	Hierarchical  bool     `json:"hierarchical"`
	Viewable      bool     `json:"viewable"`
}

type embedded struct {
	Terms [][]embeddedTermPayload `json:"wp:term"`
}

type embeddedTermPayload struct {
	Taxonomy string `json:"taxonomy"`
	Name     string `json:"name"`
	Slug     string `json:"slug"`
	ID       int64  `json:"id"`
}

type siteSettingsPayload struct {
	ShowOnFront  string `json:"show_on_front"`
	PageOnFront  int64  `json:"page_on_front"`
	PageForPosts int64  `json:"page_for_posts"`
}

func (c *Client) PostTypes(ctx context.Context) ([]PostType, error) {
	c.typesMu.Lock()
	defer c.typesMu.Unlock()

	if c.types != nil {
		return slices.Clone(c.types), nil
	}

	_, body, err := c.do(ctx, request{
		method:    http.MethodGet,
		namespace: coreNamespace,
		path:      "/types",
		query:     url.Values{"context": {"edit"}},
	})
	if err != nil {
		return nil, err
	}

	var payload map[string]postTypePayload
	if err := decodeJSON(body, &payload); err != nil {
		return nil, err
	}

	types := make([]PostType, 0, len(payload))
	for slug, described := range payload {
		if described.Slug == "" {
			described.Slug = slug
		}
		if described.RestNamespace == "" {
			described.RestNamespace = coreRestNamespace
		}
		taxonomies := described.Taxonomies
		if taxonomies == nil {
			taxonomies = []string{}
		}
		types = append(types, PostType{
			Slug:          described.Slug,
			Name:          described.Name,
			RestBase:      described.RestBase,
			RestNamespace: described.RestNamespace,
			Taxonomies:    taxonomies,
			Hierarchical:  described.Hierarchical,
			Viewable:      described.Viewable,
		})
	}
	slices.SortFunc(types, func(a, b PostType) int { return cmp.Compare(a.Slug, b.Slug) })

	c.types = types
	return slices.Clone(types), nil
}

func (c *Client) forgetTypes() {
	c.typesMu.Lock()
	defer c.typesMu.Unlock()

	c.types = nil
}

func (c *Client) readRoute(ctx context.Context, itemType ItemType) (string, error) {
	if path, decided, err := itemType.fixedRoute(); decided {
		return path, err
	}

	types, err := c.PostTypes(ctx)
	if err != nil {
		return "", err
	}
	for _, postType := range types {
		if ItemType(postType.Slug) == itemType && postType.RestNamespace == coreRestNamespace && postType.RestBase != "" {
			return "/" + postType.RestBase, nil
		}
	}
	return "", errors.New(errors.Invalid, "the site has no post type of that name under wp/v2").
		WithDetail("type", string(itemType))
}

func embeddedTerms(groups [][]embeddedTermPayload) []Term {
	var terms []Term
	for _, group := range groups {
		for _, held := range group {
			if held.Taxonomy == "" {
				continue
			}
			terms = append(terms, Term{Taxonomy: held.Taxonomy, Name: held.Name, Slug: held.Slug, ID: held.ID})
		}
	}
	return terms
}

func (c *Client) SiteSettings(ctx context.Context) (SiteSettings, error) {
	_, body, err := c.do(ctx, request{method: http.MethodGet, namespace: coreNamespace, path: "/settings"})
	if err != nil {
		return SiteSettings{}, err
	}

	var payload siteSettingsPayload
	if err := decodeJSON(body, &payload); err != nil {
		return SiteSettings{}, err
	}
	return SiteSettings{ShowOnFront: payload.ShowOnFront, FrontPageID: payload.PageOnFront, PostsPageID: payload.PageForPosts}, nil
}
