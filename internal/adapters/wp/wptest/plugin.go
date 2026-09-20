package wptest

import (
	"cmp"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
)

const (
	pluginNamespace     = "/wp-json/postulator/v1"
	defaultContentLimit = 100
	maxContentLimit     = 500
)

var seoFieldOrder = []string{"title", "description", "canonical", "ogTitle", "ogDescription"}

var seoKeys = map[string]map[string]string{
	"yoast": {
		"title":         "_yoast_wpseo_title",
		"description":   "_yoast_wpseo_metadesc",
		"canonical":     "_yoast_wpseo_canonical",
		"ogTitle":       "_yoast_wpseo_opengraph-title",
		"ogDescription": "_yoast_wpseo_opengraph-description",
	},
	"rankmath": {
		"title":         "rank_math_title",
		"description":   "rank_math_description",
		"canonical":     "rank_math_canonical_url",
		"ogTitle":       "rank_math_facebook_title",
		"ogDescription": "rank_math_facebook_description",
	},
	"none": {
		"title":         "_postulator_title",
		"description":   "_postulator_description",
		"canonical":     "_postulator_canonical",
		"ogTitle":       "_postulator_og_title",
		"ogDescription": "_postulator_og_description",
	},
}

var (
	headingPattern = regexp.MustCompile(`(?is)<h1[^>]*>(.*?)</h1>`)
	linkPattern    = regexp.MustCompile(`(?is)<a\s[^>]*href="([^"]*)"[^>]*>(.*?)</a>`)
	tagPattern     = regexp.MustCompile(`(?s)<[^>]*>`)
)

type cursor struct {
	Modified string `json:"m"`
	ID       int64  `json:"i"`
}

func (s *Server) routePlugin(mux *http.ServeMux) {
	mux.HandleFunc("GET "+pluginNamespace+"/manifest", s.handleManifest)
	mux.HandleFunc("GET "+pluginNamespace+"/content", s.handleContent)
	mux.HandleFunc("PUT "+pluginNamespace+"/seo-meta/{id}", s.handleSEOMeta)
	mux.HandleFunc("GET "+pluginNamespace+"/content/{id}/raw", s.handleRawGet)
	mux.HandleFunc("PUT "+pluginNamespace+"/content/{id}/raw", s.handleRawPut)
	mux.HandleFunc("POST "+pluginNamespace+"/content/{id}/preview", s.handlePreview)
}

func (s *Server) pluginMissing(w http.ResponseWriter) bool {
	s.mu.Lock()
	missing := s.noPlugin
	s.mu.Unlock()

	if missing {
		s.fail(w, http.StatusNotFound, "rest_no_route", "No route was found matching the URL and request method")
	}
	return missing
}

func (s *Server) handleManifest(w http.ResponseWriter, _ *http.Request) {
	if s.pluginMissing(w) {
		return
	}

	s.mu.Lock()
	plugin := s.seoPlugin
	capabilities := slices.Clone(s.capabilities)
	s.mu.Unlock()

	s.respond(w, http.StatusOK, map[string]any{
		"version":      "1.1.0",
		"capabilities": capabilities,
		"seoPlugin":    plugin,
		"wpVersion":    "6.9.1",
		"site":         s.http.URL,
	})
}

func (s *Server) handleContent(w http.ResponseWriter, r *http.Request) {
	if s.pluginMissing(w) {
		return
	}

	query := r.URL.Query()
	types := splitList(query.Get("types"))
	if len(types) == 0 {
		types = []string{TypePage, TypePost, TypeProduct, TypeProductCategory}
	}

	after, ok := decodeCursor(query.Get("cursor"))
	if !ok {
		s.fail(w, http.StatusBadRequest, "invalid_cursor", "The cursor could not be read.")
		return
	}

	since := parseQueryTime(query.Get("since"))
	limit := contentLimit(query.Get("limit"))

	s.mu.Lock()
	matched := make([]*Item, 0, len(s.order))
	for _, id := range s.order {
		stored := s.items[id]
		if !slices.Contains(types, stored.Type) {
			continue
		}
		if !since.IsZero() && !stored.Modified.After(since) {
			continue
		}
		if !afterCursor(stored, after) {
			continue
		}
		matched = append(matched, stored)
	}
	slices.SortFunc(matched, func(left, right *Item) int {
		if left.Modified.Equal(right.Modified) {
			return cmp.Compare(left.ID, right.ID)
		}
		return left.Modified.Compare(right.Modified)
	})

	var next any
	if len(matched) > limit {
		matched = matched[:limit]
		last := matched[len(matched)-1]

		encoded, err := encodeCursor(cursor{Modified: last.Modified.UTC().Format(time.RFC3339), ID: last.ID})
		if err != nil {
			s.mu.Unlock()
			s.fail(w, http.StatusInternalServerError, "cursor_failed", "The next cursor could not be built.")
			return
		}
		next = encoded
	}

	items := make([]map[string]any, 0, len(matched))
	for _, stored := range matched {
		items = append(items, s.contentItem(stored))
	}
	s.mu.Unlock()

	s.respond(w, http.StatusOK, map[string]any{"items": items, "nextCursor": next})
}

