package wails

import (
	"context"

	"go.uber.org/zap"

	"github.com/davidmovas/postulator/internal/application/runs"
	"github.com/davidmovas/postulator/internal/kernel/middleware"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

type runsUseCase interface {
	Start(ctx context.Context, req runs.StartRequest) (runs.StartResponse, error)
	Get(ctx context.Context, req runs.GetRequest) (runs.GetResponse, error)
	List(ctx context.Context, req runs.ListRequest) (paging.List[runs.Run], error)
	ListItems(ctx context.Context, req runs.ListItemsRequest) (paging.List[runs.Item], error)
	ListEvents(ctx context.Context, req runs.ListEventsRequest) (runs.ListEventsResponse, error)
	GetArtifact(ctx context.Context, req runs.GetArtifactRequest) (runs.GetArtifactResponse, error)
	Pause(ctx context.Context, req runs.PauseRequest) (runs.PauseResponse, error)
	Resume(ctx context.Context, req runs.ResumeRequest) (runs.ResumeResponse, error)
	Cancel(ctx context.Context, req runs.CancelRequest) (runs.CancelResponse, error)
	RetryStep(ctx context.Context, req runs.RetryStepRequest) (runs.RetryStepResponse, error)
}

type RunsService struct {
	start       middleware.Handler[runs.StartRequest, runs.StartResponse]
	get         middleware.Handler[runs.GetRequest, runs.GetResponse]
	list        middleware.Handler[runs.ListRequest, paging.List[runs.Run]]
	listItems   middleware.Handler[runs.ListItemsRequest, paging.List[runs.Item]]
	listEvents  middleware.Handler[runs.ListEventsRequest, runs.ListEventsResponse]
	getArtifact middleware.Handler[runs.GetArtifactRequest, runs.GetArtifactResponse]
	pause       middleware.Handler[runs.PauseRequest, runs.PauseResponse]
	resume      middleware.Handler[runs.ResumeRequest, runs.ResumeResponse]
	cancel      middleware.Handler[runs.CancelRequest, runs.CancelResponse]
	retryStep   middleware.Handler[runs.RetryStepRequest, runs.RetryStepResponse]
}

func NewRunsService(logger *zap.Logger, useCase runsUseCase) *RunsService {
	return &RunsService{
		start:       Wrap(logger, "runs.start", useCase.Start),
		get:         Wrap(logger, "runs.get", useCase.Get),
		list:        Wrap(logger, "runs.list", useCase.List),
		listItems:   Wrap(logger, "runs.listItems", useCase.ListItems),
		listEvents:  Wrap(logger, "runs.listEvents", useCase.ListEvents),
		getArtifact: Wrap(logger, "runs.getArtifact", useCase.GetArtifact),
		pause:       Wrap(logger, "runs.pause", useCase.Pause),
		resume:      Wrap(logger, "runs.resume", useCase.Resume),
		cancel:      Wrap(logger, "runs.cancel", useCase.Cancel),
		retryStep:   Wrap(logger, "runs.retryStep", useCase.RetryStep),
	}
}

func (s *RunsService) Start(c context.Context, req runs.StartRequest) (runs.StartResponse, error) {
	return s.start(c, req)
}

func (s *RunsService) Get(c context.Context, req runs.GetRequest) (runs.GetResponse, error) {
	return s.get(c, req)
}

func (s *RunsService) List(c context.Context, req runs.ListRequest) (paging.List[runs.Run], error) {
	return s.list(c, req)
}

func (s *RunsService) ListItems(c context.Context, req runs.ListItemsRequest) (paging.List[runs.Item], error) {
	return s.listItems(c, req)
}

func (s *RunsService) ListEvents(c context.Context, req runs.ListEventsRequest) (runs.ListEventsResponse, error) {
	return s.listEvents(c, req)
}

func (s *RunsService) GetArtifact(c context.Context, req runs.GetArtifactRequest) (runs.GetArtifactResponse, error) {
	return s.getArtifact(c, req)
}

func (s *RunsService) Pause(c context.Context, req runs.PauseRequest) (runs.PauseResponse, error) {
	return s.pause(c, req)
}

func (s *RunsService) Resume(c context.Context, req runs.ResumeRequest) (runs.ResumeResponse, error) {
	return s.resume(c, req)
}

func (s *RunsService) Cancel(c context.Context, req runs.CancelRequest) (runs.CancelResponse, error) {
	return s.cancel(c, req)
}

func (s *RunsService) RetryStep(c context.Context, req runs.RetryStepRequest) (runs.RetryStepResponse, error) {
	return s.retryStep(c, req)
}
