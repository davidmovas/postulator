package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/imports"
)

const importsSaveMappingName = "imports_save_mapping"

func importsSaveMapping(deps Deps) Tool {
	return newSiteTool(Def{
		Name:        importsSaveMappingName,
		Description: "Save a column mapping under a name for the next import.",
		Risk:        RiskWrite,
	}, func(ctx context.Context, b Binding, in mappingArgs) (imports.SaveMappingResponse, error) {
		return deps.Imports.SaveMapping(ctx, imports.SaveMappingRequest{Mapping: in.mapping(b.SiteID)})
	})
}
