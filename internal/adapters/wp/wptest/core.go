package wptest

import (
	"encoding/json"
	"maps"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"
)

const (
	coreNamespace  = "/wp-json/wp/v2"
	wpTimeLayout   = "2006-01-02T15:04:05"
	defaultPerPage = 10
	maxPerPage     = 100
)

func (s *Server) routeCore(mux *http.ServeMux) {
	resources := []struct {
		path     string
		itemType string
	}{
		{path: "/pages", itemType: TypePage},
		{path: "/posts", itemType: TypePost},
	}

	for _, resource := range resources {
		mux.HandleFunc("GET "+coreNamespace+resource.path, func(w http.ResponseWriter, r *http.Request) {
			s.handleList(w, r, resource.itemType)
		})
		mux.HandleFunc("POST "+coreNamespace+resource.path, func(w http.ResponseWriter, r *http.Request) {
			s.handleCreate(w, r, resource.itemType)
		})
		mux.HandleFunc("GET "+coreNamespace+resource.path+"/{id}", func(w http.ResponseWriter, r *http.Request) {
			s.handleGet(w, r, resource.itemType)
		})
		mux.HandleFunc("POST "+coreNamespace+resource.path+"/{id}", func(w http.ResponseWriter, r *http.Request) {
			s.handleUpdate(w, r, resource.itemType)
		})
		mux.HandleFunc("DELETE "+coreNamespace+resource.path+"/{id}", func(w http.ResponseWriter, r *http.Request) {
			s.handleDelete(w, r, resource.itemType)
		})
	}
}

func (s *Server) handleList(w http.ResponseWriter, r *http.Request, itemType string) {
	query := r.URL.Query()
	page, perPage, bad := listWindow(query)
	if bad != "" {
		s.fail(w, http.StatusBadRequest, "rest_invalid_param", "Invalid parameter(s): "+bad)
		return
	}

	s.mu.Lock()
	matched := s.filter(itemType, query)
	total := len(matched)
	totalPages := (total + perPage - 1) / perPage

	if total > 0 && page > totalPages {
		s.mu.Unlock()
		s.fail(w, http.StatusBadRequest, "rest_post_invalid_page_number", "The page number requested is larger than the number of pages available.")
		return
	}

	start := min((page-1)*perPage, total)
	end := min(start+perPage, total)
	payload := make([]map[string]any, 0, end-start)
	for _, stored := range matched[start:end] {
		payload = append(payload, narrowFields(s.itemPayload(stored), query.Get("_fields")))
	}
	s.mu.Unlock()

	w.Header().Set("X-WP-Total", strconv.Itoa(total))
	w.Header().Set("X-WP-TotalPages", strconv.Itoa(totalPages))
	s.respond(w, http.StatusOK, payload)
}

func (s *Server) handleGet(w http.ResponseWriter, r *http.Request, itemType string) {
	id, ok := pathID(r)
	if !ok {
		s.fail(w, http.StatusNotFound, "rest_post_invalid_id", "Invalid post ID.")
		return
	}

	s.mu.Lock()
	stored, found := s.items[id]
	var payload map[string]any
	if found && stored.Type == itemType {
		payload = s.itemPayload(stored)
	}
	s.mu.Unlock()

	if payload == nil {
		s.fail(w, http.StatusNotFound, "rest_post_invalid_id", "Invalid post ID.")
		return
	}
	s.respond(w, http.StatusOK, narrowFields(payload, r.URL.Query().Get("_fields")))
}

func (s *Server) handleCreate(w http.ResponseWriter, r *http.Request, itemType string) {
	body, ok := s.decodeBody(w, r)
	if !ok {
		return
	}

	s.mu.Lock()
	created := s.add(Item{
		Type:          itemType,
		Title:         stringField(body, "title"),
		Content:       stringField(body, "content"),
		Excerpt:       stringField(body, "excerpt"),
		Slug:          stringField(body, "slug"),
		Status:        stringField(body, "status"),
		Template:      stringField(body, "template"),
		Parent:        intField(body, "parent"),
		MenuOrder:     int(intField(body, "menu_order")),
		FeaturedMedia: intField(body, "featured_media"),
		Categories:    intListField(body, "categories"),
		Tags:          intListField(body, "tags"),
		Meta:          metaField(body),
	})
	payload := s.itemPayload(s.items[created.ID])
	s.mu.Unlock()

	s.respond(w, http.StatusCreated, payload)
}

