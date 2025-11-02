package app

import (
	"context"
	"path/filepath"
	"reflect"

	"github.com/davidmovas/postulator/internal/config"
	"github.com/davidmovas/postulator/internal/domain/articles"
	"github.com/davidmovas/postulator/internal/domain/jobs"
	"github.com/davidmovas/postulator/internal/domain/prompts"
	"github.com/davidmovas/postulator/internal/domain/providers"
	"github.com/davidmovas/postulator/internal/domain/sites"
	"github.com/davidmovas/postulator/internal/domain/topics"
	"github.com/davidmovas/postulator/internal/infra/ai"
	"github.com/davidmovas/postulator/internal/infra/database"
	"github.com/davidmovas/postulator/internal/infra/importer"
	"github.com/davidmovas/postulator/internal/infra/wp"
	"github.com/davidmovas/postulator/pkg/di"
	"github.com/davidmovas/postulator/pkg/logger"

	"github.com/adrg/xdg"
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
	topicSvc    topics.IService
	promptSvc   prompts.IService
	aiProvSvc   aiprovider.IService
	importerSvc importer.service
	jobSvc      jobs.IService
	scheduler   jobs.IScheduler
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

	a := &App{
		container: c,
		logger:    l,
		cfg:       cfg,
	}

	if err = a.InitDB(); err != nil {
		return nil, err
	}

	a.InitWP()
	a.InitAI()

	if err = a.BuildServices(); err != nil {
		return nil, err
	}

	return a, nil
}

func (a *App) InitDB() error {
	db, err := database.NewDB(filepath.Join(xdg.ConfigHome, "Postulator", "database.db"))
	if err != nil {
		return err
	}

	a.container.MustRegister(di.Instance[*database.DB](db))
	a.container.AddCloseFunc(func() { db.Close() })
	a.db = db
	return nil
}

func (a *App) InitWP() {
	if a.wpClient == nil {
		a.wpClient = wp.NewClient()
		a.container.MustRegister(di.Instance[*wp.Client](a.wpClient))
	}
}

func (a *App) InitAI() {
	client := ai.NewOpenAIClient()

	a.container.MustRegister(&di.Registration[ai.IClient]{
		Provider:      di.Must[ai.IClient](client),
		Lifecycle:     di.Singleton,
		InterfaceType: reflect.TypeOf((*ai.IClient)(nil)).Elem(),
	})
}

func (a *App) BuildServices() error {
	var err error

	// Register repositories needed by services and executor
	execRepo, err := jobs.NewExecutionRepository(a.container)
	if err != nil {
		return err
	}
	a.container.MustRegister(&di.Registration[jobs.IExecutionRepository]{
		Provider:      di.Must[jobs.IExecutionRepository](execRepo),
		Lifecycle:     di.Singleton,
		InterfaceType: reflect.TypeOf((*jobs.IExecutionRepository)(nil)).Elem(),
	})

	articleRepo, err := articles.NewRepository(a.container)
	if err != nil {
		return err
	}
	a.container.MustRegister(&di.Registration[articles.IRepository]{
		Provider:      di.Must[articles.IRepository](articleRepo),
		Lifecycle:     di.Singleton,
		InterfaceType: reflect.TypeOf((*articles.IRepository)(nil)).Elem(),
	})

	// Build and register services
	a.siteSvc, err = site.NewService(a.container)
	if err != nil {
		return err
	}

	a.container.MustRegister(&di.Registration[site.IService]{
		Provider:      di.Must[site.IService](a.siteSvc),
		Lifecycle:     di.Singleton,
		InterfaceType: reflect.TypeOf((*site.IService)(nil)).Elem(),
	})

	a.topicSvc, err = topics.NewService(a.container)
	if err != nil {
		return err
	}
	a.container.MustRegister(&di.Registration[topics.IService]{
		Provider:      di.Must[topics.IService](a.topicSvc),
		Lifecycle:     di.Singleton,
		InterfaceType: reflect.TypeOf((*topics.IService)(nil)).Elem(),
	})

	a.promptSvc, err = prompts.NewService(a.container)
	if err != nil {
		return err
	}
	a.container.MustRegister(&di.Registration[prompts.IService]{
		Provider:      di.Must[prompts.IService](a.promptSvc),
		Lifecycle:     di.Singleton,
		InterfaceType: reflect.TypeOf((*prompts.IService)(nil)).Elem(),
	})

	a.aiProvSvc, err = aiprovider.NewService(a.container)
	if err != nil {
		return err
	}
	a.container.MustRegister(&di.Registration[aiprovider.IService]{
		Provider:      di.Must[aiprovider.IService](a.aiProvSvc),
		Lifecycle:     di.Singleton,
		InterfaceType: reflect.TypeOf((*aiprovider.IService)(nil)).Elem(),
	})

	a.jobSvc, err = jobs.NewService(a.container)
	if err != nil {
		return err
	}
	a.container.MustRegister(&di.Registration[jobs.IService]{
		Provider:      di.Must[jobs.IService](a.jobSvc),
		Lifecycle:     di.Singleton,
		InterfaceType: reflect.TypeOf((*jobs.IService)(nil)).Elem(),
	})

	if svc, ierr := importer.NewImportService(a.container); ierr != nil {
		return ierr
	} else {
		a.importerSvc = svc
		a.container.MustRegister(&di.Registration[importer.service]{
			Provider:      di.Must[importer.service](a.importerSvc),
			Lifecycle:     di.Singleton,
			InterfaceType: reflect.TypeOf((*importer.service)(nil)).Elem(),
		})
	}

	a.scheduler, err = jobs.NewScheduler(a.container)
	if err != nil {
		return err
	}
	a.container.MustRegister(&di.Registration[jobs.IScheduler]{
		Provider:      di.Must[jobs.IScheduler](a.scheduler),
		Lifecycle:     di.Singleton,
		InterfaceType: reflect.TypeOf((*jobs.IScheduler)(nil)).Elem(),
	})

	return nil
}

func (a *App) Start(ctx context.Context) error {
	return a.scheduler.Start(ctx)
}

func (a *App) RestoreState(ctx context.Context) error {
	return a.scheduler.RestoreState(ctx)
}

func (a *App) Stop() {
	if a.scheduler != nil {
		if err := a.scheduler.Stop(); err != nil {
			a.logger.ErrorWithErr(err, "Error while stopping scheduler")
		}
	}

	a.logger.Info("Stopping app")
	a.container.Close()
}
