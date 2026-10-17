package wptest

import (
	"encoding/json"
	"html"
	"net/http"
	"slices"
	"strings"
)

type listItem struct {
	Type     string `json:"@type"`
	Name     string `json:"name"`
	Item     string `json:"item,omitempty"`
	Position int    `json:"position"`
}

type breadcrumbList struct {
	Context  string     `json:"@context"`
	Type     string     `json:"@type"`
	Elements []listItem `json:"itemListElement"`
}

func (s *Server) routeStorefront(mux *http.ServeMux) {
	mux.HandleFunc("GET /product/{slug}/", s.handleProductPage)
}

func (s *Server) publicOr(api http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path == rootPath || strings.HasPrefix(r.URL.Path, rootPath+"/") {
			api.ServeHTTP(w, r)
			return
		}
		s.handlePublicPage(w, r)
	})
}

func (s *Server) handlePublicPage(w http.ResponseWriter, r *http.Request) {
	asked := normalisePath(r.URL.Path)

	s.mu.Lock()
	shown, found := s.publicItem(asked)
	down := s.storefrontOff
	var title, body string
	var crumbs []Crumb
	if found {
		title, body, crumbs = shown.Title, shown.Filtered, slices.Clone(shown.Crumbs)
		if body == "" {
			body = shown.Content
		}
	}
	s.mu.Unlock()

	if down {
		http.Error(w, "the storefront is down", http.StatusServiceUnavailable)
		return
	}
	if !found {
		http.NotFound(w, r)
		return
	}

	structured, err := s.breadcrumbList(crumbs)
	if err != nil {
		s.t.Errorf("encode the breadcrumb of %s: %v", asked, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	site := s.http.URL
	page := "<!doctype html><html><head><title>" + html.EscapeString(title) + "</title>" + structured + "</head><body>" +
		`<header><nav aria-label="Primary"><a href="` + site + `/">Home</a><a href="` + site + `/shop/">Shop</a></nav></header>` +
		"<main><article>" + s.breadcrumbNav(crumbs) + "<h1>" + html.EscapeString(title) + "</h1>" + body + "</article></main>" +
		`<footer><nav><a href="` + site + `/privacy-policy/">Privacy Policy</a></nav></footer>` +
		"</body></html>"
	s.writePage(w, asked, page)
}

func (s *Server) publicItem(path string) (*Item, bool) {
	for _, id := range s.order {
		stored := s.items[id]
		if stored.Status != "publish" || stored.Type == TypeProduct || stored.Type == TypeProductVariation {
			continue
		}
		if normalisePath(s.itemPath(stored)) == path {
			return stored, true
		}
	}
	if s.noCommerce {
		return nil, false
	}
	for _, id := range s.termOrder {
		stored := s.terms[id]
		if stored.Type == TypeProductCategory && normalisePath(s.itemPath(stored)) == path {
			return stored, true
		}
	}
	return nil, false
}

func (s *Server) handleProductPage(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")

	s.mu.Lock()
	var shown *Item
	for _, id := range s.order {
		stored := s.items[id]
		if stored.Type == TypeProduct && stored.Slug == slug && stored.Status == "publish" {
			shown = stored
		}
	}
	builder, absent, down := s.builderLayout, s.noCommerce, s.storefrontOff
	var name, description string
	var crumbs []Crumb
	if shown != nil {
		name, description, crumbs = shown.Title, shown.Content, slices.Clone(shown.Crumbs)
	}
	s.mu.Unlock()

	if down {
		http.Error(w, "the storefront is down", http.StatusServiceUnavailable)
		return
	}
	if shown == nil || absent {
		http.NotFound(w, r)
		return
	}

	product, err := json.Marshal(map[string]string{
		"@type": "Product", "name": name, "description": tagPattern.ReplaceAllString(description, ""),
	})
	if err != nil {
		s.t.Errorf("encode the structured data of %s: %v", slug, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	trail, err := s.breadcrumbList(crumbs)
	if err != nil {
		s.t.Errorf("encode the breadcrumb of %s: %v", slug, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	body := description
	if builder {
		body = `<div class="builder-layout"><button>Add to cart</button></div>`
	}
	page := "<!doctype html><html><head><title>" + html.EscapeString(name) + "</title>" +
		`<script type="application/ld+json">` + string(product) + "</script>" + trail + "</head>" +
		"<body>" + s.breadcrumbNav(crumbs) + "<h1>" + html.EscapeString(name) + "</h1>" + body + "</body></html>"
	s.writePage(w, slug, page)
}

func (s *Server) breadcrumbList(crumbs []Crumb) (string, error) {
	if len(crumbs) == 0 {
		return "", nil
	}

	list := breadcrumbList{Context: "https://schema.org", Type: "BreadcrumbList", Elements: make([]listItem, 0, len(crumbs))}
	for index, crumb := range crumbs {
		element := listItem{Type: "ListItem", Name: crumb.Name, Position: index + 1}
		if crumb.Path != "" {
			element.Item = s.http.URL + crumb.Path
		}
		list.Elements = append(list.Elements, element)
	}
	encoded, err := json.Marshal(list)
	if err != nil {
		return "", err
	}
	return `<script type="application/ld+json">` + string(encoded) + "</script>", nil
}

func (s *Server) breadcrumbNav(crumbs []Crumb) string {
	if len(crumbs) == 0 {
		return ""
	}

	var nav strings.Builder
	nav.WriteString(`<nav aria-label="Breadcrumb"><ol>`)
	for index, crumb := range crumbs {
		name := html.EscapeString(crumb.Name)
		switch {
		case index == len(crumbs)-1:
			nav.WriteString(`<li><span aria-current="page">` + name + `</span></li>`)
		case crumb.Path == "":
			nav.WriteString(`<li><span>` + name + `</span></li>`)
		default:
			nav.WriteString(`<li><a href="` + html.EscapeString(s.http.URL+crumb.Path) + `">` + name + `</a></li>`)
		}
	}
	nav.WriteString(`</ol></nav>`)
	return nav.String()
}

func (s *Server) writePage(w http.ResponseWriter, name, page string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if _, err := w.Write([]byte(page)); err != nil {
		s.t.Logf("write the public page %s: %v", name, err)
	}
}
