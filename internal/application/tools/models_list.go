package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/models"
)

const modelsListName = "models_list"

func modelsList(deps Deps) Tool {
	return NewTool(Def{
		Name:        modelsListName,
		Description: "List the language models Postulator can call and their prices.",
		Risk:        RiskRead,
	}, func(ctx context.Context, _ Binding, in models.ListModelsRequest) (models.ListModelsResponse, error) {
		return deps.Models.ListModels(ctx, in)
	})
}
