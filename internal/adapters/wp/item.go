package wp

import (
	"bytes"
	"encoding/json"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	wpTimeLayout   = "2006-01-02T15:04:05"
	defaultPerPage = 50
	maxPerPage     = 100
)

type ItemType string

const (
	TypePage            ItemType = "page"
	TypePost            ItemType = "post"
	TypeProduct         ItemType = "product"
	TypeProductCategory ItemType = "product_cat"
)

func (t ItemType) route() (namespace, path string, err error) {
	switch t {
	case TypePage:
		return coreNamespace, "/pages", nil
	case TypePost:
		return coreNamespace, "/posts", nil
	case TypeProduct:
		return wooNamespace, "/products", nil
	case TypeProductCategory:
		return wooNamespace, "/products/categories", nil
	default:
		return "", "", errors.New(errors.Invalid, "unknown WordPress content type").WithDetail("type", string(t))
	}
}

func (t ItemType) core() bool {
	return t == TypePage || t == TypePost
}

type Item struct {
	Modified      time.Time
	Meta          map[string]any
	Type          ItemType
	Title         string
	Content       string
	Excerpt       string
	Slug          string
	Status        string
	Link          string
	Template      string
	Categories    []int64
	Tags          []int64
	ID            int64
	Parent        int64
	FeaturedMedia int64
	MenuOrder     int
}

type Page[T any] struct {
	Items      []T
	Total      int
	TotalPages int
	Page       int
	HasMore    bool
}

type ItemPage = Page[Item]

type ListQuery struct {
	ModifiedAfter *time.Time
	Status        []string
	Fields        []string
	Page          int
	PerPage       int
}

func (q ListQuery) pageNumber() int {
	if q.Page < 1 {
		return 1
	}
	return q.Page
}

func (q ListQuery) perPageSize() int {
	switch {
	case q.PerPage < 1:
		return defaultPerPage
	case q.PerPage > maxPerPage:
		return maxPerPage
	default:
		return q.PerPage
	}
}

func (q ListQuery) values(itemType ItemType) url.Values {
	query := url.Values{}
	if itemType.core() {
		query.Set("context", "edit")
	}
	query.Set("page", strconv.Itoa(q.pageNumber()))
	query.Set("per_page", strconv.Itoa(q.perPageSize()))
	query.Set("orderby", "id")
	query.Set("order", "asc")

	if q.ModifiedAfter != nil && itemType != TypeProductCategory {
		query.Set("modified_after", q.ModifiedAfter.UTC().Format(wpTimeLayout))
	}
	if len(q.Status) > 0 {
		if itemType.core() {
			query.Set("status", strings.Join(q.Status, ","))
		} else {
			query.Set("status", q.Status[0])
		}
	}
	if len(q.Fields) > 0 {
		query.Set("_fields", strings.Join(withID(q.Fields), ","))
	}
	return query
}

func withID(fields []string) []string {
	if slices.Contains(fields, "id") {
		return fields
	}
	return append([]string{"id"}, fields...)
}

func resourcePath(path string, id int64) string {
	return path + "/" + strconv.FormatInt(id, 10)
}

type renderedText struct {
	Raw      string `json:"raw"`
	Rendered string `json:"rendered"`
}

func (r renderedText) value() string {
	if r.Raw != "" {
		return r.Raw
	}
	return r.Rendered
}

type metaBag map[string]any

func (m *metaBag) UnmarshalJSON(data []byte) error {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || trimmed[0] == '[' || bytes.Equal(trimmed, []byte("null")) {
		*m = nil
		return nil
	}

	var values map[string]any
	if err := json.Unmarshal(trimmed, &values); err != nil {
		return err
	}
	*m = values
	return nil
}

type itemPayload struct {
	Title         renderedText `json:"title"`
	Content       renderedText `json:"content"`
	Excerpt       renderedText `json:"excerpt"`
	Meta          metaBag      `json:"meta"`
	Type          string       `json:"type"`
	Slug          string       `json:"slug"`
	Status        string       `json:"status"`
	Link          string       `json:"link"`
	Template      string       `json:"template"`
	ModifiedGMT   string       `json:"modified_gmt"`
	Categories    []int64      `json:"categories"`
	Tags          []int64      `json:"tags"`
	ID            int64        `json:"id"`
	Parent        int64        `json:"parent"`
	FeaturedMedia int64        `json:"featured_media"`
	MenuOrder     int          `json:"menu_order"`
}

