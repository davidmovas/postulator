package wp

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func (c *Client) ListItems(ctx context.Context, itemType ItemType, query ListQuery) (ItemPage, error) {
	namespace, path, err := itemType.route()
	if err != nil {
		return ItemPage{}, err
	}

	resp, body, err := c.do(ctx, request{
		method:    http.MethodGet,
		namespace: namespace,
		path:      path,
		query:     query.values(itemType),
	})
	if err != nil {
		if endOfList(err) {
			return ItemPage{Page: query.pageNumber()}, nil
		}
		return ItemPage{}, err
	}

	items, err := decodeItems(itemType, body)
	if err != nil {
		return ItemPage{}, err
	}
	return newPage(resp, query.pageNumber(), items), nil
}

func (c *Client) GetItem(ctx context.Context, itemType ItemType, id int64) (Item, error) {
	namespace, path, err := itemType.route()
	if err != nil {
		return Item{}, err
	}

	query := url.Values{}
	if itemType.core() {
		query.Set("context", "edit")
	}

	_, body, err := c.do(ctx, request{
		method:    http.MethodGet,
		namespace: namespace,
		path:      resourcePath(path, id),
		query:     query,
	})
	if err != nil {
		return Item{}, err
	}
	return decodeItem(itemType, body)
}

func newPage[T any](resp *http.Response, page int, items []T) Page[T] {
	totalPages := headerInt(resp, "X-WP-TotalPages")
	return Page[T]{
		Items:      items,
		Total:      headerInt(resp, "X-WP-Total"),
		TotalPages: totalPages,
		Page:       page,
		HasMore:    page < totalPages,
	}
}

func headerInt(resp *http.Response, name string) int {
	value, err := strconv.Atoi(resp.Header.Get(name))
	if err != nil {
		return 0
	}
	return value
}

func endOfList(err error) bool {
	if !errors.IsCode(err, errors.Invalid) {
		return false
	}
	if detailString(err, "code") == "rest_post_invalid_page_number" {
		return true
	}
	return strings.Contains(strings.ToLower(detailString(err, "wpMessage")), "larger than the number of pages")
}

func (c *Client) CreateItem(ctx context.Context, itemType ItemType, in CreateItem) (Item, error) {
	if !itemType.core() {
		return Item{}, coreOnly(itemType)
	}

	namespace, path, err := itemType.route()
	if err != nil {
		return Item{}, err
	}

	body, err := encodeJSON(in.payload())
	if err != nil {
		return Item{}, err
	}

	_, raw, err := c.do(ctx, request{
		method:      http.MethodPost,
		namespace:   namespace,
		path:        path,
		body:        body,
		contentType: contentTypeJSON,
	})
	if err != nil {
		return Item{}, err
	}

	created, err := decodeItem(itemType, raw)
	if err != nil {
		return Item{}, err
	}
	return c.GetItem(ctx, itemType, created.ID)
}

func (c *Client) UpdateItem(ctx context.Context, itemType ItemType, id int64, in UpdateItem) (Item, error) {
	if !itemType.core() {
		return Item{}, coreOnly(itemType)
	}

	namespace, path, err := itemType.route()
	if err != nil {
		return Item{}, err
	}

	payload := in.payload()
	if len(payload) == 0 {
		return Item{}, errors.New(errors.Invalid, "the update carries no fields")
	}

	body, err := encodeJSON(payload)
	if err != nil {
		return Item{}, err
	}

	_, raw, err := c.do(ctx, request{
		method:      http.MethodPost,
		namespace:   namespace,
		path:        resourcePath(path, id),
		body:        body,
		contentType: contentTypeJSON,
	})
	if err != nil {
		return Item{}, err
	}
	return decodeItem(itemType, raw)
}

func (c *Client) DeleteItem(ctx context.Context, itemType ItemType, id int64, force bool) error {
	if !itemType.core() {
		return coreOnly(itemType)
	}

	namespace, path, err := itemType.route()
	if err != nil {
		return err
	}

	query := url.Values{}
	if force {
		query.Set("force", "true")
	}

	_, _, err = c.do(ctx, request{
		method:    http.MethodDelete,
		namespace: namespace,
		path:      resourcePath(path, id),
		query:     query,
	})
	return err
}

func coreOnly(itemType ItemType) error {
	return errors.New(errors.Invalid, "only pages and posts are written through the generic item methods").
		WithDetail("type", string(itemType))
}
