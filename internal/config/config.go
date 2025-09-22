package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/adrg/xdg"
)

// AppConfig holds the application configuration
type AppConfig struct {
	DatabasePath string `json:"database_path"`
	LogLevel     string `json:"log_level"`
}

var appConfig *AppConfig

// GetConfig returns the current application configuration
func GetConfig() *AppConfig {
	if appConfig == nil {
		appConfig = getDefaultConfig()
	}
	return appConfig
}

// LoadConfig loads configuration from file or creates default if not exists
func LoadConfig() (*AppConfig, error) {
	configPath := getHomePath()

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Create default config
		appConfig = getDefaultConfig()
		if err = SaveConfig(appConfig); err != nil {
			return nil, fmt.Errorf("failed to save default config: %w", err)
		}
		return appConfig, nil
	}

	// Load existing config
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config AppConfig
	if err = json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	appConfig = &config
	return appConfig, nil
}

// SaveConfig saves the configuration to file
func SaveConfig(config *AppConfig) error {
	configPath := getHomePath()

	// Ensure the directory exists
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Marshal config to JSON
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err = os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// getHomePath returns the path to the configuration file
func getHomePath() string {
	return filepath.Join(xdg.ConfigHome, "Postulator", "config.json")
}

// getDefaultConfig returns the default application configuration
func getDefaultConfig() *AppConfig {
	return &AppConfig{
		DatabasePath: "", // Will be set by database package
		LogLevel:     "info",
	}
}
