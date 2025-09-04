package main

import (
	"context"
	"fmt"
	"log"

	"Postulator/internal/config"
	"Postulator/internal/repository"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx context.Context
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
	if _, err := config.LoadConfig(); err != nil {
		log.Printf("Error loading configuration: %v", err)
	}

	// Initialize database
	if err := repository.InitDatabase(); err != nil {
		log.Printf("Error initializing database: %v", err)
	} else {
		log.Println("Database initialized successfully")
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