func (s *Server) handleSEOMeta(w http.ResponseWriter, r *http.Request) {
	if s.pluginMissing(w) {
		return
	}

	id, ok := pathID(r)
	if !ok {
		s.fail(w, http.StatusNotFound, "not_found", "No content with that id exists.")
		return
	}

	body, ok := s.decodeBody(w, r)
	if !ok {
		return
	}

	s.mu.Lock()
	stored, found := s.items[id]
	if !found || !postType(stored.Type) {
		s.mu.Unlock()
		s.fail(w, http.StatusNotFound, "not_found", "No content with that id exists.")
		return
	}

	keys := seoKeys[s.seoPlugin]
	applied := make([]string, 0, len(seoFieldOrder))
	for _, field := range seoFieldOrder {
		value := stringField(body, field)
		if value == "" {
			continue
		}
		stored.Meta[keys[field]] = value
		applied = append(applied, field)
	}
	plugin := s.seoPlugin
	s.mu.Unlock()

	s.respond(w, http.StatusOK, map[string]any{"applied": applied, "seoPlugin": plugin})
}

func (s *Server) handleRawGet(w http.ResponseWriter, r *http.Request) {
	if s.pluginMissing(w) {
		return
	}

	id, ok := pathID(r)
	if !ok {
		s.fail(w, http.StatusNotFound, "not_found", "No content with that id exists.")
		return
	}

	s.mu.Lock()
	stored, found := s.items[id]
	var payload map[string]any
	if found && postType(stored.Type) {
		payload = map[string]any{
			"id":          stored.ID,
			"type":        stored.Type,
			"content":     stored.Content,
			"contentHash": s.reportedHash(stored.Content),
		}
	}
	s.mu.Unlock()

	if payload == nil {
		s.fail(w, http.StatusNotFound, "not_found", "No content with that id exists.")
		return
	}
	s.respond(w, http.StatusOK, payload)
}

func (s *Server) handleRawPut(w http.ResponseWriter, r *http.Request) {
	if s.pluginMissing(w) {
		return
	}

	id, ok := pathID(r)
	if !ok {
		s.fail(w, http.StatusNotFound, "not_found", "No content with that id exists.")
		return
	}

	body, ok := s.decodeBody(w, r)
	if !ok {
		return
	}

	content, present := body["content"].(string)
	if !present {
		s.fail(w, http.StatusBadRequest, "missing_content", "The content field is required.")
		return
	}
	expected := stringField(body, "expectedHash")
	_, _ = s.takePendingEdit()

	s.mu.Lock()
	stored, found := s.items[id]
	if !found || !postType(stored.Type) {
		s.mu.Unlock()
		s.fail(w, http.StatusNotFound, "not_found", "No content with that id exists.")
		return
	}

	current := contentHash(stored.Content)
	if expected != "" && expected != current {
		s.mu.Unlock()
		s.respond(w, http.StatusConflict, map[string]any{
			"code":        "hash_mismatch",
			"message":     "The stored content changed since it was read.",
			"currentHash": current,
		})
		return
	}

	stored.Content = content
	stored.Modified = s.tick()
	updated := contentHash(content)
	s.mu.Unlock()

	s.respond(w, http.StatusOK, map[string]any{"contentHash": updated})
}

func (s *Server) contentItem(stored *Item) map[string]any {
	keys := seoKeys[s.seoPlugin]
	return map[string]any{
		"id":          stored.ID,
		"type":        stored.Type,
		"slug":        stored.Slug,
		"path":        normalisePath(s.itemPath(stored)),
		"parent":      stored.Parent,
		"status":      stored.Status,
		"modified":    stored.Modified.UTC().Format(time.RFC3339),
		"contentHash": contentHash(stored.Content),
		"title":       stored.Title,
		"h1":          headingOne(stored),
		"meta": map[string]any{
			"title":       stored.Meta[keys["title"]],
			"description": stored.Meta[keys["description"]],
			"canonical":   stored.Meta[keys["canonical"]],
		},
		"links": s.internalLinks(stored.Content),
	}
}

