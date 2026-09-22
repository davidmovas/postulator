package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/pages"
)

const pagesDeleteName = "pages_delete"

func pagesDelete(deps Deps) Tool {
	return NewTool(Def{
		Name:        pagesDeleteName,
		Description: "Delete a page from the page map, and optionally move it to the WordPress trash.",
		Risk:        RiskDangerous,
	}, func(ctx context.Context, _ Binding, in pages.DeleteRequest) (pages.DeleteResponse, error) {
		return deps.Pages.Delete(ctx, in)
	})
}
