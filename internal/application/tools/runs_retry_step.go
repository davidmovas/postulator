package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/runs"
)

const runsRetryStepName = "runs_retry_step"

type retryStepArgs struct {
	ItemID         string `json:"itemId" description:"Stopped item id from runs_list_items"`
	AcceptFindings bool   `json:"acceptFindings,omitempty" description:"Let a page held at validate go on as it is"`
}

func runsRetryStep(deps Deps) Tool {
	return NewTool(Def{
		Name:        runsRetryStepName,
		Description: "Retry a stopped run item's current step; acceptFindings lets a page held at validate go on as it is.",
		Risk:        RiskWrite,
	}, func(ctx context.Context, _ Binding, in retryStepArgs) (runs.RetryStepResponse, error) {
		return deps.Runs.RetryStep(ctx, runs.RetryStepRequest{ItemID: in.ItemID, AcceptFindings: in.AcceptFindings})
	})
}
