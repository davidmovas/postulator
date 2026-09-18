package wails_test

import (
	"context"
	"testing"

	"go.uber.org/zap"

	"github.com/davidmovas/postulator/internal/application/content"
	"github.com/davidmovas/postulator/internal/application/reports"
	"github.com/davidmovas/postulator/internal/transport/wails"
)

type reportsFake struct{ mode failure }

func (f reportsFake) SiteOverview(context.Context, reports.SiteOverviewRequest) (reports.SiteOverviewResponse, error) {
	return answer[reports.SiteOverviewResponse](f.mode)
}

func (f reportsFake) PageReport(context.Context, reports.PageReportRequest) (reports.PageReportResponse, error) {
	return answer[reports.PageReportResponse](f.mode)
}

func (f reportsFake) RunReport(context.Context, reports.RunReportRequest) (reports.RunReportResponse, error) {
	return answer[reports.RunReportResponse](f.mode)
}

type judgeFake struct{ mode failure }

func (f judgeFake) Judge(context.Context, content.JudgeRequest) (content.JudgeResponse, error) {
	return answer[content.JudgeResponse](f.mode)
}

func TestReportsServiceConvertsEveryFailure(t *testing.T) {
	t.Parallel()

	assertMethodNames(t, wails.NewReportsService(zap.NewNop(),
		ready[wails.ReportsUseCase](reportsFake{}), ready[wails.JudgeUseCase](judgeFake{})), []string{
		"JudgePage", "PageReport", "RunReport", "SiteOverview",
	})
	assertEveryMethodConverts(t, wails.NewReportsService(zap.NewNop(),
		ready[wails.ReportsUseCase](reportsFake{mode: missing}), ready[wails.JudgeUseCase](judgeFake{mode: missing})), missingBody)
	assertEveryMethodConverts(t, wails.NewReportsService(zap.NewNop(),
		ready[wails.ReportsUseCase](reportsFake{mode: panicking}), ready[wails.JudgeUseCase](judgeFake{mode: panicking})), panicBody)
}
