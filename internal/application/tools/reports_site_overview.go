package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/reports"
)

const reportsSiteOverviewName = "reports_site_overview"

func reportsSiteOverview(deps Deps) Tool {
	return newSiteTool(Def{
		Name:        reportsSiteOverviewName,
		Description: "Read the entity, page, edge and depth totals of the site.",
		Risk:        RiskRead,
	}, func(ctx context.Context, b Binding, in reports.SiteOverviewRequest) (reports.SiteOverviewResponse, error) {
		in.SiteID = b.SiteID
		return deps.Reports.SiteOverview(ctx, in)
	})
}