func (p itemPayload) item(fallback ItemType) Item {
	itemType := ItemType(p.Type)
	if itemType == "" {
		itemType = fallback
	}

	return Item{
		ID:            p.ID,
		Type:          itemType,
		Title:         p.Title.value(),
		Content:       p.Content.value(),
		Excerpt:       p.Excerpt.value(),
		Slug:          p.Slug,
		Status:        p.Status,
		Link:          p.Link,
		Template:      p.Template,
		Categories:    p.Categories,
		Tags:          p.Tags,
		Parent:        p.Parent,
		FeaturedMedia: p.FeaturedMedia,
		MenuOrder:     p.MenuOrder,
		Meta:          p.Meta,
		Modified:      parseWPTime(p.ModifiedGMT),
	}
}

type productCategoryRef struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
	ID   int64  `json:"id"`
}

type productPayload struct {
	Name             string               `json:"name"`
	Slug             string               `json:"slug"`
	Permalink        string               `json:"permalink"`
	Status           string               `json:"status"`
	Description      string               `json:"description"`
	ShortDescription string               `json:"short_description"`
	DateModifiedGMT  string               `json:"date_modified_gmt"`
	Categories       []productCategoryRef `json:"categories"`
	ID               int64                `json:"id"`
	MenuOrder        int                  `json:"menu_order"`
}

func (p productPayload) item() Item {
	categories := make([]int64, 0, len(p.Categories))
	for _, ref := range p.Categories {
		categories = append(categories, ref.ID)
	}

	return Item{
		ID:         p.ID,
		Type:       TypeProduct,
		Title:      p.Name,
		Content:    p.Description,
		Excerpt:    p.ShortDescription,
		Slug:       p.Slug,
		Status:     p.Status,
		Link:       p.Permalink,
		Categories: categories,
		MenuOrder:  p.MenuOrder,
		Modified:   parseWPTime(p.DateModifiedGMT),
	}
}

type productCategoryPayload struct {
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
	ID          int64  `json:"id"`
	Parent      int64  `json:"parent"`
	Count       int    `json:"count"`
}

func (p productCategoryPayload) item() Item {
	return Item{
		ID:      p.ID,
		Type:    TypeProductCategory,
		Title:   p.Name,
		Content: p.Description,
		Slug:    p.Slug,
		Parent:  p.Parent,
	}
}

func parseWPTime(value string) time.Time {
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

func decodeItems(itemType ItemType, body []byte) ([]Item, error) {
	switch itemType {
	case TypePage, TypePost:
		var payload []itemPayload
		if err := decodeJSON(body, &payload); err != nil {
			return nil, err
		}
		items := make([]Item, 0, len(payload))
		for index := range payload {
			items = append(items, payload[index].item(itemType))
		}
		return items, nil
	case TypeProduct:
		var payload []productPayload
		if err := decodeJSON(body, &payload); err != nil {
			return nil, err
		}
		items := make([]Item, 0, len(payload))
		for index := range payload {
			items = append(items, payload[index].item())
		}
		return items, nil
	case TypeProductCategory:
		var payload []productCategoryPayload
		if err := decodeJSON(body, &payload); err != nil {
			return nil, err
		}
		items := make([]Item, 0, len(payload))
		for index := range payload {
			items = append(items, payload[index].item())
		}
		return items, nil
	default:
		return nil, errors.New(errors.Invalid, "unknown WordPress content type").WithDetail("type", string(itemType))
	}
}

func decodeItem(itemType ItemType, body []byte) (Item, error) {
	switch itemType {
	case TypePage, TypePost:
		var payload itemPayload
		if err := decodeJSON(body, &payload); err != nil {
			return Item{}, err
		}
		return payload.item(itemType), nil
	case TypeProduct:
		var payload productPayload
		if err := decodeJSON(body, &payload); err != nil {
			return Item{}, err
		}
		return payload.item(), nil
	case TypeProductCategory:
		var payload productCategoryPayload
		if err := decodeJSON(body, &payload); err != nil {
			return Item{}, err
		}
		return payload.item(), nil
	default:
		return Item{}, errors.New(errors.Invalid, "unknown WordPress content type").WithDetail("type", string(itemType))
	}
}
