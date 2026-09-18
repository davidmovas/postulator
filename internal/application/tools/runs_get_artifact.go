package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/runs"
)

const runsGetArtifactName = "runs_get_artifact"

func runsGetArtifact(deps Deps) Tool {
	return NewTool(Def{
		Name:        runsGetArtifactName,
		Description: "Read one artifact a run item produced, such as its validation report.",
		Risk:        RiskRead,
	}, func(ctx context.Context, _ Binding, in runs.GetArtifactRequest) (runs.GetArtifactResponse, error) {
		return deps.Runs.GetArtifact(ctx, in)
	})
}
