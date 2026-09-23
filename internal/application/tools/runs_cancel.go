package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/runs"
)

const runsCancelName = "runs_cancel"

func runsCancel(deps Deps) Tool {
	return NewTool(Def{
		Name:        runsCancelName,
		Description: "Cancel a run and every item still in it.",
		Risk:        RiskWrite,
	}, func(ctx context.Context, _ Binding, in runs.CancelRequest) (runs.CancelResponse, error) {
		return deps.Runs.Cancel(ctx, in)
	})
}
