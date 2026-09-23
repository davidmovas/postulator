package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/pages"
)

const pagesCreateName = "pages_create"

func pagesCreate(deps Deps) Tool {
	return newSiteTool(Def{
		Name:        pagesCreateName,
		Description: "Plan a page at a path in the site page map.",
		Risk:        RiskWrite,
	}, func(ctx context.Context, b Binding, in pages.CreateRequest) (pages.CreateResponse, error) {
		in.SiteID = b.SiteID
		return deps.Pages.Create(ctx, in)
	})
}
