package main

import (
	"Postulator/internal/services/topic_strategy"
	"context"
	"database/sql"
	"log"
	"time"

	"Postulator/internal/bindings"
	"Postulator/internal/config"
	"Postulator/internal/handlers"
	"Postulator/internal/repository"
	"Postulator/internal/services/gpt"
	"Postulator/internal/services/pipeline"
	"Postulator/internal/services/wordpress"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx      context.Context
	cancel   context.CancelFunc
	handler  *handlers.Handler
	services *ServiceContainer
	repo     *repository.Repository
	binder   *bindings.Binder
}

// ServiceContainer holds all application services
type ServiceContainer struct {
	TopicStrategyService *topic_strategy.TopicStrategyService
	GPT                  *gpt.Service
	WordPress            *wordpress.Service
	Pipeline             *pipeline.Service
}

// NewApp creates a new App application struct
func NewApp() *App {
	app := &App{
		binder: &bindings.Binder{},
	}
	return app
}

func (a *App) CTX() context.Context {
	return a.ctx
}

// startup is called at application startup
func (a *App) startup(ctx context.Context) {
	appCtx, cancel := context.WithCancel(ctx)
	a.ctx = appCtx
	a.cancel = cancel

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Printf("Error loading configuration: %v", err)
		cfg = &config.Config{}
	}

	var db *sql.DB
	if db, err = repository.InitDatabase(); err != nil {
		log.Printf("Error initializing database: %v", err)
		return
	} else {
		log.Println("Database initialized successfully")
	}

	a.repo = repository.NewRepository(db)
	a.services = a.initializeServices(cfg)
	a.handler = handlers.NewHandler(
		ctx,
		a.services.GPT,
		a.services.WordPress,
		a.services.Pipeline,
		a.services.TopicStrategyService,
		a.repo,
	)

	// Initialize binders for Wails frontend - set the handler
	a.binder.SetHandler(a.handler)

	log.Println("Application initialized successfully")
}

// initializeServices initializes all application services
func (a *App) initializeServices(_ *config.Config) *ServiceContainer {
	// Initialize GPT service with default values (will be configurable later)
	gptConfig := gpt.Config{
		APIKey:    "", // To be set through settings
		Model:     "gpt-3.5-turbo",
		MaxTokens: 4000,
		Timeout:   60 * time.Second,
	}
	gptService := gpt.NewService(gptConfig, a.repo)

	// Initialize WordPress service
	wpConfig := wordpress.Config{
		Timeout: 30 * time.Second,
	}

	topicStrategyService := topic_strategy.NewTopicStrategyService(a.repo)

	wpService := wordpress.NewService(wpConfig)

	// Initialize Pipeline service
	pipelineConfig := pipeline.Config{
		MaxWorkers:       5,
		JobTimeout:       900 * time.Second,
		RetryCount:       3,
		RetryDelay:       3 * time.Second,
		MinContentWords:  500,
		MaxDailyPosts:    10,
		WordPressTimeout: 30 * time.Second,
		GPTTimeout:       60 * time.Second,
	}
	pipelineService := pipeline.NewService(pipelineConfig, a.repo, gptService, wpService, topicStrategyService, a.ctx)

	return &ServiceContainer{
		GPT:                  gptService,
		WordPress:            wpService,
		Pipeline:             pipelineService,
		TopicStrategyService: topicStrategyService,
	}
}

// domReady is called after front-end resources have been loaded
func (a *App) domReady(_ context.Context) {
	// Add your action here
}

// beforeClose is called when the application is about to quit,
// either by clicking the window close button or calling runtime.Quit.
// Returning true will cause the application to continue, false will continue shutdown as normal.
func (a *App) beforeClose(ctx context.Context) (prevent bool) {
	runtime.WindowHide(ctx)
	return true
}

// shutdown is called at application termination
func (a *App) shutdown(_ context.Context) {
	// Perform your teardown here
	log.Println("Shutting down application...")

	// Close database connection
	if err := a.repo.Close(); err != nil {
		log.Printf("Error closing database: %v", err)
	} else {
		log.Println("Database closed successfully")
	}
}

// ShowWindow shows the application window
func (a *App) ShowWindow() {
	runtime.WindowShow(a.ctx)
}

// QuitApp quits the application
func (a *App) QuitApp() {
	runtime.Quit(a.ctx)
}
