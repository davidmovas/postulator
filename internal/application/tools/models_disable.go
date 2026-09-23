package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/models"
)

const modelsDisableName = "models_disable"

func modelsDisable(deps Deps) Tool {
	return NewTool(Def{
		Name:        modelsDisableName,
		Description: "Disable a model so no role may resolve to it.",
		Risk:        RiskWrite,
	}, func(ctx context.Context, _ Binding, in models.DisableModelRequest) (models.DisableModelResponse, error) {
		return deps.Models.DisableModel(ctx, in)
	})
}
