package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/imports"
)

const importsListMappingsName = "imports_list_mappings"

func importsListMappings(deps Deps) Tool {
	return newSiteTool(Def{
		Name:        importsListMappingsName,
		Description: "List the saved column mappings of the site.",
		Risk:        RiskRead,
	}, func(ctx context.Context, b Binding, in imports.ListMappingsRequest) (imports.ListMappingsResponse, error) {
		in.SiteID = b.SiteID
		return deps.Imports.ListMappings(ctx, in)
	})
}
