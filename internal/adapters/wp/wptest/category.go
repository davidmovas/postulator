package wptest

import (
	"net/http"
	"slices"
	"strconv"
)

func (s *Server) routeCategories(mux *http.ServeMux) {
	mux.HandleFunc("GET "+coreNamespace+"/categories", s.handleCategoryList)
	mux.HandleFunc("POST "+coreNamespace+"/categories", s.handleCategoryCreate)
	mux.HandleFunc("GET "+coreNamespace+"/categories/{id}", s.handleCategoryGet)
}

func (s *Server) Categories() []Category {
	s.mu.Lock()
	defer s.mu.Unlock()

	categories := make([]Category, 0, len(s.categoryOrder))
	for _, id := range s.categoryOrder {
		categories = append(categories, *s.categories[id])
	}
	return categories
}

func (s *Server) handleCategoryList(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	page, perPage, bad := listWindow(query)
	if bad != "" {
		s.fail(w, http.StatusBadRequest, "rest_invalid_param", "Invalid parameter(s): "+bad)
		return
	}

	s.mu.Lock()
	total := len(s.categoryOrder)
	totalPages := (total + perPage - 1) / perPage
	if total > 0 && page > totalPages {
		s.mu.Unlock()
		s.fail(w, http.StatusBadRequest, "rest_post_invalid_page_number", "The page number requested is larger than the number of pages available.")
		return
	}

	start := min((page-1)*perPage, total)
	end := min(start+perPage, total)
	payload := make([]map[string]any, 0, end-start)
	for _, id := range s.categoryOrder[start:end] {
		payload = append(payload, narrowFields(categoryPayload(s.categories[id]), query.Get("_fields")))
	}
	s.mu.Unlock()

	w.Header().Set("X-WP-Total", strconv.Itoa(total))
	w.Header().Set("X-WP-TotalPages", strconv.Itoa(totalPages))
	s.respond(w, http.StatusOK, payload)
}

func (s *Server) handleCategoryGet(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r)
	if !ok {
		s.fail(w, http.StatusNotFound, "rest_term_invalid", "Term does not exist.")
		return
	}

	s.mu.Lock()
	stored, found := s.categories[id]
	var payload map[string]any
	if found {
		payload = categoryPayload(stored)
	}
	s.mu.Unlock()

	if payload == nil {
		s.fail(w, http.StatusNotFound, "rest_term_invalid", "Term does not exist.")
		return
	}
	s.respond(w, http.StatusOK, payload)
}

func (s *Server) handleCategoryCreate(w http.ResponseWriter, r *http.Request) {
	body, ok := s.decodeBody(w, r)
	if !ok {
		return
	}

	name := stringField(body, "name")
	if name == "" {
		s.fail(w, http.StatusBadRequest, "rest_missing_callback_param", "Missing parameter(s): name")
		return
	}

	s.mu.Lock()
	base := stringField(body, "slug")
	if base == "" {
		base = slugify(name)
	}

	s.nextID++
	stored := &Category{
		ID:          s.nextID,
		Name:        name,
		Slug:        s.uniqueCategorySlug(base),
		Description: stringField(body, "description"),
		Parent:      intField(body, "parent"),
	}
	s.categories[stored.ID] = stored
	s.categoryOrder = append(s.categoryOrder, stored.ID)
	payload := categoryPayload(stored)
	s.mu.Unlock()

	s.respond(w, http.StatusCreated, payload)
}

func (s *Server) uniqueCategorySlug(base string) string {
	taken := make([]string, 0, len(s.categoryOrder))
	for _, id := range s.categoryOrder {
		taken = append(taken, s.categories[id].Slug)
	}

	candidate := base
	for suffix := 2; slices.Contains(taken, candidate); suffix++ {
		candidate = base + "-" + strconv.Itoa(suffix)
	}
	return candidate
}

func categoryPayload(stored *Category) map[string]any {
	return map[string]any{
		"id":          stored.ID,
		"name":        stored.Name,
		"slug":        stored.Slug,
		"description": stored.Description,
		"parent":      stored.Parent,
		"count":       stored.Count,
	}
}
