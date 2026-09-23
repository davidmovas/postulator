package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/schedules"
)

const schedulesRunNowName = "schedules_run_now"

func schedulesRunNow(deps Deps) Tool {
	return NewTool(Def{
		Name:        schedulesRunNowName,
		Description: "Start the run of a schedule at once, without waiting for its next time.",
		Risk:        RiskWrite,
	}, func(ctx context.Context, _ Binding, in schedules.RunNowRequest) (schedules.RunNowResponse, error) {
		return deps.Schedules.RunNow(ctx, in)
	})
}
