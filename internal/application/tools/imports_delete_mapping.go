package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/imports"
)

const importsDeleteMappingName = "imports_delete_mapping"

func importsDeleteMapping(deps Deps) Tool {
	return NewTool(Def{
		Name:        importsDeleteMappingName,
		Description: "Delete a saved column mapping.",
		Risk:        RiskDangerous,
	}, func(ctx context.Context, _ Binding, in imports.DeleteMappingRequest) (imports.DeleteMappingResponse, error) {
		return deps.Imports.DeleteMapping(ctx, in)
	})
}
