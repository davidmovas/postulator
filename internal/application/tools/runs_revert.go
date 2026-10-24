package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/runs"
)

const runsRevertName = "runs_revert"

type revertArgs struct {
	RunID string `json:"runId" description:"Finished run id"`
}

func runsRevert(deps Deps) Tool {
	return NewTool(Def{
		Name: runsRevertName,
		Description: "Undo what a finished run wrote: created pages are trashed, updated pages and relinked " +
			"neighbors get their previous body back. Answers the reverting run's id.",
		Risk: RiskDangerous,
	}, func(ctx context.Context, _ Binding, in revertArgs) (runs.RevertResponse, error) {
		return deps.Runs.Revert(ctx, runs.RevertRequest{RunID: in.RunID})
	})
}
