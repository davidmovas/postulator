package wails_test

import (
	"context"
	"testing"

	"go.uber.org/zap"

	"github.com/davidmovas/postulator/internal/application/findings"
	"github.com/davidmovas/postulator/internal/kernel/paging"
	"github.com/davidmovas/postulator/internal/transport/wails"
)

type findingsFake struct{ mode failure }

func (f findingsFake) List(context.Context, findings.ListRequest) (paging.List[findings.Finding], error) {
	return answer[paging.List[findings.Finding]](f.mode)
}

func (f findingsFake) Get(context.Context, findings.GetRequest) (findings.Finding, error) {
	return answer[findings.Finding](f.mode)
}

func (f findingsFake) Counts(context.Context, findings.CountsRequest) (findings.CountsResponse, error) {
	return answer[findings.CountsResponse](f.mode)
}

func (f findingsFake) Explain(context.Context, findings.ExplainRequest) (findings.Explanation, error) {
	return answer[findings.Explanation](f.mode)
}

func (f findingsFake) Resolve(context.Context, findings.ResolveRequest) (findings.ResolveResponse, error) {
	return answer[findings.ResolveResponse](f.mode)
}

func (f findingsFake) ResolveMany(context.Context, findings.ResolveManyRequest) (findings.ResolveManyResponse, error) {
	return answer[findings.ResolveManyResponse](f.mode)
}

func findingsService(mode failure) *wails.FindingsService {
	return wails.NewFindingsService(zap.NewNop(), ready[wails.FindingsUseCase](findingsFake{mode: mode}))
}

func TestFindingsServiceConvertsEveryFailure(t *testing.T) {
	t.Parallel()

	assertMethodNames(t, findingsService(missing), []string{"Counts", "Explain", "Get", "List", "Resolve", "ResolveMany"})
	assertEveryMethodConverts(t, findingsService(missing), missingBody)
	assertEveryMethodConverts(t, findingsService(panicking), panicBody)
}
