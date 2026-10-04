package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/reports"
)

const reportsLinkAuditPageName = "reports_link_audit_page"

func reportsLinkAuditPage(deps Deps) Tool {
	return NewTool(Def{
		Name:        reportsLinkAuditPageName,
		Description: "Read the links a page owes the graph, which it carries, and links the graph never asked for.",
		Risk:        RiskRead,
	}, func(ctx context.Context, _ Binding, in reports.LinkAuditPageRequest) (reports.LinkAuditPageResponse, error) {
		return deps.Reports.LinkAuditPage(ctx, in)
	})
}
