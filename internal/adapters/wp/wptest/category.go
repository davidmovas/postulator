package wptest

import "net/http"

func (s *Server) routeCategories(mux *http.ServeMux) {
	mux.HandleFunc("GET "+coreNamespace+"/categories", func(w http.ResponseWriter, r *http.Request) {
		s.listTerms(w, r, taxonomyCategory)
	})
	mux.HandleFunc("POST "+coreNamespace+"/categories", func(w http.ResponseWriter, r *http.Request) {
		s.createTerm(w, r, taxonomyCategory)
	})
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
	s.respond(w, http.StatusOK, narrowFields(payload, r.URL.Query().Get("_fields")))
}

func categoryPayload(stored *Category) map[string]any {
	return map[string]any{
		"id":          stored.ID,
		"name":        stored.Name,
		"slug":        stored.Slug,
		"description": stored.Description,
		"parent":      stored.Parent,
		"count":       stored.Count,
		"taxonomy":    taxonomyCategory,
	}
}
