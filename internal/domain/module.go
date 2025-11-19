package domain

import (
	"context"

	"github.com/davidmovas/postulator/internal/domain/articles"
	"github.com/davidmovas/postulator/internal/domain/categories"
	"github.com/davidmovas/postulator/internal/domain/entities"
	"github.com/davidmovas/postulator/internal/domain/healthcheck"
	"github.com/davidmovas/postulator/internal/domain/jobs"
	"github.com/davidmovas/postulator/internal/domain/jobs/execution"
	"github.com/davidmovas/postulator/internal/domain/jobs/schedule"
	"github.com/davidmovas/postulator/internal/domain/linking"
	"github.com/davidmovas/postulator/internal/domain/prompts"
	"github.com/davidmovas/postulator/internal/domain/providers"
	"github.com/davidmovas/postulator/internal/domain/settings"
	"github.com/davidmovas/postulator/internal/domain/sites"
	"github.com/davidmovas/postulator/internal/domain/stats"
	"github.com/davidmovas/postulator/internal/domain/topics"
	"github.com/davidmovas/postulator/internal/infra/window"
	"go.uber.org/fx"
)

var Module = fx.Module("domain",
	fx.Provide(
		// Articles
		articles.NewRepository,
		articles.NewService,

		// Categories
		categories.NewRepository,
		categories.NewStatsRepository,
		categories.NewService,

		// Jobs
		jobs.NewRepository,
		jobs.NewStateRepository,
		jobs.NewService,

		// Execution
		execution.NewRepository,
		execution.NewService,
		execution.NewExecutor,
		// Adapter: bind execution.Service + jobs.Repository to stats.ExecutionStatsReader
		func(es execution.Service, jr jobs.Repository) stats.ExecutionStatsReader {
			return &execStatsAdapter{exec: es, jobs: jr}
		},

		// Linking
		linking.NewRepository,
		linking.NewProposalRepository,
		linking.NewLinkRepository,

		// Prompts
		prompts.NewRepository,
		prompts.NewService,

		// Providers
		providers.NewRepository,
		providers.NewService,

		// Sites
		sites.NewRepository,
		sites.NewService,

		// Stats
		stats.NewRepository,
		stats.NewService,

		// Healthcheck
		healthcheck.NewRepository,
		healthcheck.NewService,
		healthcheck.NewNotifier,

		// Topics
		topics.NewRepository,
		topics.NewUsageRepository,
		topics.NewSiteTopicRepository,
		topics.NewService,

		// Settings
		settings.NewRepository,
		settings.NewService,
	),

	// Job Scheduler lifecycle
	fx.Provide(schedule.NewCalculator, schedule.NewScheduler),
	fx.Invoke(func(lc fx.Lifecycle, scheduler jobs.Scheduler) {
		lc.Append(fx.Hook{
			OnStart: func(ctx context.Context) error {
				return scheduler.Start(ctx)
			},
			OnStop: func(ctx context.Context) error {
				return scheduler.Stop()
			},
		})
	}),

	// Health check Scheduler lifecycle
	fx.Provide(healthcheck.NewScheduler, func() healthcheck.WindowVisibilityChecker {
		return window.IsWindowOpen
	}),
	fx.Invoke(func(lc fx.Lifecycle, scheduler healthcheck.Scheduler) {
		lc.Append(fx.Hook{
			OnStart: func(ctx context.Context) error {
				return scheduler.Start(ctx)
			},
			OnStop: func(ctx context.Context) error {
				return scheduler.Stop()
			},
		})
	}),
)

// execStatsAdapter adapts execution.Service to the narrow stats.ExecutionStatsReader
// interface without creating an import cycle. It augments the missing ListExecutions
// functionality by aggregating per-job execution lists.
type execStatsAdapter struct {
	exec execution.Service
	jobs jobs.Repository
}

func (a *execStatsAdapter) GetPendingValidations(ctx context.Context) ([]*entities.Execution, error) {
	return a.exec.GetPendingValidations(ctx)
}

// ListExecutions aggregates executions across all jobs. The offset/limit are applied
// after aggregation to keep behavior predictable for dashboard summarization use-cases.
func (a *execStatsAdapter) ListExecutions(ctx context.Context, offset, limit int, siteID int64) ([]*entities.Execution, int, error) {
	jobsList, err := a.jobs.GetAll(ctx)
	if err != nil {
		return nil, 0, err
	}

	// Collect all executions from all jobs (simple aggregation; optimize if needed)
	var all []*entities.Execution
	for _, j := range jobsList {
		exs, _, err := a.exec.ListExecutions(ctx, j.ID, 10000, 0)
		if err != nil {
			return nil, 0, err
		}
		// Optional filter by siteID
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
	// Apply offset/limit
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
