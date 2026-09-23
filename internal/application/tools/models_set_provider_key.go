package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/models"
)

const modelsSetProviderKeyName = "models_set_provider_key"

func modelsSetProviderKey(deps Deps) Tool {
	return NewTool(Def{
		Name:        modelsSetProviderKeyName,
		Description: "Store the api key of a model provider.",
		Risk:        RiskDangerous,
	}, func(ctx context.Context, _ Binding, in models.SetProviderKeyRequest) (models.SetProviderKeyResponse, error) {
		return deps.Models.SetProviderKey(ctx, in)
	})
}
