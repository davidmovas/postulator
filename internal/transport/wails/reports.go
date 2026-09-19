package wails

import (
	"context"

	"go.uber.org/zap"

	"github.com/davidmovas/postulator/internal/application/content"
	"github.com/davidmovas/postulator/internal/application/reports"
	"github.com/davidmovas/postulator/internal/kernel/middleware"
)

type ReportsUseCase interface {
	SiteOverview(ctx context.Context, req reports.SiteOverviewRequest) (reports.SiteOverviewResponse, error)
	PageReport(ctx context.Context, req reports.PageReportRequest) (reports.PageReportResponse, error)
	RunReport(ctx context.Context, req reports.RunReportRequest) (reports.RunReportResponse, error)
	LinkAudit(ctx context.Context, req reports.LinkAuditRequest) (reports.LinkAuditResponse, error)
	LinkAuditPage(ctx context.Context, req reports.LinkAuditPageRequest) (reports.LinkAuditPageResponse, error)
}

type JudgeUseCase interface {
	Judge(ctx context.Context, req content.JudgeRequest) (content.JudgeResponse, error)
}

type ReportsService struct {
	siteOverview middleware.Handler[reports.SiteOverviewRequest, reports.SiteOverviewResponse]
	pageReport   middleware.Handler[reports.PageReportRequest, reports.PageReportResponse]
	runReport    middleware.Handler[reports.RunReportRequest, reports.RunReportResponse]
	linkAudit    middleware.Handler[reports.LinkAuditRequest, reports.LinkAuditResponse]
	linkAuditPage  middleware.Handler[reports.LinkAuditPageRequest, reports.LinkAuditPageResponse]
	judgePage    middleware.Handler[content.JudgeRequest, content.JudgeResponse]
}

func NewReportsService(logger *zap.Logger, useCase Source[ReportsUseCase], judge Source[JudgeUseCase]) *ReportsService {
	return &ReportsService{
		siteOverview: Wrap(logger, "reports.siteOverview", call(useCase, ReportsUseCase.SiteOverview)),
		pageReport:   Wrap(logger, "reports.pageReport", call(useCase, ReportsUseCase.PageReport)),
		runReport:    Wrap(logger, "reports.runReport", call(useCase, ReportsUseCase.RunReport)),
		linkAudit:    Wrap(logger, "reports.linkAudit", call(useCase, ReportsUseCase.LinkAudit)),
		linkAuditPage:  Wrap(logger, "reports.linkAuditPage", call(useCase, ReportsUseCase.LinkAuditPage)),
		judgePage:    Wrap(logger, "reports.judgePage", call(judge, JudgeUseCase.Judge)),
	}
}

func (s *ReportsService) SiteOverview(c context.Context, req reports.SiteOverviewRequest) (reports.SiteOverviewResponse, error) {
	return s.siteOverview(c, req)
}

func (s *ReportsService) PageReport(c context.Context, req reports.PageReportRequest) (reports.PageReportResponse, error) {
	return s.pageReport(c, req)
}

func (s *ReportsService) RunReport(c context.Context, req reports.RunReportRequest) (reports.RunReportResponse, error) {
	return s.runReport(c, req)
}

func (s *ReportsService) LinkAudit(c context.Context, req reports.LinkAuditRequest) (reports.LinkAuditResponse, error) {
	return s.linkAudit(c, req)
}

func (s *ReportsService) LinkAuditPage(c context.Context, req reports.LinkAuditPageRequest) (reports.LinkAuditPageResponse, error) {
	return s.linkAuditPage(c, req)
}

func (s *ReportsService) JudgePage(c context.Context, req content.JudgeRequest) (content.JudgeResponse, error) {
	return s.judgePage(c, req)
}
