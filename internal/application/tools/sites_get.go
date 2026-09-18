package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/sites"
)

const sitesGetName = "sites_get"

func sitesGet(deps Deps) Tool {
	return NewTool(Def{
		Name:        sitesGetName,
		Description: "Read one site by id, with its plugin state and defaults.",
		Risk:        RiskRead,
	}, func(ctx context.Context, _ Binding, in sites.GetRequest) (sites.GetResponse, error) {
		return deps.Sites.Get(ctx, in)
	})
}
