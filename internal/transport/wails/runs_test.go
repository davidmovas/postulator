package wails_test

import (
	"context"
	"testing"

	"go.uber.org/zap"

	"github.com/davidmovas/postulator/internal/application/runs"
	"github.com/davidmovas/postulator/internal/kernel/paging"
	"github.com/davidmovas/postulator/internal/transport/wails"
)

type runsFake struct{ mode failure }

func (f runsFake) Start(context.Context, runs.StartRequest) (runs.StartResponse, error) {
	return answer[runs.StartResponse](f.mode)
}

func (f runsFake) Get(context.Context, runs.GetRequest) (runs.GetResponse, error) {
	return answer[runs.GetResponse](f.mode)
}

func (f runsFake) List(context.Context, runs.ListRequest) (paging.List[runs.Run], error) {
	return answer[paging.List[runs.Run]](f.mode)
}

func (f runsFake) ListItems(context.Context, runs.ListItemsRequest) (paging.List[runs.Item], error) {
	return answer[paging.List[runs.Item]](f.mode)
}

func (f runsFake) ListEvents(context.Context, runs.ListEventsRequest) (runs.ListEventsResponse, error) {
	return answer[runs.ListEventsResponse](f.mode)
}

func (f runsFake) GetArtifact(context.Context, runs.GetArtifactRequest) (runs.GetArtifactResponse, error) {
	return answer[runs.GetArtifactResponse](f.mode)
}

func (f runsFake) ListArtifacts(context.Context, runs.ListArtifactsRequest) (runs.ListArtifactsResponse, error) {
	return answer[runs.ListArtifactsResponse](f.mode)
}

func (f runsFake) Pause(context.Context, runs.PauseRequest) (runs.PauseResponse, error) {
	return answer[runs.PauseResponse](f.mode)
}

func (f runsFake) Resume(context.Context, runs.ResumeRequest) (runs.ResumeResponse, error) {
	return answer[runs.ResumeResponse](f.mode)
}

func (f runsFake) Cancel(context.Context, runs.CancelRequest) (runs.CancelResponse, error) {
	return answer[runs.CancelResponse](f.mode)
}

func (f runsFake) RetryStep(context.Context, runs.RetryStepRequest) (runs.RetryStepResponse, error) {
	return answer[runs.RetryStepResponse](f.mode)
}

func TestRunsServiceConvertsEveryFailure(t *testing.T) {
	t.Parallel()

	assertMethodNames(t, wails.NewRunsService(zap.NewNop(), ready[wails.RunsUseCase](runsFake{})), []string{
		"Cancel", "Get", "GetArtifact", "List", "ListArtifacts", "ListEvents", "ListItems",
		"Pause", "Resume", "RetryStep", "Start",
	})
	assertEveryMethodConverts(t, wails.NewRunsService(zap.NewNop(), ready[wails.RunsUseCase](runsFake{mode: missing})), missingBody)
	assertEveryMethodConverts(t, wails.NewRunsService(zap.NewNop(), ready[wails.RunsUseCase](runsFake{mode: panicking})), panicBody)
}
