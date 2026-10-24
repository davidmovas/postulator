package wails

import (
	"context"

	"go.uber.org/zap"

	"github.com/davidmovas/postulator/internal/application/findings"
	"github.com/davidmovas/postulator/internal/kernel/middleware"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

type FindingsUseCase interface {
	List(ctx context.Context, req findings.ListRequest) (paging.List[findings.Finding], error)
	Get(ctx context.Context, req findings.GetRequest) (findings.Finding, error)
	Counts(ctx context.Context, req findings.CountsRequest) (findings.CountsResponse, error)
	Explain(ctx context.Context, req findings.ExplainRequest) (findings.Explanation, error)
	Resolve(ctx context.Context, req findings.ResolveRequest) (findings.ResolveResponse, error)
	ResolveMany(ctx context.Context, req findings.ResolveManyRequest) (findings.ResolveManyResponse, error)
}

type FindingsService struct {
	list        middleware.Handler[findings.ListRequest, paging.List[findings.Finding]]
	get         middleware.Handler[findings.GetRequest, findings.Finding]
	counts      middleware.Handler[findings.CountsRequest, findings.CountsResponse]
	explain     middleware.Handler[findings.ExplainRequest, findings.Explanation]
	resolve     middleware.Handler[findings.ResolveRequest, findings.ResolveResponse]
	resolveMany middleware.Handler[findings.ResolveManyRequest, findings.ResolveManyResponse]
}

func NewFindingsService(logger *zap.Logger, useCase Source[FindingsUseCase]) *FindingsService {
	return &FindingsService{
		list:        Wrap(logger, "findings.list", call(useCase, FindingsUseCase.List)),
		get:         Wrap(logger, "findings.get", call(useCase, FindingsUseCase.Get)),
		counts:      Wrap(logger, "findings.counts", call(useCase, FindingsUseCase.Counts)),
		explain:     Wrap(logger, "findings.explain", call(useCase, FindingsUseCase.Explain)),
		resolve:     Wrap(logger, "findings.resolve", call(useCase, FindingsUseCase.Resolve)),
		resolveMany: Wrap(logger, "findings.resolveMany", call(useCase, FindingsUseCase.ResolveMany)),
	}
}

func (s *FindingsService) List(c context.Context, req findings.ListRequest) (paging.List[findings.Finding], error) {
	return s.list(c, req)
}

func (s *FindingsService) Get(c context.Context, req findings.GetRequest) (findings.Finding, error) {
	return s.get(c, req)
}

func (s *FindingsService) Counts(c context.Context, req findings.CountsRequest) (findings.CountsResponse, error) {
	return s.counts(c, req)
}

func (s *FindingsService) Explain(c context.Context, req findings.ExplainRequest) (findings.Explanation, error) {
	return s.explain(c, req)
}

func (s *FindingsService) Resolve(c context.Context, req findings.ResolveRequest) (findings.ResolveResponse, error) {
	return s.resolve(c, req)
}

func (s *FindingsService) ResolveMany(c context.Context, req findings.ResolveManyRequest) (findings.ResolveManyResponse, error) {
	return s.resolveMany(c, req)
}
