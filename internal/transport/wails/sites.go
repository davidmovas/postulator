package wails

import (
	"context"

	"go.uber.org/zap"

	"github.com/davidmovas/postulator/internal/application/sites"
	"github.com/davidmovas/postulator/internal/kernel/middleware"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

type SitesUseCase interface {
	Create(ctx context.Context, req sites.CreateRequest) (sites.CreateResponse, error)
	Update(ctx context.Context, req sites.UpdateRequest) (sites.UpdateResponse, error)
	Delete(ctx context.Context, req sites.DeleteRequest) (sites.DeleteResponse, error)
	Get(ctx context.Context, req sites.GetRequest) (sites.GetResponse, error)
	List(ctx context.Context, req sites.ListRequest) (paging.List[sites.Site], error)
	TestConnection(ctx context.Context, req sites.TestConnectionRequest) (sites.TestConnectionResponse, error)
}

type SitesService struct {
	create middleware.Handler[sites.CreateRequest, sites.CreateResponse]
	update middleware.Handler[sites.UpdateRequest, sites.UpdateResponse]
	remove middleware.Handler[sites.DeleteRequest, sites.DeleteResponse]
	get    middleware.Handler[sites.GetRequest, sites.GetResponse]
	list   middleware.Handler[sites.ListRequest, paging.List[sites.Site]]
	test   middleware.Handler[sites.TestConnectionRequest, sites.TestConnectionResponse]
}

func NewSitesService(logger *zap.Logger, useCase Source[SitesUseCase]) *SitesService {
	return &SitesService{
		create: Wrap(logger, "sites.create", call(useCase, SitesUseCase.Create)),
		update: Wrap(logger, "sites.update", call(useCase, SitesUseCase.Update)),
		remove: Wrap(logger, "sites.delete", call(useCase, SitesUseCase.Delete)),
		get:    Wrap(logger, "sites.get", call(useCase, SitesUseCase.Get)),
		list:   Wrap(logger, "sites.list", call(useCase, SitesUseCase.List)),
		test:   Wrap(logger, "sites.testConnection", call(useCase, SitesUseCase.TestConnection)),
	}
}

func (s *SitesService) Create(c context.Context, req sites.CreateRequest) (sites.CreateResponse, error) {
	return s.create(c, req)
}

func (s *SitesService) Update(c context.Context, req sites.UpdateRequest) (sites.UpdateResponse, error) {
	return s.update(c, req)
}

func (s *SitesService) Delete(c context.Context, req sites.DeleteRequest) (sites.DeleteResponse, error) {
	return s.remove(c, req)
}

func (s *SitesService) Get(c context.Context, req sites.GetRequest) (sites.GetResponse, error) {
	return s.get(c, req)
}

func (s *SitesService) List(c context.Context, req sites.ListRequest) (paging.List[sites.Site], error) {
	return s.list(c, req)
}

func (s *SitesService) TestConnection(c context.Context, req sites.TestConnectionRequest) (sites.TestConnectionResponse, error) {
	return s.test(c, req)
}
