package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/schedules"
)

const schedulesEnableName = "schedules_enable"

func schedulesEnable(deps Deps) Tool {
	return NewTool(Def{
		Name:        schedulesEnableName,
		Description: "Enable a schedule and arm its next run.",
		Risk:        RiskWrite,
	}, func(ctx context.Context, _ Binding, in schedules.EnableRequest) (schedules.EnableResponse, error) {
		return deps.Schedules.Enable(ctx, in)
	})
}
