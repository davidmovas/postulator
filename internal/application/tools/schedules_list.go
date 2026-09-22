package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/schedules"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

const schedulesListName = "schedules_list"

func schedulesList(deps Deps) Tool {
	return sortedBy(newSiteTool(Def{
		Name:        schedulesListName,
		Description: "List the schedules of the site.",
		Risk:        RiskRead,
	}, func(ctx context.Context, b Binding, in schedules.ListRequest) (paging.List[schedules.Schedule], error) {
		in.SiteID = b.SiteID
		return deps.Schedules.List(ctx, in)
	}), sortByCreatedAt)
}
