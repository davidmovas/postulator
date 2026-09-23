package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/reports"
)

const reportsRunName = "reports_run"

func reportsRun(deps Deps) Tool {
	return NewTool(Def{
		Name:        reportsRunName,
		Description: "Read the per item report of a run.",
		Risk:        RiskRead,
	}, func(ctx context.Context, _ Binding, in reports.RunReportRequest) (reports.RunReportResponse, error) {
		return deps.Reports.RunReport(ctx, in)
	})
}
