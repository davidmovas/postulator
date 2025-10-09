package app

import (
	"Postulator/internal/config"
	"Postulator/internal/domain/aiprovider"
	"Postulator/internal/domain/job"
	"Postulator/internal/domain/prompt"
	"Postulator/internal/domain/site"
	"Postulator/internal/domain/topic"
	"Postulator/internal/infra/ai"
	"Postulator/internal/infra/database"
	"Postulator/internal/infra/importer"
	"Postulator/internal/infra/wp"
	"Postulator/pkg/di"
	"Postulator/pkg/logger"
	"context"
	"reflect"
)

type App struct {
	container di.Container
	logger    *logger.Logger
	cfg       *config.Config

	// Infra
	db       *database.DB
	wpClient *wp.Client

	// Services
	siteSvc     site.IService
	topicSvc    topic.IService
	promptSvc   prompt.IService
	aiProvSvc   aiprovider.IService
	importerSvc importer.IImportService
	jobSvc      job.IService
	scheduler   job.IScheduler
}

func New(cfg *config.Config) (*App, error) {
	c := di.New()
	// Config
	c.MustRegister(di.Instance[*config.Config](cfg))

	// Logger
	l, err := logger.New(cfg)
	if err != nil {
		return nil, err
	}
	c.MustRegister(di.Instance[*logger.Logger](l))

	// Ensure logger files closed on shutdown
	c.AddCloseFunc(func() { _ = l.Close() })

	return &App{
		container: c,
		logger:    l,
		cfg:       cfg,
	}, nil
}

// InitDB opens/creates DB and registers it in DI.
func (a *App) InitDB(dbPath string) error {
	db, err := database.NewDB(dbPath)
	if err != nil {
		return err
	}
	a.container.MustRegister(di.Instance[*database.DB](db))
	a.container.AddCloseFunc(func() { db.Close() })
	a.db = db
	return nil
}

// InitWP registers a WordPress client.
func (a *App) InitWP() {
	if a.wpClient == nil {
		a.wpClient = wp.NewClient()
		a.container.MustRegister(di.Instance[*wp.Client](a.wpClient))
	}
}

// InitAI allows the host to inject an AI client implementation.
func (a *App) InitAI(client ai.Client) {
	if client == nil {
		return
	}
	a.container.MustRegister(&di.Registration[ai.Client]{
		Provider:      di.Must[ai.Client](client),
		Lifecycle:     di.Singleton,
		InterfaceType: reflect.TypeOf((*ai.Client)(nil)).Elem(),
	})
}

// BuildServices resolves and caches domain services & scheduler.
func (a *App) BuildServices() error {
	// Site/Topic/Prompt/AIProvider services
	var err error
	if a.siteSvc == nil {
		a.siteSvc, err = site.NewService(a.container)
		if err != nil {
			return err
		}
	}
	if a.topicSvc == nil {
		a.topicSvc, err = topic.NewService(a.container)
		if err != nil {
			return err
		}
	}
	if a.promptSvc == nil {
		a.promptSvc, err = prompt.NewService(a.container)
		if err != nil {
			return err
		}
	}
	if a.aiProvSvc == nil {
		a.aiProvSvc, err = aiprovider.NewService(a.container)
		if err != nil {
			return err
		}
	}
	if a.jobSvc == nil {
		a.jobSvc, err = job.NewService(a.container)
		if err != nil {
			return err
		}
	}
	if a.importerSvc == nil {
		if svc, ierr := importer.NewImportService(a.container); ierr != nil {
			return ierr
		} else {
			a.importerSvc = svc
		}
	}
	if a.scheduler == nil {
		a.scheduler, err = job.NewScheduler(a.container)
		if err != nil {
			return err
		}
	}
	return nil
}

// Start application subsystems (scheduler) and restore state.
func (a *App) Start(ctx context.Context) error {
	a.InitWP() // ensure WP client present
	if err := a.BuildServices(); err != nil {
		return err
	}
	if err := a.scheduler.RestoreState(ctx); err != nil {
		return err
	}
	return a.scheduler.Start(ctx)
}

func (a *App) RestoreState(ctx context.Context) error {
	if err := a.BuildServices(); err != nil {
		return err
	}
	return a.scheduler.RestoreState(ctx)
}

func (a *App) Stop() {
	if a.scheduler != nil {
		_ = a.scheduler.Stop()
	}
	if a.logger != nil {
		a.logger.Info("Stopping app")
	}
	a.container.Close()
}

func (a *App) SiteService() site.IService             { return a.siteSvc }
func (a *App) TopicService() topic.IService           { return a.topicSvc }
func (a *App) PromptService() prompt.IService         { return a.promptSvc }
func (a *App) AIProviderService() aiprovider.IService { return a.aiProvSvc }
func (a *App) JobService() job.IService               { return a.jobSvc }
