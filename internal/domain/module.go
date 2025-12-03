package domain

import (
	"context"

	"github.com/davidmovas/postulator/internal/domain/articles"
	"github.com/davidmovas/postulator/internal/domain/categories"
	"github.com/davidmovas/postulator/internal/domain/deletion"
	"github.com/davidmovas/postulator/internal/domain/entities"
	"github.com/davidmovas/postulator/internal/domain/healthcheck"
	"github.com/davidmovas/postulator/internal/domain/jobs"
	"github.com/davidmovas/postulator/internal/domain/jobs/execution"
	"github.com/davidmovas/postulator/internal/domain/jobs/schedule"
	"github.com/davidmovas/postulator/internal/domain/linking"
	"github.com/davidmovas/postulator/internal/domain/prompts"
	"github.com/davidmovas/postulator/internal/domain/providers"
	"github.com/davidmovas/postulator/internal/domain/proxy"
	"github.com/davidmovas/postulator/internal/domain/settings"
	"github.com/davidmovas/postulator/internal/domain/sitemap"
	"github.com/davidmovas/postulator/internal/domain/sitemap/scanner"
	"github.com/davidmovas/postulator/internal/domain/sites"
	"github.com/davidmovas/postulator/internal/domain/stats"
	"github.com/davidmovas/postulator/internal/domain/topics"
	"github.com/davidmovas/postulator/internal/infra/ai"
	"github.com/davidmovas/postulator/internal/infra/window"
	"github.com/davidmovas/postulator/pkg/logger"
	"go.uber.org/fx"
)

var Module = fx.Module("domain",
	fx.Provide(
		// Deletion validator
		deletion.NewValidator,

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
		execution.NewExecutionStatsAdapter,

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
		stats.NewRecorder,

		// Healthcheck
		healthcheck.NewRepository,
		healthcheck.NewService,
		healthcheck.NewNotifier,

		// Topics
		topics.NewRepository,
		topics.NewUsageRepository,
		topics.NewSiteTopicRepository,
		// Use jobs.Repository as topics.JobTopicReader to provide cross-domain topic filtering
		fx.Annotate(
			func(repo jobs.Repository) topics.JobTopicReader { return repo },
			fx.As(new(topics.JobTopicReader)),
		),
		topics.NewService,

		// Settings
		settings.NewRepository,
		settings.NewService,

		// Proxy
		proxy.NewService,

		// Sitemap
		sitemap.NewRepository,
		sitemap.NewNodeRepository,
		sitemap.NewKeywordRepository,
		sitemap.NewService,
		sitemap.NewSyncService,

		// Sitemap Generation Service (with AI client factory)
		fx.Annotate(
			func(
				sitemapSvc sitemap.Service,
				sitesSvc sites.Service,
				promptSvc prompts.Service,
				providerSvc providers.Service,
				logger *logger.Logger,
			) *sitemap.GenerationService {
				return sitemap.NewGenerationService(
					sitemapSvc,
					sitesSvc,
					promptSvc,
					providerSvc,
					func(provider *entities.Provider) (ai.Client, error) {
						return ai.CreateClient(provider)
					},
					logger,
				)
			},
		),

		// Sitemap Scanner
		scanner.NewScanner,
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

	// Proxy initialization
	fx.Invoke(func(lc fx.Lifecycle, proxyService proxy.Service) {
		type initializer interface {
			Initialize(context.Context) error
		}
		lc.Append(fx.Hook{
			OnStart: func(ctx context.Context) error {
				if svc, ok := proxyService.(initializer); ok {
					return svc.Initialize(ctx)
				}
				return nil
			},
		})
	}),
)
