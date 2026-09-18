package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/schedules"
)

const schedulesUpdateName = "schedules_update"

func schedulesUpdate(deps Deps) Tool {
	return NewTool(Def{
		Name:        schedulesUpdateName,
		Description: "Change the timing, the targets or the recipe of a schedule.",
		Risk:        RiskWrite,
	}, func(ctx context.Context, _ Binding, in schedules.UpdateRequest) (schedules.UpdateResponse, error) {
		return deps.Schedules.Update(ctx, in)
	})
}
