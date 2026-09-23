package wails

import (
	"context"

	"go.uber.org/zap"

	"github.com/davidmovas/postulator/internal/application/schedules"
	"github.com/davidmovas/postulator/internal/kernel/middleware"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

type SchedulesUseCase interface {
	Create(ctx context.Context, req schedules.CreateRequest) (schedules.CreateResponse, error)
	Update(ctx context.Context, req schedules.UpdateRequest) (schedules.UpdateResponse, error)
	Delete(ctx context.Context, req schedules.DeleteRequest) (schedules.DeleteResponse, error)
	Get(ctx context.Context, req schedules.GetRequest) (schedules.GetResponse, error)
	List(ctx context.Context, req schedules.ListRequest) (paging.List[schedules.Schedule], error)
	Enable(ctx context.Context, req schedules.EnableRequest) (schedules.EnableResponse, error)
	Disable(ctx context.Context, req schedules.DisableRequest) (schedules.DisableResponse, error)
	RunNow(ctx context.Context, req schedules.RunNowRequest) (schedules.RunNowResponse, error)
}

type SchedulesService struct {
	create  middleware.Handler[schedules.CreateRequest, schedules.CreateResponse]
	update  middleware.Handler[schedules.UpdateRequest, schedules.UpdateResponse]
	remove  middleware.Handler[schedules.DeleteRequest, schedules.DeleteResponse]
	get     middleware.Handler[schedules.GetRequest, schedules.GetResponse]
	list    middleware.Handler[schedules.ListRequest, paging.List[schedules.Schedule]]
	enable  middleware.Handler[schedules.EnableRequest, schedules.EnableResponse]
	disable middleware.Handler[schedules.DisableRequest, schedules.DisableResponse]
	runNow  middleware.Handler[schedules.RunNowRequest, schedules.RunNowResponse]
}

func NewSchedulesService(logger *zap.Logger, useCase Source[SchedulesUseCase]) *SchedulesService {
	return &SchedulesService{
		create:  Wrap(logger, "schedules.create", call(useCase, SchedulesUseCase.Create)),
		update:  Wrap(logger, "schedules.update", call(useCase, SchedulesUseCase.Update)),
		remove:  Wrap(logger, "schedules.delete", call(useCase, SchedulesUseCase.Delete)),
		get:     Wrap(logger, "schedules.get", call(useCase, SchedulesUseCase.Get)),
		list:    Wrap(logger, "schedules.list", call(useCase, SchedulesUseCase.List)),
		enable:  Wrap(logger, "schedules.enable", call(useCase, SchedulesUseCase.Enable)),
		disable: Wrap(logger, "schedules.disable", call(useCase, SchedulesUseCase.Disable)),
		runNow:  Wrap(logger, "schedules.runNow", call(useCase, SchedulesUseCase.RunNow)),
	}
}

func (s *SchedulesService) Create(c context.Context, req schedules.CreateRequest) (schedules.CreateResponse, error) {
	return s.create(c, req)
}

func (s *SchedulesService) Update(c context.Context, req schedules.UpdateRequest) (schedules.UpdateResponse, error) {
	return s.update(c, req)
}

func (s *SchedulesService) Delete(c context.Context, req schedules.DeleteRequest) (schedules.DeleteResponse, error) {
	return s.remove(c, req)
}

func (s *SchedulesService) Get(c context.Context, req schedules.GetRequest) (schedules.GetResponse, error) {
	return s.get(c, req)
}

func (s *SchedulesService) List(c context.Context, req schedules.ListRequest) (paging.List[schedules.Schedule], error) {
	return s.list(c, req)
}

func (s *SchedulesService) Enable(c context.Context, req schedules.EnableRequest) (schedules.EnableResponse, error) {
	return s.enable(c, req)
}

func (s *SchedulesService) Disable(c context.Context, req schedules.DisableRequest) (schedules.DisableResponse, error) {
	return s.disable(c, req)
}

func (s *SchedulesService) RunNow(c context.Context, req schedules.RunNowRequest) (schedules.RunNowResponse, error) {
	return s.runNow(c, req)
}
