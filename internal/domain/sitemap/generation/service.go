package generation

import (
	"context"

	"github.com/davidmovas/postulator/internal/domain/aiusage"
	"github.com/davidmovas/postulator/internal/domain/articles"
	"github.com/davidmovas/postulator/internal/domain/entities"
	"github.com/davidmovas/postulator/internal/domain/prompts"
	"github.com/davidmovas/postulator/internal/domain/providers"
	"github.com/davidmovas/postulator/internal/domain/sitemap"
	"github.com/davidmovas/postulator/internal/domain/sites"
	"github.com/davidmovas/postulator/internal/infra/ai"
	"github.com/davidmovas/postulator/internal/infra/events"
	"github.com/davidmovas/postulator/internal/infra/wp"
	"github.com/davidmovas/postulator/pkg/logger"
)

type Service interface {
	StartGeneration(ctx context.Context, config GenerationConfig) (*Task, error)
	PauseGeneration(taskID string) error
	ResumeGeneration(taskID string) error
	CancelGeneration(taskID string) error
	GetTask(taskID string) *Task
	ListActiveTasks() []*Task
	GetDefaultPrompt() *entities.Prompt
}

type serviceImpl struct {
	executor    *Executor
	rateLimiter *RateLimiter
	logger      *logger.Logger
}

func NewService(
	sitemapSvc sitemap.Service,
	articleSvc articles.Service,
	siteSvc sites.Service,
	promptSvc prompts.Service,
	providerSvc providers.Service,
	aiUsageService aiusage.Service,
	wpClient wp.Client,
	eventBus *events.EventBus,
	aiClientFactory func(provider *entities.Provider) (ai.Client, error),
	logger *logger.Logger,
) Service {
	log := logger.WithScope("page_generation")
	rateLimiter := NewRateLimiter()

	generator := NewGenerator(
		sitemapSvc,
		promptSvc,
		providerSvc,
		aiClientFactory,
		rateLimiter,
		aiUsageService,
		log,
	)

	publisher := NewPublisher(
		sitemapSvc,
		articleSvc,
		siteSvc,
		wpClient,
		log,
	)

	executor := NewExecutor(
		sitemapSvc,
		generator,
		publisher,
		eventBus,
		log,
	)

	return &serviceImpl{
		executor:    executor,
		rateLimiter: rateLimiter,
		logger:      log,
	}
}

func (s *serviceImpl) StartGeneration(ctx context.Context, config GenerationConfig) (*Task, error) {
	if config.MaxConcurrency <= 0 {
		config.MaxConcurrency = 3
	}
	if config.PublishAs == "" {
		config.PublishAs = PublishAsDraft
	}

	s.logger.Infof("Starting page generation: sitemapID=%d, nodes=%d, provider=%d",
		config.SitemapID, len(config.NodeIDs), config.ProviderID)

	return s.executor.Start(ctx, config)
}

func (s *serviceImpl) PauseGeneration(taskID string) error {
	return s.executor.Pause(taskID)
}

func (s *serviceImpl) ResumeGeneration(taskID string) error {
	return s.executor.Resume(taskID)
}

func (s *serviceImpl) CancelGeneration(taskID string) error {
	return s.executor.Cancel(taskID)
}

func (s *serviceImpl) GetTask(taskID string) *Task {
	return s.executor.GetTask(taskID)
}

func (s *serviceImpl) ListActiveTasks() []*Task {
	return s.executor.ListActiveTasks()
}

func (s *serviceImpl) GetDefaultPrompt() *entities.Prompt {
	return GetDefaultPromptEntity()
}
