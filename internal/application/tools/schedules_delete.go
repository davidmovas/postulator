package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/schedules"
)

const schedulesDeleteName = "schedules_delete"

func schedulesDelete(deps Deps) Tool {
	return NewTool(Def{
		Name:        schedulesDeleteName,
		Description: "Delete a schedule.",
		Risk:        RiskDangerous,
	}, func(ctx context.Context, _ Binding, in schedules.DeleteRequest) (schedules.DeleteResponse, error) {
		return deps.Schedules.Delete(ctx, in)
	})
}
