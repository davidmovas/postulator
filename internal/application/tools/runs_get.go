package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/runs"
)

const runsGetName = "runs_get"

func runsGet(deps Deps) Tool {
	return NewTool(Def{
		Name:        runsGetName,
		Description: "Read a run with its status, budget and statistics.",
		Risk:        RiskRead,
	}, func(ctx context.Context, _ Binding, in runs.GetRequest) (runs.GetResponse, error) {
		return deps.Runs.Get(ctx, in)
	})
}
