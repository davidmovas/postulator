package app

import (
	"context"
	"encoding/json"

	"github.com/davidmovas/postulator/internal/kernel/settings"
)

type settingsSource interface {
	All(ctx context.Context) (map[string]json.RawMessage, error)
}

func LoadSettings(ctx context.Context, source settingsSource, registry *settings.Registry) (*settings.Values, []string, error) {
	stored, err := source.All(ctx)
	if err != nil {
		return nil, nil, err
	}

	values := registry.NewValues()
	unknown, err := registry.Apply(values, stored)
	if err != nil {
		return nil, unknown, err
	}
	return values, unknown, nil
}
