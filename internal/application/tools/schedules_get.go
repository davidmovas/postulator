package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/schedules"
)

const schedulesGetName = "schedules_get"

func schedulesGet(deps Deps) Tool {
	return NewTool(Def{
		Name:        schedulesGetName,
		Description: "Read one schedule with its next run time.",
		Risk:        RiskRead,
	}, func(ctx context.Context, _ Binding, in schedules.GetRequest) (schedules.GetResponse, error) {
		return deps.Schedules.Get(ctx, in)
	})
}
