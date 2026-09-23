package wp

import (
	"context"
	"net/http"
	"strings"

	"github.com/davidmovas/postulator/internal/kernel/errors"
)

type Category struct {
	Name        string
	Slug        string
	Description string
	ID          int64
	Parent      int64
	Count       int
}

type CategoryPage = Page[Category]

type CreateCategory struct {
	Parent      *int64
	Name        string
	Slug        string
	Description string
}

type categoryPayload struct {
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
	ID          int64  `json:"id"`
	Parent      int64  `json:"parent"`
	Count       int    `json:"count"`
}

func (p categoryPayload) category() Category {
	return Category(p)
}

func (c *Client) ListCategories(ctx context.Context, query ListQuery) (CategoryPage, error) {
	resp, body, err := c.do(ctx, request{
		method:    http.MethodGet,
		namespace: coreNamespace,
		path:      "/categories",
		query:     query.termValues(),
	})
	if err != nil {
		if endOfList(err) {
			return CategoryPage{Page: query.pageNumber()}, nil
		}
		return CategoryPage{}, err
	}

	var payload []categoryPayload
	if err := decodeJSON(body, &payload); err != nil {
		return CategoryPage{}, err
	}

	categories := make([]Category, 0, len(payload))
	for _, entry := range payload {
		categories = append(categories, entry.category())
	}
	return newPage(resp, query.pageNumber(), categories), nil
}

func (c *Client) CreateCategory(ctx context.Context, in CreateCategory) (Category, error) {
	if strings.TrimSpace(in.Name) == "" {
		return Category{}, errors.New(errors.Invalid, "a WordPress category needs a name")
	}

	attributes := map[string]any{"name": in.Name}
	if in.Slug != "" {
		attributes["slug"] = in.Slug
	}
	if in.Description != "" {
		attributes["description"] = in.Description
	}
	if in.Parent != nil {
		attributes["parent"] = *in.Parent
	}

	body, err := encodeJSON(attributes)
	if err != nil {
		return Category{}, err
	}

	_, raw, err := c.do(ctx, request{
		method:      http.MethodPost,
		namespace:   coreNamespace,
		path:        "/categories",
		body:        body,
		contentType: contentTypeJSON,
	})
	if err != nil {
		return Category{}, err
	}

	var payload categoryPayload
	if err := decodeJSON(raw, &payload); err != nil {
		return Category{}, err
	}
	return payload.category(), nil
}
