package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/schedules"
)

const schedulesCreateName = "schedules_create"

func schedulesCreate(deps Deps) Tool {
	return newSiteTool(Def{
		Name:        schedulesCreateName,
		Description: "Create a schedule that starts a run on a cron expression or on an interval.",
		Risk:        RiskWrite,
	}, func(ctx context.Context, b Binding, in schedules.CreateRequest) (schedules.CreateResponse, error) {
		in.SiteID = b.SiteID
		return deps.Schedules.Create(ctx, in)
	})
}
