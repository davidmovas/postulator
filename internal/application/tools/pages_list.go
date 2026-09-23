package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/pages"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

const pagesListName = "pages_list"

func pagesList(deps Deps) Tool {
	return sortedBy(newSiteTool(Def{
		Name:        pagesListName,
		Description: "List the pages of the site, optionally by status, entity or path prefix.",
		Risk:        RiskRead,
	}, func(ctx context.Context, b Binding, in pages.ListRequest) (paging.List[pages.Page], error) {
		in.SiteID = b.SiteID
		return deps.Pages.List(ctx, in)
	}), string(pagemap.SortCreatedAt), string(pagemap.SortPath))
}
