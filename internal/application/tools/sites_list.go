package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/sites"
	"github.com/davidmovas/postulator/internal/domain/site"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

const sitesListName = "sites_list"

func sitesList(deps Deps) Tool {
	return sortedBy(NewTool(Def{
		Name:        sitesListName,
		Description: "List the WordPress sites Postulator manages.",
		Risk:        RiskRead,
	}, func(ctx context.Context, _ Binding, in sites.ListRequest) (paging.List[sites.Site], error) {
		return deps.Sites.List(ctx, in)
	}), string(site.SortCreatedAt), string(site.SortName))
}
