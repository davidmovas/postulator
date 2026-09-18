package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/models"
)

const modelsGetProfilesName = "models_get_profiles"

func modelsGetProfiles(deps Deps) Tool {
	return NewTool(Def{
		Name:        modelsGetProfilesName,
		Description: "Read the model bound to each role.",
		Risk:        RiskRead,
	}, func(ctx context.Context, _ Binding, in models.GetProfilesRequest) (models.GetProfilesResponse, error) {
		return deps.Models.GetProfiles(ctx, in)
	})
}
