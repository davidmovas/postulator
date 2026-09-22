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
		Description: "List the items of a run and the step each one is on, in the order they were planned.",
		Risk:        RiskRead,
	}, func(ctx context.Context, _ Binding, in runs.ListItemsRequest) (paging.List[runs.Item], error) {
		return deps.Runs.ListItems(ctx, in)
	}))
}
