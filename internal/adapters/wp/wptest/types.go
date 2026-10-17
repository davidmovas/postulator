package wptest

import (
	"net/http"
	"net/url"
	"slices"
	"strings"
)

type Term struct {
	Taxonomy string
	Name     string
	Slug     string
	ID       int64
	Parent   int64
}

type PostType struct {
	Slug         string
	Name         string
	RestBase     string
	Taxonomies   []string
	Hierarchical bool
	Viewable     bool
}

type Taxonomy struct {
	Name        string
	RestBase    string
	ObjectTypes []string
	ShowInREST  bool
}

type reading struct {
	showOnFront string
	frontPage   int64
	postsPage   int64
	privacyPage int64
}

func WithPostType(postType PostType) Option {
	return func(s *Server) { s.postTypes = append(s.postTypes, postType) }
}

func WithTaxonomy(taxonomy Taxonomy) Option {
	return func(s *Server) { s.taxonomies = append(s.taxonomies, taxonomy) }
}

func WithoutSettingsAccess() Option {
	return func(s *Server) { s.noSettings = true }
}

func (s *Server) SetReading(showOnFront string, frontPage, postsPage int64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.reading.showOnFront, s.reading.frontPage, s.reading.postsPage = showOnFront, frontPage, postsPage
}

func (s *Server) SetPrivacyPage(id int64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.reading.privacyPage = id
}

func (s *Server) builtinTypes() []PostType {
	types := []PostType{
		{Slug: "attachment", Name: "Media", RestBase: "media", Viewable: true},
		{Slug: TypePage, Name: "Pages", RestBase: "pages", Hierarchical: true, Viewable: true},
		{Slug: TypePost, Name: "Posts", RestBase: "posts", Taxonomies: []string{"category", "post_tag"}, Viewable: true},
	}
	if !s.noCommerce {
		types = append(types, PostType{Slug: TypeProduct, Name: "Products", RestBase: "product", Taxonomies: []string{"product_cat", "product_tag"}, Viewable: true})
	}
	return types
}

func (s *Server) builtinTaxonomies() []Taxonomy {
	taxonomies := []Taxonomy{
		{Name: "category", RestBase: "categories", ObjectTypes: []string{TypePost}, ShowInREST: true},
		{Name: "post_tag", RestBase: "tags", ObjectTypes: []string{TypePost}, ShowInREST: true},
	}
	if !s.noCommerce {
		taxonomies = append(taxonomies,
			Taxonomy{Name: "product_cat", RestBase: "product_cat", ObjectTypes: []string{TypeProduct}, ShowInREST: true},
			Taxonomy{Name: "product_tag", RestBase: "product_tag", ObjectTypes: []string{TypeProduct}, ShowInREST: true},
		)
	}
	return taxonomies
}

func (s *Server) contentTypes() []string {
	var names []string
	for _, postType := range append(s.builtinTypes(), s.postTypes...) {
		if postType.Slug != "attachment" {
			names = append(names, postType.Slug)
		}
	}
	slices.Sort(names)
	return slices.Compact(names)
}

func (s *Server) contentTaxonomies() []string {
	posts := s.contentTypes()
	var names []string
	for _, taxonomy := range append(s.builtinTaxonomies(), s.taxonomies...) {
		if !slices.Contains(posts, taxonomy.Name) {
			names = append(names, taxonomy.Name)
		}
	}
	slices.Sort(names)
	return slices.Compact(names)
}

func (s *Server) routeTypes(mux *http.ServeMux) {
	mux.HandleFunc("GET "+coreNamespace+"/types", s.handleTypes)
	mux.HandleFunc("GET "+coreNamespace+"/settings", s.handleSettings)

	for _, postType := range s.postTypes {
		itemType := postType.Slug
		mux.HandleFunc("GET "+coreNamespace+"/"+postType.RestBase, func(w http.ResponseWriter, r *http.Request) {
			s.handleList(w, r, itemType)
		})
		mux.HandleFunc("GET "+coreNamespace+"/"+postType.RestBase+"/{id}", func(w http.ResponseWriter, r *http.Request) {
			s.handleGet(w, r, itemType)
		})
	}
}

func (s *Server) handleTypes(w http.ResponseWriter, _ *http.Request) {
	s.mu.Lock()
	types := append(s.builtinTypes(), s.postTypes...)
	s.mu.Unlock()

	payload := make(map[string]any, len(types))
	for _, postType := range types {
		taxonomies := postType.Taxonomies
		if taxonomies == nil {
			taxonomies = []string{}
		}
		payload[postType.Slug] = map[string]any{
			"name":           postType.Name,
			"slug":           postType.Slug,
			"rest_base":      postType.RestBase,
			"rest_namespace": "wp/v2",
			"taxonomies":     taxonomies,
			"hierarchical":   postType.Hierarchical,
			"viewable":       postType.Viewable,
		}
	}
	s.respond(w, http.StatusOK, payload)
}

func (s *Server) handleSettings(w http.ResponseWriter, _ *http.Request) {
	s.mu.Lock()
	refused := s.noSettings
	current := s.reading
	s.mu.Unlock()

	if refused {
		s.fail(w, http.StatusForbidden, "rest_forbidden", "Sorry, you are not allowed to do that.")
		return
	}
	if current.showOnFront == "" {
		current.showOnFront = "posts"
	}
	s.respond(w, http.StatusOK, map[string]any{
		"title":          "Test site",
		"description":    "",
		"url":            s.http.URL,
		"show_on_front":  current.showOnFront,
		"page_on_front":  current.frontPage,
		"page_for_posts": current.postsPage,
	})
}

func embedsTerms(query url.Values) bool {
	if !query.Has("_embed") {
		return false
	}
	value := query.Get("_embed")
	if value == "" || value == "1" || value == "true" {
		return true
	}
	return slices.Contains(splitList(value), "wp:term")
}

func (s *Server) withEmbed(payload map[string]any, stored *Item, query url.Values) map[string]any {
	if !embedsTerms(query) {
		return payload
	}

	groups := make([][]map[string]any, 0)
	links := make([]map[string]any, 0)
	for _, taxonomy := range append(s.builtinTaxonomies(), s.taxonomies...) {
		if !taxonomy.ShowInREST || !slices.Contains(taxonomy.ObjectTypes, stored.Type) {
			continue
		}
		group := make([]map[string]any, 0)
		for _, held := range stored.Terms {
			if held.Taxonomy != taxonomy.Name {
				continue
			}
			group = append(group, map[string]any{
				"id":       held.ID,
				"name":     held.Name,
				"slug":     held.Slug,
				"taxonomy": held.Taxonomy,
				"link":     s.http.URL + "/" + strings.ReplaceAll(taxonomy.Name, "_", "-") + "/" + held.Slug + "/",
			})
		}
		groups = append(groups, group)
		links = append(links, map[string]any{
			"taxonomy":   taxonomy.Name,
			"embeddable": true,
			"href":       s.http.URL + coreNamespace + "/" + taxonomy.RestBase,
		})
	}

	payload["_links"] = map[string]any{"wp:term": links}
	payload["_embedded"] = map[string]any{"wp:term": groups}
	return payload
}
