package wp

import (
	"context"
	"maps"
	"net/http"
	"slices"
	"strconv"

	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

type MetaWrite struct {
	Value    *string
	Expected *string
}

type StoredMeta struct {
	Value *string
	Hash  *string
}

type storedMetaPayload struct {
	Value *string `json:"value"`
	Hash  *string `json:"hash"`
}

func (c *Client) PutMeta(ctx context.Context, itemType ItemType, id int64, writes map[string]MetaWrite) (map[string]StoredMeta, error) {
	keys := slices.Sorted(maps.Keys(writes))
	switch {
	case len(keys) == 0:
		return nil, errors.New(errors.Invalid, "a meta write names at least one key")
	case len(keys) > MaxMetaKeys:
		return nil, errors.New(errors.Invalid, "a meta write names at most "+strconv.Itoa(MaxMetaKeys)+" keys").
			WithDetail("keys", len(keys))
	}
	for _, key := range keys {
		if !field.MetaKeyAllowed(key) {
			return nil, errors.New(errors.Invalid, "the meta key is protected, the store's own, or not a plain key").
				WithDetail("reason", field.ReasonDenied).WithDetail("key", key)
		}
	}
	if err := c.contentPost(ctx, itemType, id); err != nil {
		return nil, err
	}
	if err := c.requireCapability(ctx, CapabilityGuardedWrite); err != nil {
		return nil, err
	}
	if err := c.requireCapability(ctx, CapabilityMetaWrite); err != nil {
		return nil, err
	}

	values := make(map[string]*string, len(keys))
	expected := make(map[string]*string, len(keys))
	for _, key := range keys {
		values[key], expected[key] = writes[key].Value, writes[key].Expected
	}
	body, err := encodeJSON(map[string]any{"meta": values, "expected": expected})
	if err != nil {
		return nil, err
	}

	_, raw, err := c.do(ctx, request{
		method:      http.MethodPut,
		namespace:   pluginNamespace,
		path:        resourcePath("/meta", id),
		body:        body,
		contentType: contentTypeJSON,
	})
	if err != nil {
		return nil, err
	}

	var payload struct {
		Meta map[string]storedMetaPayload `json:"meta"`
	}
	if err := decodeJSON(raw, &payload); err != nil {
		return nil, err
	}
	stored := make(map[string]StoredMeta, len(keys))
	for _, key := range keys {
		held, answered := payload.Meta[key]
		if !answered || (held.Value == nil) != (held.Hash == nil) || (held.Value != nil && *held.Hash != ContentHash(*held.Value)) {
			return nil, errors.New(errors.External, "the site answered a meta value whose hash it did not report").
				WithDetail("id", id).WithDetail("key", key)
		}
		stored[key] = StoredMeta(held)
	}
	return stored, nil
}