func (s *Server) handleUpdate(w http.ResponseWriter, r *http.Request, itemType string) {
	id, ok := pathID(r)
	if !ok {
		s.fail(w, http.StatusNotFound, "rest_post_invalid_id", "Invalid post ID.")
		return
	}

	body, ok := s.decodeBody(w, r)
	if !ok {
		return
	}

	s.mu.Lock()
	stored, found := s.items[id]
	if !found || stored.Type != itemType {
		s.mu.Unlock()
		s.fail(w, http.StatusNotFound, "rest_post_invalid_id", "Invalid post ID.")
		return
	}

	applyUpdate(stored, body)
	if slug := stringField(body, "slug"); slug != "" {
		stored.Slug = s.uniqueSlug(slug, stored.Type, stored.Parent, stored.ID)
	}
	stored.Modified = s.tick()
	payload := s.itemPayload(stored)
	s.mu.Unlock()

	s.respond(w, http.StatusOK, payload)
}

func (s *Server) handleDelete(w http.ResponseWriter, r *http.Request, itemType string) {
	id, ok := pathID(r)
	if !ok {
		s.fail(w, http.StatusNotFound, "rest_post_invalid_id", "Invalid post ID.")
		return
	}
	forced := r.URL.Query().Get("force") == "true"

	s.mu.Lock()
	stored, found := s.items[id]
	if !found || stored.Type != itemType {
		s.mu.Unlock()
		s.fail(w, http.StatusNotFound, "rest_post_invalid_id", "Invalid post ID.")
		return
	}

	var payload map[string]any
	if forced {
		payload = map[string]any{"deleted": true, "previous": s.itemPayload(stored)}
		delete(s.items, id)
		s.order = slices.DeleteFunc(s.order, func(other int64) bool { return other == id })
	} else {
		stored.Status = "trash"
		stored.Modified = s.tick()
		payload = s.itemPayload(stored)
	}
	s.mu.Unlock()

	s.respond(w, http.StatusOK, payload)
}

func (s *Server) decodeBody(w http.ResponseWriter, r *http.Request) (map[string]any, bool) {
	var body map[string]any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		s.fail(w, http.StatusBadRequest, "rest_invalid_json", "The request body is not valid JSON.")
		return nil, false
	}
	return body, true
}

func (s *Server) filter(itemType string, query url.Values) []*Item {
	statuses := splitList(query.Get("status"))
	after := parseQueryTime(query.Get("modified_after"))

	matched := make([]*Item, 0, len(s.order))
	for _, id := range s.order {
		stored := s.items[id]
		if stored.Type != itemType {
			continue
		}
		if len(statuses) > 0 && !slices.Contains(statuses, stored.Status) {
			continue
		}
		if !after.IsZero() && !stored.Modified.After(after) {
			continue
		}
		matched = append(matched, stored)
	}
	return matched
}

func (s *Server) itemPayload(stored *Item) map[string]any {
	stamp := stored.Modified.UTC().Format(wpTimeLayout)
	return map[string]any{
		"id":             stored.ID,
		"type":           stored.Type,
		"slug":           stored.Slug,
		"status":         stored.Status,
		"link":           s.http.URL + s.itemPath(stored),
		"parent":         stored.Parent,
		"menu_order":     stored.MenuOrder,
		"template":       stored.Template,
		"categories":     idList(stored.Categories),
		"tags":           idList(stored.Tags),
		"featured_media": stored.FeaturedMedia,
		"modified":       stamp,
		"modified_gmt":   stamp,
		"title":          renderedField(stored.Title),
		"content":        renderedField(stored.Content),
		"excerpt":        renderedField(stored.Excerpt),
		"meta":           metaPayload(stored.Meta),
	}
}

