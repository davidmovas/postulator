package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/runs"
)

const runsRetryStepName = "runs_retry_step"

func runsRetryStep(deps Deps) Tool {
	return NewTool(Def{
		Name:        runsRetryStepName,
		Description: "Retry a stopped run item's current step; acceptFindings lets a page held at validate go on as it is.",
		Risk:        RiskWrite,
	}, func(ctx context.Context, _ Binding, in runs.RetryStepRequest) (runs.RetryStepResponse, error) {
		return deps.Runs.RetryStep(ctx, in)
	})
}
