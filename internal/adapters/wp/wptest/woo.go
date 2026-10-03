package wptest

import (
	"encoding/json"
	"html"
	"net/http"
	"regexp"
	"strconv"
)

const wooNamespace = "/wp-json/wc/v3"

var (
	scriptPattern    = regexp.MustCompile(`(?is)<script\b[^>]*>.*?</script>`)
	breakPattern     = regexp.MustCompile(`(?i)<br\s*/?>`)
	ampersandPattern = regexp.MustCompile(`&([^a-zA-Z#]|$)`)
)

func (s *Server) routeWoo(mux *http.ServeMux) {
	mux.HandleFunc("GET "+wooNamespace+"/products", s.withCommerce(s.handleProductList))
	mux.HandleFunc("GET "+wooNamespace+"/products/{id}", s.withCommerce(s.handleProductGet))
	mux.HandleFunc("POST "+wooNamespace+"/products/{id}", s.withCommerce(s.handleProductUpdate))
	mux.HandleFunc("PUT "+wooNamespace+"/products/{id}", s.withCommerce(s.handleProductUpdate))
	mux.HandleFunc("GET "+wooNamespace+"/products/categories", s.withCommerce(s.handleProductCategoryList))
	mux.HandleFunc("GET "+wooNamespace+"/products/categories/{id}", s.withCommerce(s.handleProductCategoryGet))
}

