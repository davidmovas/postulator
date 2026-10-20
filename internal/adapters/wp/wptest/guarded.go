package wptest

import (
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

var (
	plainMetaKey    = regexp.MustCompile(`^[A-Za-z0-9_-]{1,255}$`)
	storeMetaKeys   = []string{"_price", "_regular_price", "_sale_price", "_sku", "_stock_status", "_stock", "_manage_stock", "_product_attributes", "total_sales"}
	deniedMetaHeads = []string{"_", "attribute_", "rank_math_"}
)

func metaKeyRefused(key string) bool {
	if !plainMetaKey.MatchString(key) {
		return true
	}
	lower := strings.ToLower(key)
	if slices.Contains(storeMetaKeys, lower) {
		return true
	}
	for _, head := range deniedMetaHeads {
		if strings.HasPrefix(lower, head) {
			return true
		}
	}
	return false
}

func (s *Server) EditMeta(id int64, key, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	stored, found := s.items[id]
	if !found {
		return
	}
	if stored.Meta == nil {
		stored.Meta = make(map[string]string)
	}
	stored.Meta[key] = value
	stored.Modified = s.tick()
}

func (s *Server) saveStable(content string) bool {
	return !s.filteredHTML || kses(content) == content
}

func (s *Server) saved(content string) string {
	if s.filteredHTML {
		return kses(content)
	}
	return content
}

func metaState(meta map[string]string, key string) *string {
	value, held := meta[key]
	if !held {
		return nil
	}
	hash := contentHash(value)
	return &hash
}

func sameState(current, expected *string) bool {
	if current == nil || expected == nil {
		return current == nil && expected == nil
	}
	return *current == *expected
}

func (s *Server) handleMetaPut(w http.ResponseWriter, r *http.Request) {
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
	values, valuesOK := body["meta"].(map[string]any)
	expected, expectedOK := body["expected"].(map[string]any)
	if !valuesOK || !expectedOK || len(values) == 0 {
		s.fail(w, http.StatusBadRequest, "invalid_body", "meta and expected are required objects.")
		return
	}
	if len(values) > maxMetaKeys {
		s.fail(w, http.StatusBadRequest, "too_many_keys", "meta names more than "+strconv.Itoa(maxMetaKeys)+" meta keys")
		return
	}

	keys := make([]string, 0, len(values))
	var refused []string
	for key, value := range values {
		_, isText := value.(string)
		if (value != nil && !isText) || metaKeyRefused(key) {
			refused = append(refused, key)
		}
		if _, guarded := expected[key]; !guarded {
			s.fail(w, http.StatusBadRequest, "missing_expected", "every key written names the hash it was read with.")
			return
		}
		keys = append(keys, key)
	}
	slices.Sort(keys)
	if len(refused) > 0 {
		slices.Sort(refused)
		s.respond(w, http.StatusBadRequest, map[string]any{
			"code": "meta_key_denied", "message": "a protected meta key is never written: " + strings.Join(refused, ", "),
			"data": map[string]any{"status": http.StatusBadRequest, "keys": refused},
		})
		return
	}

	s.mu.Lock()
	stored, found := s.editablePost(id)
	if !found {
		s.mu.Unlock()
		s.fail(w, http.StatusNotFound, "not_found", "No content with that id exists.")
		return
	}
	moved := make(map[string]any)
	for _, key := range keys {
		var hash *string
		want, named := expected[key].(string)
		switch {
		case named:
			hash = &want
		case expected[key] != nil:
			moved[key] = metaState(stored.Meta, key)
			continue
		}
		if current := metaState(stored.Meta, key); !sameState(current, hash) {
			moved[key] = current
		}
	}
	if len(moved) > 0 {
		s.mu.Unlock()
		s.respond(w, http.StatusConflict, map[string]any{
			"code": "hash_mismatch", "message": "a meta value changed since it was read.", "moved": moved,
		})
		return
	}
	if stored.Meta == nil {
		stored.Meta = make(map[string]string)
	}
	answer := make(map[string]any, len(keys))
	for _, key := range keys {
		value, isText := values[key].(string)
		if !isText {
			delete(stored.Meta, key)
			answer[key] = map[string]any{"value": nil, "hash": nil}
			continue
		}
		stored.Meta[key] = value
		answer[key] = map[string]any{"value": value, "hash": contentHash(value)}
	}
	stored.Modified = s.tick()
	s.mu.Unlock()

	s.respond(w, http.StatusOK, map[string]any{"id": id, "meta": answer})
}

func (s *Server) movedSEO(stored *Item, keys map[string]string, body map[string]any) map[string]any {
	expected, guarded := body["expected"].(map[string]any)
	if !guarded {
		return nil
	}
	moved := make(map[string]any)
	for _, field := range seoFieldOrder {
		want, named := expected[field].(string)
		if _, written := body[field]; !named || !written {
			continue
		}
		if current := contentHash(stored.Meta[keys[field]]); current != want {
			moved[field] = current
		}
	}
	return moved
}
