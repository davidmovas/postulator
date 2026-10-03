package wptest

import "net/http"

func (s *Server) routeProductCategories(mux *http.ServeMux) {
	mux.HandleFunc("GET "+wooNamespace+"/products/categories", s.withCommerce(func(w http.ResponseWriter, r *http.Request) {
		s.listTerms(w, r, TypeProductCategory)
	}))
	mux.HandleFunc("POST "+wooNamespace+"/products/categories", s.withCommerce(func(w http.ResponseWriter, r *http.Request) {
		s.createTerm(w, r, TypeProductCategory)
	}))
	mux.HandleFunc("GET "+wooNamespace+"/products/categories/{id}", s.withCommerce(s.handleProductCategoryGet))
}

func (s *Server) handleProductCategoryGet(w http.ResponseWriter, r *http.Request) {
	s.handleWooGet(w, r, TypeProductCategory, s.productCategoryPayload)
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
