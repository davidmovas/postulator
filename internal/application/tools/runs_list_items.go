package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/runs"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

const runsListItemsName = "runs_list_items"

func runsListItems(deps Deps) Tool {
	return orderedOnly(NewTool(Def{
		Name:        runsListItemsName,
		Description: "List a run's items and the step each is on, in planned order.",
		Risk:        RiskRead,
	}, func(ctx context.Context, _ Binding, in runs.ListItemsRequest) (paging.List[runs.Item], error) {
		return deps.Runs.ListItems(ctx, in)
	}))
}
