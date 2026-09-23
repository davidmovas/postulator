package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/runs"
)

const runsPauseName = "runs_pause"

func runsPause(deps Deps) Tool {
	return NewTool(Def{
		Name:        runsPauseName,
		Description: "Pause a run.",
		Risk:        RiskWrite,
	}, func(ctx context.Context, _ Binding, in runs.PauseRequest) (runs.PauseResponse, error) {
		return deps.Runs.Pause(ctx, in)
	})
}
