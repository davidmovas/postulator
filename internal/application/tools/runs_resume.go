package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/runs"
)

const runsResumeName = "runs_resume"

func runsResume(deps Deps) Tool {
	return NewTool(Def{
		Name:        runsResumeName,
		Description: "Resume a paused run.",
		Risk:        RiskWrite,
	}, func(ctx context.Context, _ Binding, in runs.ResumeRequest) (runs.ResumeResponse, error) {
		return deps.Runs.Resume(ctx, in)
	})
}
