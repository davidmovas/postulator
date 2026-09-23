package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/runs"
)

const runsRevertName = "runs_revert"

func runsRevert(deps Deps) Tool {
	return NewTool(Def{
		Name: runsRevertName,
		Description: "Undo what a finished run wrote to the site: a page it created is trashed, a page " +
			"it updated gets its previous body back, and every relinked neighbor is restored. " +
			"Answers the id of the run doing it.",
		Risk: RiskDangerous,
	}, func(ctx context.Context, _ Binding, in runs.RevertRequest) (runs.RevertResponse, error) {
		return deps.Runs.Revert(ctx, in)
	})
}
