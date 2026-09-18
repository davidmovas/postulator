package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/runs"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

const runsListName = "runs_list"

func runsList(deps Deps) Tool {
	return newSiteTool(Def{
		Name:        runsListName,
		Description: "List the runs of the site.",
		Risk:        RiskRead,
	}, func(ctx context.Context, b Binding, in runs.ListRequest) (paging.List[runs.Run], error) {
		in.SiteID = b.SiteID
		return deps.Runs.List(ctx, in)
	})
}
