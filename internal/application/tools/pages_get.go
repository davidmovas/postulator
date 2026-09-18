package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/pages"
)

const pagesGetName = "pages_get"

func pagesGet(deps Deps) Tool {
	return NewTool(Def{
		Name:        pagesGetName,
		Description: "Read one page with the links it carries.",
		Risk:        RiskRead,
	}, func(ctx context.Context, _ Binding, in pages.GetRequest) (pages.GetResponse, error) {
		return deps.Pages.Get(ctx, in)
	})
}
