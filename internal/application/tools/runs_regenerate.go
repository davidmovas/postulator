package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/runs"
)

const runsRegenerateName = "runs_regenerate"

func runsRegenerate(deps Deps) Tool {
	return NewTool(Def{
		Name:        runsRegenerateName,
		Description: "Write stopped run items again from their first step, in the same run.",
		Risk:        RiskWrite,
	}, func(ctx context.Context, _ Binding, in runs.RegenerateRequest) (runs.RegenerateResponse, error) {
		return deps.Runs.Regenerate(ctx, in)
	})
}
