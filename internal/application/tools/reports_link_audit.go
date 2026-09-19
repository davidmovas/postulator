package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/reports"
)

const reportsLinkAuditName = "reports_link_audit"

func reportsLinkAudit(deps Deps) Tool {
	return newSiteTool(Def{
		Name:        reportsLinkAuditName,
		Description: "Audit every page of the site against the links the entity graph asks it to carry.",
		Risk:        RiskRead,
	}, func(ctx context.Context, b Binding, in reports.LinkAuditRequest) (reports.LinkAuditResponse, error) {
		in.SiteID = b.SiteID
		return deps.Reports.LinkAudit(ctx, in)
	})
}
