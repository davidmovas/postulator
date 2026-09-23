package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/pages"
)

const pagesTreeName = "pages_tree"

func pagesTree(deps Deps) Tool {
	return newSiteTool(Def{
		Name:        pagesTreeName,
		Description: "Read the page map of the site as a tree.",
		Risk:        RiskRead,
	}, func(ctx context.Context, b Binding, in pages.TreeRequest) (pages.TreeResponse, error) {
		in.SiteID = b.SiteID
		return deps.Pages.Tree(ctx, in)
	})
}
