package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/pages"
)

const pagesUnmapName = "pages_unmap"

func pagesUnmap(deps Deps) Tool {
	return NewTool(Def{
		Name:        pagesUnmapName,
		Description: "Detach a page from its entity.",
		Risk:        RiskWrite,
	}, func(ctx context.Context, _ Binding, in pages.UnmapRequest) (pages.UnmapResponse, error) {
		return deps.Pages.Unmap(ctx, in)
	})
}
