package schedules

import (
	"context"
	"time"

	"github.com/davidmovas/postulator/internal/application/runs"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	domainrun "github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/domain/schedule"
	"github.com/davidmovas/postulator/internal/domain/site"
	"github.com/davidmovas/postulator/internal/kernel/clock"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

const DueBatch = 20

type scheduleStore interface {
	Insert(ctx context.Context, s schedule.Schedule) error
	Update(ctx context.Context, s schedule.Schedule) error
	Delete(ctx context.Context, id string) error
	Get(ctx context.Context, id string) (schedule.Schedule, error)
	Due(ctx context.Context, now time.Time, limit int) ([]schedule.Schedule, error)
	List(ctx context.Context, q schedule.Query, page paging.Request) (paging.List[schedule.Schedule], error)
}

type pageReader interface {
	ListBySite(ctx context.Context, siteID string) ([]pagemap.Page, error)
}

type siteReader interface {
	Get(ctx context.Context, id string) (site.Site, error)
}

type runStarter interface {
	Start(ctx context.Context, req runs.StartRequest) (runs.StartResponse, error)
}

type runReader interface {
	Get(ctx context.Context, id string) (domainrun.Run, error)
}

type Deps struct {
	Schedules scheduleStore
	Pages     pageReader
	Sites     siteReader
	Runs      runStarter
	RunReader runReader
	Clock     clock.Clock
}

type Service struct {
	deps Deps
}

func New(deps Deps) *Service {
	return &Service{deps: deps}
}

func (s *Service) now() time.Time {
	return s.deps.Clock.Now().UTC().Truncate(time.Second)
}

func invalid(message, field string) *errors.Error {
	return errors.New(errors.Invalid, message).WithDetail("field", field)
}
