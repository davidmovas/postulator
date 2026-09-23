package wptest

import (
	"net/http"
	"strconv"
)

const wooNamespace = "/wp-json/wc/v3"

func (s *Server) routeWoo(mux *http.ServeMux) {
	mux.HandleFunc("GET "+wooNamespace+"/products", s.handleProductList)
	mux.HandleFunc("GET "+wooNamespace+"/products/{id}", s.handleProductGet)
	mux.HandleFunc("POST "+wooNamespace+"/products/{id}", s.handleProductUpdate)
	mux.HandleFunc("GET "+wooNamespace+"/products/categories", s.handleProductCategoryList)
	mux.HandleFunc("GET "+wooNamespace+"/products/categories/{id}", s.handleProductCategoryGet)
}

func (s *Server) handleProductList(w http.ResponseWriter, r *http.Request) {
	s.handleWooList(w, r, TypeProduct, s.productPayload)
}

func (s *Server) handleProductCategoryList(w http.ResponseWriter, r *http.Request) {
	s.handleWooList(w, r, TypeProductCategory, s.productCategoryPayload)
}

func (s *Server) handleWooList(w http.ResponseWriter, r *http.Request, itemType string, render func(*Item) map[string]any) {
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

	s.mu.Lock()
	matched := s.filter(itemType, query)
	total := len(matched)
	totalPages := (total + perPage - 1) / perPage

	start := min((page-1)*perPage, total)
	end := min(start+perPage, total)
	payload := make([]map[string]any, 0, end-start)
	for _, stored := range matched[start:end] {
		payload = append(payload, narrowFields(render(stored), query.Get("_fields")))
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

func (s *Server) handleWooGet(w http.ResponseWriter, r *http.Request, itemType string, render func(*Item) map[string]any) {
	id, ok := pathID(r)
	if !ok {
		s.fail(w, http.StatusNotFound, "woocommerce_rest_invalid_id", "Invalid ID.")
		return
	}

	s.mu.Lock()
	stored, found := s.items[id]
	var payload map[string]any
	if found && stored.Type == itemType {
		payload = render(stored)
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
	stored, found := s.items[id]
	if !found || stored.Type != TypeProduct {
		s.mu.Unlock()
		s.fail(w, http.StatusNotFound, "woocommerce_rest_invalid_id", "Invalid ID.")
		return
	}

	if value, present := body["name"].(string); present {
		stored.Title = value
	}
	if value, present := body["description"].(string); present {
		stored.Content = value
	}
	if value, present := body["short_description"].(string); present {
		stored.Excerpt = value
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
	stored.Modified = s.tick()
	payload := s.productPayload(stored)
	s.mu.Unlock()

	s.respond(w, http.StatusOK, payload)
}

func (s *Server) productPayload(stored *Item) map[string]any {
	return map[string]any{
		"id":                stored.ID,
		"name":              stored.Title,
		"slug":              stored.Slug,
		"permalink":         s.http.URL + s.itemPath(stored),
		"status":            stored.Status,
		"description":       stored.Content,
		"short_description": stored.Excerpt,
		"menu_order":        stored.MenuOrder,
		"date_modified_gmt": stored.Modified.UTC().Format(wpTimeLayout),
		"categories":        s.categoryRefs(stored.Categories),
	}
}

func (s *Server) productCategoryPayload(stored *Item) map[string]any {
	return map[string]any{
		"id":          stored.ID,
		"name":        stored.Title,
		"slug":        stored.Slug,
		"parent":      stored.Parent,
		"description": stored.Content,
		"count":       s.countProducts(stored.ID),
	}
}

func (s *Server) categoryRefs(ids []int64) []map[string]any {
	refs := make([]map[string]any, 0, len(ids))
	for _, id := range ids {
		stored, ok := s.items[id]
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
