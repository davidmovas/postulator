package wptest

import (
	"cmp"
	"net/http"
	"slices"
	"strconv"
	"strings"
)

const maxMetaKeys = 50

func (s *Server) handlePluginSettings(w http.ResponseWriter, _ *http.Request) {
	if s.pluginMissing(w) {
		return
	}

	s.mu.Lock()
	current, store, absent := s.reading, s.storePages, s.noCommerce
	s.mu.Unlock()

	if current.showOnFront == "" {
		current.showOnFront = "posts"
	}
	if current.showOnFront != "page" {
		current.frontPage, current.postsPage = 0, 0
	}
	if absent {
		store = StorePages{}
	}
	s.respond(w, http.StatusOK, map[string]any{
		"showOnFront": current.showOnFront,
		"pages": map[string]int64{
			"front":    current.frontPage,
			"posts":    current.postsPage,
			"privacy":  current.privacyPage,
			"shop":     store.Shop,
			"cart":     store.Cart,
			"checkout": store.Checkout,
			"account":  store.Account,
			"terms":    store.Terms,
		},
	})
}

func termsPayload(terms []Term) []map[string]any {
	ordered := slices.Clone(terms)
	slices.SortFunc(ordered, func(a, b Term) int {
		return cmp.Or(cmp.Compare(a.Taxonomy, b.Taxonomy), cmp.Compare(a.ID, b.ID))
	})
	payload := make([]map[string]any, 0, len(ordered))
	for _, held := range ordered {
		payload = append(payload, map[string]any{
			"taxonomy": held.Taxonomy,
			"id":       held.ID,
			"name":     held.Name,
			"slug":     held.Slug,
			"parent":   held.Parent,
		})
	}
	return payload
}

func (s *Server) handleTerms(w http.ResponseWriter, r *http.Request) {
	if s.pluginMissing(w) {
		return
	}
	id, ok := pathID(r)
	if !ok {
		s.fail(w, http.StatusNotFound, "not_found", "No content with that id exists.")
		return
	}

	s.mu.Lock()
	stored, found := s.editablePost(id)
	var terms []map[string]any
	if found {
		terms = termsPayload(stored.Terms)
	}
	s.mu.Unlock()

	if !found {
		s.fail(w, http.StatusNotFound, "not_found", "No content with that id exists.")
		return
	}
	s.respond(w, http.StatusOK, map[string]any{"id": id, "terms": terms})
}

func (s *Server) handleMeta(w http.ResponseWriter, r *http.Request) {
	if s.pluginMissing(w) {
		return
	}
	id, ok := pathID(r)
	if !ok {
		s.fail(w, http.StatusNotFound, "not_found", "No content with that id exists.")
		return
	}

	var keys []string
	for _, key := range splitList(r.URL.Query().Get("keys")) {
		if !slices.Contains(keys, key) {
			keys = append(keys, key)
		}
	}
	switch {
	case len(keys) == 0:
		s.fail(w, http.StatusBadRequest, "invalid_param", "keys must name at least one meta key")
		return
	case len(keys) > maxMetaKeys:
		s.fail(w, http.StatusBadRequest, "too_many_keys", "keys names more than "+strconv.Itoa(maxMetaKeys)+" meta keys")
		return
	}
	var denied []string
	for _, key := range keys {
		if metaKeyRefused(key) {
			denied = append(denied, key)
		}
	}
	if len(denied) > 0 {
		s.respond(w, http.StatusBadRequest, map[string]any{
			"code":    "meta_key_denied",
			"message": "a protected meta key is never read: " + strings.Join(denied, ", "),
			"data":    map[string]any{"status": http.StatusBadRequest, "keys": denied},
		})
		return
	}

	s.mu.Lock()
	stored, found := s.editablePost(id)
	values := make(map[string]any, len(keys))
	if found {
		for _, key := range keys {
			if value, held := stored.Meta[key]; held {
				values[key] = value
			} else {
				values[key] = nil
			}
		}
	}
	s.mu.Unlock()

	if !found {
		s.fail(w, http.StatusNotFound, "not_found", "No content with that id exists.")
		return
	}
	s.respond(w, http.StatusOK, map[string]any{"id": id, "meta": values})
}

func (s *Server) handleRendered(w http.ResponseWriter, r *http.Request) {
	if s.pluginMissing(w) {
		return
	}

	id, ok := pathID(r)
	if !ok {
		s.fail(w, http.StatusNotFound, "not_found", "No content with that id exists.")
		return
	}

	s.mu.Lock()
	stored, found := s.editablePost(id)
	var payload map[string]any
	if found {
		filtered := stored.Filtered
		if filtered == "" {
			filtered = stored.Content
		}
		payload = map[string]any{
			"id":          stored.ID,
			"type":        stored.Type,
			"content":     filtered,
			"contentHash": contentHash(filtered),
		}
	}
	s.mu.Unlock()

	if payload == nil {
		s.fail(w, http.StatusNotFound, "not_found", "No content with that id exists.")
		return
	}
	s.respond(w, http.StatusOK, payload)
}