func applyUpdate(stored *Item, body map[string]any) {
	if value, ok := body["title"].(string); ok {
		stored.Title = value
	}
	if value, ok := body["content"].(string); ok {
		stored.Content = value
	}
	if value, ok := body["excerpt"].(string); ok {
		stored.Excerpt = value
	}
	if value, ok := body["status"].(string); ok {
		stored.Status = value
	}
	if value, ok := body["template"].(string); ok {
		stored.Template = value
	}
	if value, ok := body["parent"].(float64); ok {
		stored.Parent = int64(value)
	}
	if value, ok := body["menu_order"].(float64); ok {
		stored.MenuOrder = int(value)
	}
	if value, ok := body["featured_media"].(float64); ok {
		stored.FeaturedMedia = int64(value)
	}
	if _, ok := body["categories"]; ok {
		stored.Categories = intListField(body, "categories")
	}
	if _, ok := body["tags"]; ok {
		stored.Tags = intListField(body, "tags")
	}
	if meta := metaField(body); meta != nil {
		maps.Copy(stored.Meta, meta)
	}
}

func renderedField(value string) map[string]any {
	return map[string]any{"raw": value, "rendered": value}
}

func metaPayload(meta map[string]string) any {
	if len(meta) == 0 {
		return []any{}
	}

	payload := make(map[string]any, len(meta))
	for key, value := range meta {
		payload[key] = value
	}
	return payload
}

func idList(ids []int64) []int64 {
	if ids == nil {
		return []int64{}
	}
	return ids
}

func narrowFields(item map[string]any, fields string) map[string]any {
	wanted := splitList(fields)
	if len(wanted) == 0 {
		return item
	}

	narrow := make(map[string]any, len(wanted))
	for _, field := range wanted {
		if value, ok := item[field]; ok {
			narrow[field] = value
		}
	}
	return narrow
}

func listWindow(query url.Values) (page, perPage int, invalid string) {
	page = 1
	if raw := query.Get("page"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 {
			return 0, 0, "page"
		}
		page = parsed
	}

	perPage = defaultPerPage
	if raw := query.Get("per_page"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > maxPerPage {
			return 0, 0, "per_page"
		}
		perPage = parsed
	}
	return page, perPage, ""
}

func pathID(r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		return 0, false
	}
	return id, true
}

func stringField(body map[string]any, key string) string {
	value, ok := body[key].(string)
	if !ok {
		return ""
	}
	return value
}

func intField(body map[string]any, key string) int64 {
	value, ok := body[key].(float64)
	if !ok {
		return 0
	}
	return int64(value)
}

func intListField(body map[string]any, key string) []int64 {
	raw, ok := body[key].([]any)
	if !ok {
		return nil
	}

	ids := make([]int64, 0, len(raw))
	for _, entry := range raw {
		number, valid := entry.(float64)
		if !valid {
			continue
		}
		ids = append(ids, int64(number))
	}
	return ids
}

func metaField(body map[string]any) map[string]string {
	raw, ok := body["meta"].(map[string]any)
	if !ok {
		return nil
	}

	meta := make(map[string]string, len(raw))
	for key, value := range raw {
		text, valid := value.(string)
		if !valid {
			continue
		}
		meta[key] = text
	}
	return meta
}

func splitList(value string) []string {
	if value == "" {
		return nil
	}

	parts := strings.Split(value, ",")
	trimmed := make([]string, 0, len(parts))
	for _, part := range parts {
		cleaned := strings.TrimSpace(part)
		if cleaned != "" {
			trimmed = append(trimmed, cleaned)
		}
	}
	return trimmed
}

func parseQueryTime(value string) time.Time {
	if value == "" {
		return time.Time{}
	}
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return parsed.UTC()
	}
	if parsed, err := time.Parse(wpTimeLayout, value); err == nil {
		return parsed.UTC()
	}
	return time.Time{}
}
