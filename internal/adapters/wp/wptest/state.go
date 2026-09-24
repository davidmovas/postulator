package wptest

import (
	"crypto/sha256"
	"encoding/hex"
	"slices"
	"strconv"
	"strings"
	"time"
)

const (
	TypePage            = "page"
	TypePost            = "post"
	TypeProduct         = "product"
	TypeProductCategory = "product_cat"

	maxPathDepth = 10
)

var startInstant = time.Date(2026, time.September, 18, 10, 0, 0, 0, time.UTC)

type Item struct {
	Modified       time.Time
	PreviewExpires time.Time
	PreviewHash    string
	Meta           map[string]string
	Type           string
	Title          string
	H1             string
	Content        string
	Excerpt        string
	Slug           string
	Status         string
	Template       string
	Categories     []int64
	Tags           []int64
	ID             int64
	Parent         int64
	MenuOrder      int
	FeaturedMedia  int64
}

type Category struct {
	Name        string
	Slug        string
	Description string
	ID          int64
	Parent      int64
	Count       int
}

type upload struct {
	Filename string
	MimeType string
	Alt      string
	Title    string
	Bytes    []byte
	ID       int64
}

func contentHash(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func slugify(title string) string {
	var builder strings.Builder
	dashed := false
	for _, symbol := range strings.ToLower(title) {
		if (symbol >= 'a' && symbol <= 'z') || (symbol >= '0' && symbol <= '9') {
			builder.WriteRune(symbol)
			dashed = false
			continue
		}
		if !dashed && builder.Len() > 0 {
			builder.WriteByte('-')
			dashed = true
		}
	}

	trimmed := strings.Trim(builder.String(), "-")
	if trimmed == "" {
		return "item"
	}
	return trimmed
}

func hierarchical(itemType string) bool {
	return itemType == TypePage || itemType == TypeProductCategory
}

func (s *Server) instant() time.Time {
	if s.now != nil {
		return s.now().UTC().Truncate(time.Second)
	}
	return s.clock
}

func (s *Server) tick() time.Time {
	if s.now != nil {
		return s.instant()
	}
	s.clock = s.clock.Add(time.Second)
	return s.clock
}

func (s *Server) Now() time.Time {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.instant()
}

func (s *Server) uniqueSlug(base, itemType string, parent, exclude int64) string {
	candidate := base
	for suffix := 2; s.slugTaken(candidate, itemType, parent, exclude); suffix++ {
		candidate = base + "-" + strconv.Itoa(suffix)
	}
	return candidate
}

func (s *Server) slugTaken(slug, itemType string, parent, exclude int64) bool {
	for _, id := range s.order {
		stored := s.items[id]
		if stored.ID == exclude || stored.Type != itemType || stored.Slug != slug {
			continue
		}
		if !hierarchical(itemType) || stored.Parent == parent {
			return true
		}
	}
	return false
}

func (s *Server) itemPath(stored *Item) string {
	segments := []string{stored.Slug}
	parent := stored.Parent
	for depth := 0; parent != 0 && depth < maxPathDepth; depth++ {
		ancestor, ok := s.items[parent]
		if !ok {
			break
		}
		segments = append([]string{ancestor.Slug}, segments...)
		parent = ancestor.Parent
	}
	return "/" + strings.Join(segments, "/") + "/"
}

func (s *Server) add(item Item) Item {
	if !hierarchical(item.Type) {
		item.Parent = 0
	}
	if item.Type == "" {
		item.Type = TypePage
	}
	if item.Status == "" {
		item.Status = "publish"
	}
	if item.Meta == nil {
		item.Meta = make(map[string]string)
	}

	if item.ID <= 0 {
		s.nextID++
		item.ID = s.nextID
	}
	s.nextID = max(s.nextID, item.ID)

	base := item.Slug
	if base == "" {
		base = slugify(item.Title)
	}
	item.Slug = s.uniqueSlug(base, item.Type, item.Parent, item.ID)

	if item.Modified.IsZero() {
		item.Modified = s.tick()
	}

	stored := item
	s.items[item.ID] = &stored
	s.order = append(s.order, item.ID)
	return stored
}

func (s *Server) Seed(items ...Item) []Item {
	s.mu.Lock()
	defer s.mu.Unlock()

	stored := make([]Item, 0, len(items))
	for index := range items {
		items[index].ID = 0
		stored = append(stored, s.add(items[index]))
	}
	return stored
}

func (s *Server) Restore(items ...Item) []Item {
	s.mu.Lock()
	defer s.mu.Unlock()

	stored := make([]Item, 0, len(items))
	for index := range items {
		stored = append(stored, s.add(items[index]))
	}
	return stored
}

func (s *Server) SeedCategory(category Category) Category {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.nextID++
	category.ID = s.nextID
	if category.Slug == "" {
		category.Slug = slugify(category.Name)
	}

	stored := category
	s.categories[category.ID] = &stored
	s.categoryOrder = append(s.categoryOrder, category.ID)
	return stored
}

type edit struct {
	content string
	id      int64
}

func (s *Server) Rewrite(id int64, content string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.rewrite(id, content)
}

func (s *Server) rewrite(id int64, content string) bool {
	stored, ok := s.items[id]
	if !ok {
		return false
	}
	stored.Content = content
	stored.Modified = s.tick()
	return true
}

func (s *Server) Delete(id int64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.items[id]; !ok {
		return false
	}
	delete(s.items, id)
	s.order = slices.DeleteFunc(s.order, func(other int64) bool { return other == id })
	return true
}

func (s *Server) EditBeforeNextRawWrite(id int64, content string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pendingEdit = &edit{id: id, content: content}
}

func (s *Server) takePendingEdit() (edit, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.pendingEdit == nil {
		return edit{}, false
	}
	pending := *s.pendingEdit
	s.pendingEdit = nil
	s.rewrite(pending.id, pending.content)
	return pending, true
}

func (s *Server) Lookup(id int64) (Item, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	stored, ok := s.items[id]
	if !ok {
		return Item{}, false
	}
	return *stored, true
}

func (s *Server) Items() []Item {
	s.mu.Lock()
	defer s.mu.Unlock()

	items := make([]Item, 0, len(s.order))
	for _, id := range s.order {
		items = append(items, *s.items[id])
	}
	return items
}
