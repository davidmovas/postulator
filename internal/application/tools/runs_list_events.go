package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/runs"
)

const runsListEventsName = "runs_list_events"

func runsListEvents(deps Deps) Tool {
	return NewTool(Def{
		Name:        runsListEventsName,
		Description: "Read the durable event log of a run after a sequence number.",
		Risk:        RiskRead,
	}, func(ctx context.Context, _ Binding, in runs.ListEventsRequest) (runs.ListEventsResponse, error) {
		return deps.Runs.ListEvents(ctx, in)
	})
}
