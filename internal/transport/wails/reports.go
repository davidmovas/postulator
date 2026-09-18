package wails

import (
	"context"

	"go.uber.org/zap"

	"github.com/davidmovas/postulator/internal/application/reports"
	"github.com/davidmovas/postulator/internal/kernel/middleware"
)

type reportsUseCase interface {
	SiteOverview(ctx context.Context, req reports.SiteOverviewRequest) (reports.SiteOverviewResponse, error)
	PageReport(ctx context.Context, req reports.PageReportRequest) (reports.PageReportResponse, error)
	RunReport(ctx context.Context, req reports.RunReportRequest) (reports.RunReportResponse, error)
}

type ReportsService struct {
	siteOverview middleware.Handler[reports.SiteOverviewRequest, reports.SiteOverviewResponse]
	pageReport   middleware.Handler[reports.PageReportRequest, reports.PageReportResponse]
	runReport    middleware.Handler[reports.RunReportRequest, reports.RunReportResponse]
}

func NewReportsService(logger *zap.Logger, useCase reportsUseCase) *ReportsService {
	return &ReportsService{
		siteOverview: Wrap(logger, "reports.siteOverview", useCase.SiteOverview),
		pageReport:   Wrap(logger, "reports.pageReport", useCase.PageReport),
		runReport:    Wrap(logger, "reports.runReport", useCase.RunReport),
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
