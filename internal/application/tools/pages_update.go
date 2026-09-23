package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/pages"
)

const pagesUpdateName = "pages_update"

func pagesUpdate(deps Deps) Tool {
	return NewTool(Def{
		Name:        pagesUpdateName,
		Description: "Change the path, titles, meta or template of a page.",
		Risk:        RiskWrite,
	}, func(ctx context.Context, _ Binding, in pages.UpdateRequest) (pages.UpdateResponse, error) {
		return deps.Pages.Update(ctx, in)
	})
}
