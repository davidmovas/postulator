package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/pages"
)

const pagesMapToEntityName = "pages_map_to_entity"

func pagesMapToEntity(deps Deps) Tool {
	return NewTool(Def{
		Name:        pagesMapToEntityName,
		Description: "Map a page to the entity it is about.",
		Risk:        RiskWrite,
	}, func(ctx context.Context, _ Binding, in pages.MapToEntityRequest) (pages.MapToEntityResponse, error) {
		return deps.Pages.MapToEntity(ctx, in)
	})
}
