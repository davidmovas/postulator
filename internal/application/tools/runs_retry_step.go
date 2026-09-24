package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/runs"
)

const runsRetryStepName = "runs_retry_step"

func runsRetryStep(deps Deps) Tool {
	return NewTool(Def{
		Name:        runsRetryStepName,
		Description: "Retry the current step of a run item that stopped; with acceptFindings, a page held at validate goes on as it is.",
		Risk:        RiskWrite,
	}, func(ctx context.Context, _ Binding, in runs.RetryStepRequest) (runs.RetryStepResponse, error) {
		return deps.Runs.RetryStep(ctx, in)
	})
}
