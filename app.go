package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"Postulator/internal/config"
	"Postulator/internal/dto"
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
	repos    *repository.RepositoryContainer
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
	gptService := gpt.NewService(gptConfig)

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
		RetryDelay:       300 * time.Second,
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
func (a App) domReady(ctx context.Context) {
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

// Greet returns a greeting for the given name
func (a *App) Greet(name string) string {
	return fmt.Sprintf("Hello %s, It's show time!", name)
}

// ShowWindow shows the application window
func (a *App) ShowWindow() {
	runtime.WindowShow(a.ctx)
}

// QuitApp quits the application
func (a *App) QuitApp() {
	runtime.Quit(a.ctx)
}

// Database-related methods for frontend access

// GetSetting retrieves a setting value
func (a *App) GetSetting(key string) (string, error) {
	setting, err := repository.GetSetting(key)
	if err != nil {
		return "", err
	}
	return setting.Value, nil
}

// SetSetting sets a setting value
func (a *App) SetSetting(key, value string) error {
	return repository.SetSetting(key, value)
}

// Site Management Methods - Wails API bindings

// CreateSite creates a new WordPress site
func (a *App) CreateSite(req dto.CreateSiteRequest) *dto.BaseResponse {
	if a.handlers == nil {
		return dto.ErrorResponse(fmt.Errorf("handlers not initialized"))
	}
	return a.handlers.CreateSite(req)
}

// GetSites retrieves all sites with pagination
func (a *App) GetSites(pagination dto.PaginationRequest) *dto.BaseResponse {
	if a.handlers == nil {
		return dto.ErrorResponse(fmt.Errorf("handlers not initialized"))
	}
	return a.handlers.GetSites(pagination)
}

// UpdateSite updates an existing site
func (a *App) UpdateSite(req dto.UpdateSiteRequest) *dto.BaseResponse {
	if a.handlers == nil {
		return dto.ErrorResponse(fmt.Errorf("handlers not initialized"))
	}
	return a.handlers.UpdateSite(req)
}

// DeleteSite deletes a site
func (a *App) DeleteSite(siteID int64) *dto.BaseResponse {
	if a.handlers == nil {
		return dto.ErrorResponse(fmt.Errorf("handlers not initialized"))
	}
	return a.handlers.DeleteSite(siteID)
}

// TestSiteConnection tests connection to a WordPress site
func (a *App) TestSiteConnection(req dto.TestSiteConnectionRequest) *dto.BaseResponse {
	if a.handlers == nil {
		return dto.ErrorResponse(fmt.Errorf("handlers not initialized"))
	}
	return a.handlers.TestSiteConnection(req)
}

// Topic Management Methods

// CreateTopic creates a new topic
func (a *App) CreateTopic(req dto.CreateTopicRequest) *dto.BaseResponse {
	if a.handlers == nil {
		return dto.ErrorResponse(fmt.Errorf("handlers not initialized"))
	}
	return a.handlers.CreateTopic(req)
}

// GetTopics retrieves all topics with pagination
func (a *App) GetTopics(pagination dto.PaginationRequest) *dto.BaseResponse {
	if a.handlers == nil {
		return dto.ErrorResponse(fmt.Errorf("handlers not initialized"))
	}
	return a.handlers.GetTopics(pagination)
}

// Schedule Management Methods

// CreateSchedule creates a new posting schedule
func (a *App) CreateSchedule(req dto.CreateScheduleRequest) *dto.BaseResponse {
	if a.handlers == nil {
		return dto.ErrorResponse(fmt.Errorf("handlers not initialized"))
	}
	return a.handlers.CreateSchedule(req)
}

// GetSchedules retrieves all schedules with pagination
func (a *App) GetSchedules(pagination dto.PaginationRequest) *dto.BaseResponse {
	if a.handlers == nil {
		return dto.ErrorResponse(fmt.Errorf("handlers not initialized"))
	}
	return a.handlers.GetSchedules(pagination)
}

// Article Management Methods

// CreateArticle creates a new article manually
func (a *App) CreateArticle(req dto.CreateArticleManualRequest) *dto.BaseResponse {
	if a.handlers == nil {
		return dto.ErrorResponse(fmt.Errorf("handlers not initialized"))
	}
	return a.handlers.CreateArticle(req)
}

// GetArticles retrieves all articles with pagination
func (a *App) GetArticles(pagination dto.PaginationRequest) *dto.BaseResponse {
	if a.handlers == nil {
		return dto.ErrorResponse(fmt.Errorf("handlers not initialized"))
	}
	return a.handlers.GetArticles(pagination)
}

// PreviewArticle generates a preview of an article without saving
func (a *App) PreviewArticle(req dto.PreviewArticleRequest) *dto.BaseResponse {
	if a.handlers == nil {
		return dto.ErrorResponse(fmt.Errorf("handlers not initialized"))
	}
	return a.handlers.PreviewArticle(req)
}

// PostingJob Management Methods

// GetPostingJobs retrieves all posting jobs with pagination
func (a *App) GetPostingJobs(pagination dto.PaginationRequest) *dto.BaseResponse {
	if a.handlers == nil {
		return dto.ErrorResponse(fmt.Errorf("handlers not initialized"))
	}
	return a.handlers.GetPostingJobs(pagination)
}

// Dashboard Methods

// GetDashboard retrieves dashboard data
func (a *App) GetDashboard() *dto.BaseResponse {
	if a.handlers == nil {
		return dto.ErrorResponse(fmt.Errorf("handlers not initialized"))
	}
	return a.handlers.GetDashboard()
}

// Settings Management Methods

// GetSettings retrieves all settings
func (a *App) GetSettings() *dto.BaseResponse {
	if a.handlers == nil {
		return dto.ErrorResponse(fmt.Errorf("handlers not initialized"))
	}
	return a.handlers.GetSettings()
}

// UpdateSetting updates a setting
func (a *App) UpdateSetting(req dto.SettingRequest) *dto.BaseResponse {
	if a.handlers == nil {
		return dto.ErrorResponse(fmt.Errorf("handlers not initialized"))
	}
	return a.handlers.UpdateSetting(req)
}
