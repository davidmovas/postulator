package wp

import (
	"context"
	"net/http"

	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const CapabilityRendered = "rendered"

type Rendered struct {
	Type        ItemType
	Content     string
	ContentHash string
	ID          int64
}

type renderedPayload struct {
	Type        string `json:"type"`
	Content     string `json:"content"`
	ContentHash string `json:"contentHash"`
	ID          int64  `json:"id"`
}

func (c *Client) GetRendered(ctx context.Context, itemType ItemType, id int64) (Rendered, error) {
	if err := postRoute(itemType, id); err != nil {
		return Rendered{}, err
	}
	if err := c.requireCapability(ctx, CapabilityRendered); err != nil {
		return Rendered{}, err
	}

	_, body, err := c.do(ctx, request{method: http.MethodGet, namespace: pluginNamespace, path: resourcePath("/content", id) + "/rendered"})
	if err != nil {
		return Rendered{}, err
	}

	var payload renderedPayload
	if err := decodeJSON(body, &payload); err != nil {
		return Rendered{}, err
	}
	if ItemType(payload.Type) != itemType {
		return Rendered{}, errors.New(errors.NotFound, "the site holds a "+payload.Type+" under that number, not a "+string(itemType)).
			WithDetail("id", id).
			WithDetail("type", string(itemType)).
			WithDetail("found", payload.Type)
	}
	if payload.ContentHash != ContentHash(payload.Content) {
		return Rendered{}, errors.New(errors.External, "the site reported a hash that does not match the filtered content it returned").
			WithDetail("id", payload.ID)
	}
	return Rendered{ID: payload.ID, Type: ItemType(payload.Type), Content: payload.Content, ContentHash: payload.ContentHash}, nil
}
