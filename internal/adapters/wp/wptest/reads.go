package wptest

import "net/http"

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
