package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/sites"
)

const sitesCreateName = "sites_create"

func sitesCreate(deps Deps) Tool {
	return NewTool(Def{
		Name:        sitesCreateName,
		Description: "Register a WordPress site with its base url and application password.",
		Risk:        RiskWrite,
	}, func(ctx context.Context, _ Binding, in sites.CreateRequest) (sites.CreateResponse, error) {
		return deps.Sites.Create(ctx, in)
	})
}
