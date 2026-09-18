package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/models"
)

const modelsUpsertName = "models_upsert"

func modelsUpsert(deps Deps) Tool {
	return NewTool(Def{
		Name:        modelsUpsertName,
		Description: "Add or correct a catalog entry for a model.",
		Risk:        RiskWrite,
	}, func(ctx context.Context, _ Binding, in models.UpsertModelRequest) (models.UpsertModelResponse, error) {
		return deps.Models.UpsertModel(ctx, in)
	})
}
