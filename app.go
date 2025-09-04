package main

import (
	"context"
	"log"
	"time"

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
	handlers *handlers.Handler
	repos    *repository.Container
	services *ServiceContainer
}

// ServiceContainer holds all application services
type ServiceContainer struct {
	GPT       *gpt.Service
	WordPress *wordpress.Service
	Pipeline  *pipeline.Service
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called at application startup
func (a *App) startup(ctx context.Context) {
	// Perform your setup here
	a.ctx = ctx
	// Store context globally for systray access
	appContext = ctx

	// Initialize configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Printf("Error loading configuration: %v", err)
		// Use default config
		cfg = &config.AppConfig{}
	}

	// Initialize database
	if err = repository.InitDatabase(); err != nil {
		log.Printf("Error initializing database: %v", err)
		return
	} else {
		log.Println("Database initialized successfully")
	}

	// Initialize repositories
	repos, err := repository.NewRepositoryContainer()
	if err != nil {
		log.Printf("Error initializing repositories: %v", err)
		return
	}
	a.repos = repos

	// Initialize services
	a.services = a.initializeServices(cfg)

	// Initialize handlers
	a.handlers = handlers.NewHandler(
		a.repos,
		a.services.GPT,
		a.services.WordPress,
		a.services.Pipeline,
		ctx,
	)

	log.Println("Application initialized successfully")
}

// initializeServices initializes all application services
func (a *App) initializeServices(cfg *config.AppConfig) *ServiceContainer {
	// Initialize GPT service with default values (will be configurable later)
	gptConfig := gpt.Config{
		APIKey:    "", // To be set through settings
		Model:     "gpt-3.5-turbo",
		MaxTokens: 4000,
		Timeout:   60 * time.Second,
	}
	gptService := gpt.NewService(gptConfig, a.repos)

	// Initialize WordPress service
	wpConfig := wordpress.Config{
		Timeout: 30 * time.Second,
	}

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
	pipelineService := pipeline.NewService(pipelineConfig, a.repos, gptService, wpService, a.ctx)

	return &ServiceContainer{
		GPT:       gptService,
		WordPress: wpService,
		Pipeline:  pipelineService,
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
func (a *App) shutdown(ctx context.Context) {
	// Perform your teardown here
	log.Println("Shutting down application...")

	// Close database connection
	if err := repository.CloseDatabase(); err != nil {
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
