package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/models"
)

const modelsSetProfileName = "models_set_profile"

func modelsSetProfile(deps Deps) Tool {
	return NewTool(Def{
		Name:        modelsSetProfileName,
		Description: "Bind a model role such as writer or judge to a model.",
		Risk:        RiskWrite,
	}, func(ctx context.Context, _ Binding, in models.SetProfileRequest) (models.SetProfileResponse, error) {
		return deps.Models.SetProfile(ctx, in)
	})
}
