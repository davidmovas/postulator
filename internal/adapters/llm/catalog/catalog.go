package catalog

import (
	"context"
	_ "embed"
	"encoding/json"
	"slices"
	"strings"

	"github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

//go:embed models.json
var embedded []byte

type seed struct {
	Defaults map[llm.Role]llm.ModelRef `json:"defaults"`
	Models   []llm.ModelInfo           `json:"models"`
}

type overrideStore interface {
	List(ctx context.Context) ([]llm.ModelOverride, error)
}

type Catalog struct {
	store    overrideStore
	base     map[string]llm.ModelInfo
	defaults map[llm.Role]llm.ModelRef
}

func New(store overrideStore) (*Catalog, error) {
	var parsed seed
	if err := json.Unmarshal(embedded, &parsed); err != nil {
		return nil, errors.Wrap(err, errors.Internal, "read the embedded model catalog")
	}

	base := make(map[string]llm.ModelInfo, len(parsed.Models))
	for _, info := range parsed.Models {
		if err := info.Validate(); err != nil {
			return nil, err
		}
		base[info.Ref.String()] = info
	}

	for role, ref := range parsed.Defaults {
		if !role.Valid() {
			return nil, errors.New(errors.Internal, "the embedded model catalog names an unknown role").WithDetail("role", string(role))
		}
		if _, known := base[ref.String()]; !known {
			return nil, errors.New(errors.Internal, "the embedded model catalog defaults to a model it does not list").
				WithDetail("role", string(role)).
				WithDetail("model", ref.String())
		}
	}

	return &Catalog{store: store, base: base, defaults: parsed.Defaults}, nil
}

func (c *Catalog) resolved(ctx context.Context) (map[string]llm.ModelInfo, error) {
	overrides, err := c.store.List(ctx)
	if err != nil {
		return nil, err
	}

	merged := make(map[string]llm.ModelInfo, len(c.base)+len(overrides))
	for key, info := range c.base {
		merged[key] = info
	}
	for _, override := range overrides {
		key := override.Info.Ref.String()
		if !override.Enabled {
			delete(merged, key)
			continue
		}
		merged[key] = override.Info
	}
	return merged, nil
}

func (c *Catalog) Lookup(ctx context.Context, ref llm.ModelRef) (llm.ModelInfo, error) {
	models, err := c.resolved(ctx)
	if err != nil {
		return llm.ModelInfo{}, err
	}

	info, ok := models[ref.String()]
	if !ok {
		return llm.ModelInfo{}, errors.New(errors.NotFound, "the model catalog does not carry this model").
			WithDetail("model", ref.String())
	}
	return info, nil
}

func (c *Catalog) List(ctx context.Context) ([]llm.ModelInfo, error) {
	models, err := c.resolved(ctx)
	if err != nil {
		return nil, err
	}

	out := make([]llm.ModelInfo, 0, len(models))
	for _, info := range models {
		out = append(out, info)
	}
	slices.SortFunc(out, func(a, b llm.ModelInfo) int {
		if provider := strings.Compare(a.Ref.Provider, b.Ref.Provider); provider != 0 {
			return provider
		}
		return strings.Compare(a.Ref.Model, b.Ref.Model)
	})
	return out, nil
}

func (c *Catalog) Default(role llm.Role) (llm.ModelRef, error) {
	ref, ok := c.defaults[role]
	if !ok {
		return llm.ModelRef{}, errors.New(errors.NotFound, "the model catalog carries no default for this role").
			WithDetail("role", string(role))
	}
	return ref, nil
}
