package execution

import (
	"context"

	"github.com/davidmovas/postulator/internal/domain/entities"
	"github.com/davidmovas/postulator/internal/domain/jobs"
	"github.com/davidmovas/postulator/internal/domain/stats"
)

var _ stats.ExecutionStatsReader = (*execStatsAdapter)(nil)

type execStatsAdapter struct {
	exec Service
	jobs jobs.Repository
}

func NewExecutionStatsAdapter(exec Service, jobs jobs.Repository) stats.ExecutionStatsReader {
	return &execStatsAdapter{exec: exec, jobs: jobs}
}

func (a *execStatsAdapter) GetPendingValidations(ctx context.Context) ([]*entities.Execution, error) {
	return a.exec.GetPendingValidations(ctx)
}

func (a *execStatsAdapter) ListExecutions(ctx context.Context, offset, limit int, siteID int64) ([]*entities.Execution, int, error) {
	jobsList, err := a.jobs.GetAll(ctx)
	if err != nil {
		return nil, 0, err
	}

	var all []*entities.Execution
	for _, j := range jobsList {
		var exs []*entities.Execution
		exs, _, err = a.exec.ListExecutions(ctx, j.ID, 10000, 0)
		if err != nil {
			return nil, 0, err
		}

		if siteID > 0 {
			for _, e := range exs {
				if e.SiteID == siteID {
					all = append(all, e)
				}
			}
		} else {
			all = append(all, exs...)
		}
	}

	total := len(all)
	if offset < 0 {
		offset = 0
	}
	if offset > total {
		return []*entities.Execution{}, total, nil
	}
	end := offset + limit
	if limit <= 0 || end > total {
		end = total
	}
	return all[offset:end], total, nil
}
