package app

import (
	"Postulator/internal/config"
	"Postulator/internal/domain/aiprovider"
	"Postulator/internal/domain/article"
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
	"path/filepath"
	"reflect"

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
	client := ai.NewClient()

	a.container.MustRegister(&di.Registration[ai.IClient]{
		Provider:      di.Must[ai.IClient](client),
		Lifecycle:     di.Singleton,
		InterfaceType: reflect.TypeOf((*ai.IClient)(nil)).Elem(),
	})
}

func (a *App) BuildServices() error {
	var err error

	// Register repositories needed by services and executor
	execRepo, err := job.NewExecutionRepository(a.container)
	if err != nil {
		return err
	}
	a.container.MustRegister(&di.Registration[job.IExecutionRepository]{
		Provider:      di.Must[job.IExecutionRepository](execRepo),
		Lifecycle:     di.Singleton,
		InterfaceType: reflect.TypeOf((*job.IExecutionRepository)(nil)).Elem(),
	})

	articleRepo, err := article.NewRepository(a.container)
	if err != nil {
		return err
	}
	a.container.MustRegister(&di.Registration[article.IRepository]{
		Provider:      di.Must[article.IRepository](articleRepo),
		Lifecycle:     di.Singleton,
		InterfaceType: reflect.TypeOf((*article.IRepository)(nil)).Elem(),
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

	a.topicSvc, err = topic.NewService(a.container)
	if err != nil {
		return err
	}
	a.container.MustRegister(&di.Registration[topic.IService]{
		Provider:      di.Must[topic.IService](a.topicSvc),
		Lifecycle:     di.Singleton,
		InterfaceType: reflect.TypeOf((*topic.IService)(nil)).Elem(),
	})

	a.promptSvc, err = prompt.NewService(a.container)
	if err != nil {
		return err
	}
	a.container.MustRegister(&di.Registration[prompt.IService]{
		Provider:      di.Must[prompt.IService](a.promptSvc),
		Lifecycle:     di.Singleton,
		InterfaceType: reflect.TypeOf((*prompt.IService)(nil)).Elem(),
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

	a.jobSvc, err = job.NewService(a.container)
	if err != nil {
		return err
	}
	a.container.MustRegister(&di.Registration[job.IService]{
		Provider:      di.Must[job.IService](a.jobSvc),
		Lifecycle:     di.Singleton,
		InterfaceType: reflect.TypeOf((*job.IService)(nil)).Elem(),
	})

	if svc, ierr := importer.NewImportService(a.container); ierr != nil {
		return ierr
	} else {
		a.importerSvc = svc
		a.container.MustRegister(&di.Registration[importer.IImportService]{
			Provider:      di.Must[importer.IImportService](a.importerSvc),
			Lifecycle:     di.Singleton,
			InterfaceType: reflect.TypeOf((*importer.IImportService)(nil)).Elem(),
		})
	}

	a.scheduler, err = job.NewScheduler(a.container)
	if err != nil {
		return err
	}
	a.container.MustRegister(&di.Registration[job.IScheduler]{
		Provider:      di.Must[job.IScheduler](a.scheduler),
		Lifecycle:     di.Singleton,
		InterfaceType: reflect.TypeOf((*job.IScheduler)(nil)).Elem(),
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