func postType(itemType string) bool {
	return itemType != TypeProductCategory
}

func (s *Server) reportedHash(content string) string {
	if s.brokenHash {
		return strings.Repeat("0", 64)
	}
	return contentHash(content)
}

func headingOne(stored *Item) string {
	if stored.H1 != "" {
		return stored.H1
	}

	match := headingPattern.FindStringSubmatch(stored.Content)
	if match == nil {
		return ""
	}
	return strings.TrimSpace(tagPattern.ReplaceAllString(match[1], ""))
}

func (s *Server) internalLinks(content string) []map[string]any {
	host := hostOf(s.http.URL)
	links := make([]map[string]any, 0)
	for _, match := range linkPattern.FindAllStringSubmatch(content, -1) {
		target, ok := internalTarget(host, match[1])
		if !ok {
			continue
		}
		links = append(links, map[string]any{
			"href":   target,
			"anchor": strings.TrimSpace(tagPattern.ReplaceAllString(match[2], "")),
		})
	}
	return links
}

func hostOf(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return parsed.Host
}

func internalTarget(host, href string) (string, bool) {
	trimmed := strings.TrimSpace(href)
	if trimmed == "" {
		return "", false
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return "", false
	}
	if parsed.Scheme != "" && parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", false
	}
	if parsed.Host != "" && !strings.EqualFold(parsed.Host, host) {
		return "", false
	}

	target := parsed.EscapedPath()
	if target == "" && parsed.Host == "" {
		return "", false
	}
	return normalisePath(target), true
}

func normalisePath(value string) string {
	var builder strings.Builder
	builder.Grow(len(value) + 2)
	builder.WriteByte('/')

	slashed := true
	for index := range len(value) {
		symbol := value[index]
		if symbol == '/' {
			if slashed {
				continue
			}
			slashed = true
			builder.WriteByte('/')
			continue
		}
		slashed = false
		if symbol >= 'A' && symbol <= 'Z' {
			symbol += 'a' - 'A'
		}
		builder.WriteByte(symbol)
	}

	normalised := builder.String()
	if strings.HasSuffix(normalised, "/") {
		return normalised
	}
	return normalised + "/"
}

func contentLimit(raw string) int {
	if raw == "" {
		return defaultContentLimit
	}

	parsed, err := strconv.Atoi(raw)
	if err != nil || parsed <= 0 {
		return defaultContentLimit
	}
	if parsed > maxContentLimit {
		return maxContentLimit
	}
	return parsed
}

func encodeCursor(value cursor) (string, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func decodeCursor(value string) (cursor, bool) {
	if value == "" {
		return cursor{}, true
	}

	raw, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return cursor{}, false
	}

	var decoded cursor
	if err = json.Unmarshal(raw, &decoded); err != nil {
		return cursor{}, false
	}
	return decoded, true
}

func afterCursor(stored *Item, after cursor) bool {
	if after.Modified == "" {
		return true
	}

	bound, err := time.Parse(time.RFC3339, after.Modified)
	if err != nil {
		return true
	}
	if stored.Modified.After(bound) {
		return true
	}
	return stored.Modified.Equal(bound) && stored.ID > after.ID
}

const previewLifetime = time.Hour

func (s *Server) handlePreview(w http.ResponseWriter, r *http.Request) {
	if s.pluginMissing(w) {
		return
	}

	id, ok := pathID(r)
	if !ok {
		s.fail(w, http.StatusNotFound, "not_found", "No content with that id exists.")
		return
	}

	s.mu.Lock()
	stored, found := s.items[id]
	if !found || !postType(stored.Type) {
		s.mu.Unlock()
		s.fail(w, http.StatusNotFound, "not_found", "No content with that id exists.")
		return
	}
	s.previewSeq++
	token := fmt.Sprintf("%032x", s.previewSeq)
	expires := s.instant().Add(previewLifetime).UTC().Truncate(time.Second)
	stored.PreviewHash = contentHash(token)
	stored.PreviewExpires = expires
	link := s.previewURL(stored, token)
	broken := s.brokenExpiry
	s.mu.Unlock()

	reported := expires.Format(time.RFC3339)
	if broken {
		reported = "soon"
	}
	s.respond(w, http.StatusOK, map[string]any{"url": link, "expiresAt": reported})
}

func (s *Server) previewURL(stored *Item, token string) string {
	parsed, err := url.Parse(s.permalink(stored))
	if err != nil {
		return s.http.URL
	}
	query := parsed.Query()
	query.Set("preview", "true")
	query.Set("postulator_preview", token)
	parsed.RawQuery = query.Encode()
	return parsed.String()
}
