package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/schedules"
)

const schedulesDisableName = "schedules_disable"

func schedulesDisable(deps Deps) Tool {
	return NewTool(Def{
		Name:        schedulesDisableName,
		Description: "Disable a schedule without deleting it.",
		Risk:        RiskWrite,
	}, func(ctx context.Context, _ Binding, in schedules.DisableRequest) (schedules.DisableResponse, error) {
		return deps.Schedules.Disable(ctx, in)
	})
}
