package wails_test

import (
	"context"
	"testing"

	"go.uber.org/zap"

	"github.com/davidmovas/postulator/internal/application/schedules"
	"github.com/davidmovas/postulator/internal/kernel/paging"
	"github.com/davidmovas/postulator/internal/transport/wails"
)

type schedulesFake struct{ mode failure }

func (f schedulesFake) Create(context.Context, schedules.CreateRequest) (schedules.CreateResponse, error) {
	return answer[schedules.CreateResponse](f.mode)
}

func (f schedulesFake) Update(context.Context, schedules.UpdateRequest) (schedules.UpdateResponse, error) {
	return answer[schedules.UpdateResponse](f.mode)
}

func (f schedulesFake) Delete(context.Context, schedules.DeleteRequest) (schedules.DeleteResponse, error) {
	return answer[schedules.DeleteResponse](f.mode)
}

func (f schedulesFake) Get(context.Context, schedules.GetRequest) (schedules.GetResponse, error) {
	return answer[schedules.GetResponse](f.mode)
}

func (f schedulesFake) List(context.Context, schedules.ListRequest) (paging.List[schedules.Schedule], error) {
	return answer[paging.List[schedules.Schedule]](f.mode)
}

func (f schedulesFake) Enable(context.Context, schedules.EnableRequest) (schedules.EnableResponse, error) {
	return answer[schedules.EnableResponse](f.mode)
}

func (f schedulesFake) Disable(context.Context, schedules.DisableRequest) (schedules.DisableResponse, error) {
	return answer[schedules.DisableResponse](f.mode)
}

func (f schedulesFake) RunNow(context.Context, schedules.RunNowRequest) (schedules.RunNowResponse, error) {
	return answer[schedules.RunNowResponse](f.mode)
}

func TestSchedulesServiceConvertsEveryFailure(t *testing.T) {
	t.Parallel()

	assertMethodNames(t, wails.NewSchedulesService(zap.NewNop(), schedulesFake{}), []string{
		"Create", "Delete", "Disable", "Enable", "Get", "List", "RunNow", "Update",
	})
	assertEveryMethodConverts(t, wails.NewSchedulesService(zap.NewNop(), schedulesFake{mode: missing}), missingBody)
	assertEveryMethodConverts(t, wails.NewSchedulesService(zap.NewNop(), schedulesFake{mode: panicking}), panicBody)
}
