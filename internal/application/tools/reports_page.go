package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/reports"
)

const reportsPageName = "reports_page"

func reportsPage(deps Deps) Tool {
	return NewTool(Def{
		Name:        reportsPageName,
		Description: "Read the newest validation, judge, publish and relink reports of a page.",
		Risk:        RiskRead,
	}, func(ctx context.Context, _ Binding, in reports.PageReportRequest) (reports.PageReportResponse, error) {
		return deps.Reports.PageReport(ctx, in)
	})
}
