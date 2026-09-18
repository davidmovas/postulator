package wails

import (
	"context"

	"go.uber.org/zap"

	"github.com/davidmovas/postulator/internal/application/sites"
	"github.com/davidmovas/postulator/internal/kernel/middleware"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

type sitesUseCase interface {
	Create(ctx context.Context, req sites.CreateRequest) (sites.CreateResponse, error)
	Update(ctx context.Context, req sites.UpdateRequest) (sites.UpdateResponse, error)
	Delete(ctx context.Context, req sites.DeleteRequest) (sites.DeleteResponse, error)
	Get(ctx context.Context, req sites.GetRequest) (sites.GetResponse, error)
	List(ctx context.Context, req sites.ListRequest) (paging.List[sites.Site], error)
}

type SitesService struct {
	create middleware.Handler[sites.CreateRequest, sites.CreateResponse]
	update middleware.Handler[sites.UpdateRequest, sites.UpdateResponse]
	remove middleware.Handler[sites.DeleteRequest, sites.DeleteResponse]
	get    middleware.Handler[sites.GetRequest, sites.GetResponse]
	list   middleware.Handler[sites.ListRequest, paging.List[sites.Site]]
}

func NewSitesService(logger *zap.Logger, useCase sitesUseCase) *SitesService {
	return &SitesService{
		create: Wrap(logger, "sites.create", useCase.Create),
		update: Wrap(logger, "sites.update", useCase.Update),
		remove: Wrap(logger, "sites.delete", useCase.Delete),
		get:    Wrap(logger, "sites.get", useCase.Get),
		list:   Wrap(logger, "sites.list", useCase.List),
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