func (s *Server) routeStorefront(mux *http.ServeMux) {
	mux.HandleFunc("GET /product/{slug}/", s.handleProductPage)
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
	if shown != nil {
		name, description = shown.Title, shown.Content
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

	structured, err := json.Marshal(map[string]string{
		"@type": "Product", "name": name, "description": tagPattern.ReplaceAllString(description, ""),
	})
	if err != nil {
		s.t.Errorf("encode the structured data of %s: %v", slug, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	body := description
	if builder {
		body = `<div class="builder-layout"><button>Add to cart</button></div>`
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	page := "<!doctype html><html><head><title>" + html.EscapeString(name) + "</title>" +
		`<script type="application/ld+json">` + string(structured) + "</script></head>" +
		"<body><h1>" + html.EscapeString(name) + "</h1>" + body + "</body></html>"
	if _, writeErr := w.Write([]byte(page)); writeErr != nil {
		s.t.Logf("write the product page %s: %v", slug, writeErr)
	}
}

func (s *Server) withCommerce(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		absent := s.noCommerce
		s.mu.Unlock()

		if absent {
			s.fail(w, http.StatusNotFound, "rest_no_route", "No route was found matching the URL and request method.")
			return
		}
		next(w, r)
	}
}

func (s *Server) handleProductList(w http.ResponseWriter, r *http.Request) {
	s.handleWooList(w, r, TypeProduct, s.productPayload)
}

func (s *Server) handleProductCategoryList(w http.ResponseWriter, r *http.Request) {
	s.handleWooList(w, r, TypeProductCategory, s.productCategoryPayload)
}

func (s *Server) handleWooList(w http.ResponseWriter, r *http.Request, itemType string, render func(*Item, bool) map[string]any) {
	query := r.URL.Query()
	page, perPage, bad := listWindow(query)
	if bad != "" {
		s.fail(w, http.StatusBadRequest, "woocommerce_rest_invalid_param", "Invalid parameter(s): "+bad)
		return
	}
	if len(splitList(query.Get("status"))) > 1 {
		s.fail(w, http.StatusBadRequest, "woocommerce_rest_invalid_param", "Invalid parameter(s): status")
		return
	}
	edit := query.Get("context") == "edit"

	s.mu.Lock()
	matched := s.filter(itemType, query)
	total := len(matched)
	totalPages := (total + perPage - 1) / perPage

	start := min((page-1)*perPage, total)
	end := min(start+perPage, total)
	payload := make([]map[string]any, 0, end-start)
	for _, stored := range matched[start:end] {
		payload = append(payload, narrowFields(render(stored, edit), query.Get("_fields")))
	}
	s.mu.Unlock()

	w.Header().Set("X-WP-Total", strconv.Itoa(total))
	w.Header().Set("X-WP-TotalPages", strconv.Itoa(totalPages))
	s.respond(w, http.StatusOK, payload)
}

func (s *Server) handleProductGet(w http.ResponseWriter, r *http.Request) {
	s.handleWooGet(w, r, TypeProduct, s.productPayload)
}

func (s *Server) handleProductCategoryGet(w http.ResponseWriter, r *http.Request) {
	s.handleWooGet(w, r, TypeProductCategory, s.productCategoryPayload)
}

func (s *Server) handleWooGet(w http.ResponseWriter, r *http.Request, itemType string, render func(*Item, bool) map[string]any) {
	id, ok := pathID(r)
	if !ok {
		s.fail(w, http.StatusNotFound, "woocommerce_rest_invalid_id", "Invalid ID.")
		return
	}

	s.mu.Lock()
	store, _ := s.storeOf(itemType)
	stored, found := store[id]
	var payload map[string]any
	if found && stored.Type == itemType {
		payload = render(stored, r.URL.Query().Get("context") == "edit")
	}
	s.mu.Unlock()

	if payload == nil {
		s.fail(w, http.StatusNotFound, "woocommerce_rest_invalid_id", "Invalid ID.")
		return
	}
	s.respond(w, http.StatusOK, narrowFields(payload, r.URL.Query().Get("_fields")))
}

func (s *Server) handleProductUpdate(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r)
	if !ok {
		s.fail(w, http.StatusNotFound, "woocommerce_rest_invalid_id", "Invalid ID.")
		return
	}

	body, ok := s.decodeBody(w, r)
	if !ok {
		return
	}

	s.mu.Lock()
	if s.noProductEdit {
		s.mu.Unlock()
		s.fail(w, http.StatusForbidden, "woocommerce_rest_cannot_edit", "Sorry, you are not allowed to edit this resource.")
		return
	}
	stored, found := s.items[id]
	if !found || stored.Type != TypeProduct {
		s.mu.Unlock()
		s.fail(w, http.StatusNotFound, "woocommerce_rest_invalid_id", "Invalid ID.")
		return
	}

	if s.filteredHTML && resavesPost(body) {
		stored.Content = kses(stored.Content)
	}
	if value, present := body["name"].(string); present {
		stored.Title = kses(value)
	}
	if value, present := body["description"].(string); present {
		stored.Content = kses(value)
	}
	if value, present := body["short_description"].(string); present {
		stored.Excerpt = kses(value)
	}
	if value, present := body["status"].(string); present {
		stored.Status = value
	}
	if slug := stringField(body, "slug"); slug != "" {
		stored.Slug = s.uniqueSlug(slug, stored.Type, stored.Parent, stored.ID)
	}
	if _, present := body["categories"]; present {
		stored.Categories = objectIDList(body, "categories")
	}
	if _, present := body["attributes"]; present {
		stored.Attributes = attributesField(body)
	}
	if _, present := body["images"]; present {
		stored.Images = objectIDList(body, "images")
	}
	stored.Modified = s.tick()
	payload := s.productPayload(stored, true)
	s.mu.Unlock()

	s.respond(w, http.StatusOK, payload)
}

func (s *Server) productPayload(stored *Item, edit bool) map[string]any {
	return map[string]any{
		"id":                stored.ID,
		"name":              stored.Title,
		"slug":              stored.Slug,
		"permalink":         s.permalink(stored),
		"type":              stored.ProductType,
		"status":            stored.Status,
		"description":       rendered(stored.Content, edit),
		"short_description": rendered(stored.Excerpt, edit),
		"regular_price":     stored.RegularPrice,
		"sku":               stored.SKU,
		"menu_order":        stored.MenuOrder,
		"date_modified_gmt": stored.Modified.UTC().Format(wpTimeLayout),
		"categories":        s.categoryRefs(stored.Categories),
		"attributes":        attributePayload(stored.Attributes),
		"images":            s.imagePayload(stored.Images),
	}
}

func (s *Server) productCategoryPayload(stored *Item, _ bool) map[string]any {
	return map[string]any{
		"id":          stored.ID,
		"name":        stored.Title,
		"slug":        stored.Slug,
		"parent":      stored.Parent,
		"description": stored.Content,
		"count":       s.countProducts(stored.ID),
	}
}

func rendered(value string, edit bool) string {
	if edit || value == "" {
		return value
	}
	return value + "\n"
}

func resavesPost(body map[string]any) bool {
	for _, field := range []string{"name", "description", "short_description", "status", "slug"} {
		if _, present := body[field]; present {
			return true
		}
	}
	return false
}

func kses(value string) string {
	cleaned := scriptPattern.ReplaceAllString(value, "")
	cleaned = breakPattern.ReplaceAllString(cleaned, "<br />")
	return ampersandPattern.ReplaceAllString(cleaned, "&amp;$1")
}

func attributePayload(attributes []Attribute) []map[string]any {
	payload := make([]map[string]any, 0, len(attributes))
	for _, attribute := range attributes {
		payload = append(payload, map[string]any{
			"id":        attribute.ID,
			"name":      attribute.Name,
			"position":  attribute.Position,
			"visible":   attribute.Visible,
			"variation": attribute.Variation,
			"options":   append([]string{}, attribute.Options...),
		})
	}
	return payload
}

func attributesField(body map[string]any) []Attribute {
	raw, ok := body["attributes"].([]any)
	if !ok {
		return []Attribute{}
	}

	attributes := make([]Attribute, 0, len(raw))
	for _, entry := range raw {
		object, valid := entry.(map[string]any)
		if !valid {
			continue
		}
		id := intField(object, "id")
		name := stringField(object, "name")
		if id == 0 && name == "" {
			continue
		}
		attributes = append(attributes, Attribute{
			ID:        id,
			Name:      name,
			Position:  int(intField(object, "position")),
			Visible:   boolField(object, "visible"),
			Variation: boolField(object, "variation"),
			Options:   optionsField(object),
		})
	}
	return attributes
}

func boolField(object map[string]any, key string) bool {
	value, ok := object[key].(bool)
	return ok && value
}

func optionsField(object map[string]any) []string {
	raw, ok := object["options"].([]any)
	if !ok {
		return []string{}
	}

	options := make([]string, 0, len(raw))
	for _, entry := range raw {
		text, valid := entry.(string)
		if !valid {
			continue
		}
		options = append(options, tagPattern.ReplaceAllString(text, ""))
	}
	return options
}

func (s *Server) imagePayload(ids []int64) []map[string]any {
	payload := make([]map[string]any, 0, len(ids))
	for _, id := range ids {
		image := map[string]any{"id": id, "src": "", "alt": ""}
		if stored, ok := s.uploads[id]; ok {
			image["src"] = s.http.URL + "/wp-content/uploads/" + stored.Filename
			image["alt"] = stored.Alt
		}
		payload = append(payload, image)
	}
	return payload
}

func (s *Server) categoryRefs(ids []int64) []map[string]any {
	refs := make([]map[string]any, 0, len(ids))
	for _, id := range ids {
		stored, ok := s.terms[id]
		if !ok {
			continue
		}
		refs = append(refs, map[string]any{"id": stored.ID, "name": stored.Title, "slug": stored.Slug})
	}
	return refs
}

func (s *Server) countProducts(categoryID int64) int {
	count := 0
	for _, id := range s.order {
		stored := s.items[id]
		if stored.Type != TypeProduct {
			continue
		}
		for _, assigned := range stored.Categories {
			if assigned == categoryID {
				count++
				break
			}
		}
	}
	return count
}

func objectIDList(body map[string]any, key string) []int64 {
	raw, ok := body[key].([]any)
	if !ok {
		return nil
	}

	ids := make([]int64, 0, len(raw))
	for _, entry := range raw {
		object, valid := entry.(map[string]any)
		if !valid {
			continue
		}
		number, present := object["id"].(float64)
		if !present {
			continue
		}
		ids = append(ids, int64(number))
	}
	return ids
}
