package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/models"
)

const modelsGetProfilesName = "models_get_profiles"

func modelsGetProfiles(deps Deps) Tool {
	return newSiteTool(Def{
		Name:        modelsGetProfilesName,
		Description: "Read the model bound to each role on this site.",
		Risk:        RiskRead,
	}, func(ctx context.Context, b Binding, in models.GetProfilesRequest) (models.GetProfilesResponse, error) {
		in.SiteID = b.SiteID
		return deps.Models.GetProfiles(ctx, in)
	})
}
