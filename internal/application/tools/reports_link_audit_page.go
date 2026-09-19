package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/reports"
)

const reportsLinkAuditPageName = "reports_link_audit_page"

func reportsLinkAuditPage(deps Deps) Tool {
	return NewTool(Def{
		Name:        reportsLinkAuditPageName,
		Description: "Read the links one page owes to the entity graph, which of them it carries, and the links the graph never asked for.",
		Risk:        RiskRead,
	}, func(ctx context.Context, _ Binding, in reports.LinkAuditPageRequest) (reports.LinkAuditPageResponse, error) {
		return deps.Reports.LinkAuditPage(ctx, in)
	})
}
