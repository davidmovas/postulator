package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/sites"
)

const sitesDeleteName = "sites_delete"

func sitesDelete(deps Deps) Tool {
	return NewTool(Def{
		Name:        sitesDeleteName,
		Description: "Delete a site and everything Postulator holds about it.",
		Risk:        RiskDangerous,
	}, func(ctx context.Context, _ Binding, in sites.DeleteRequest) (sites.DeleteResponse, error) {
		return deps.Sites.Delete(ctx, in)
	})
}
