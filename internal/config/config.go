package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// AppConfig holds the application configuration
type AppConfig struct {
	DatabasePath string       `json:"database_path"`
	LogLevel     string       `json:"log_level"`
	WindowConfig WindowConfig `json:"window_config"`
}

// WindowConfig holds window-specific configuration
type WindowConfig struct {
	Width     int  `json:"width"`
	Height    int  `json:"height"`
	MinWidth  int  `json:"min_width"`
	MinHeight int  `json:"min_height"`
	Resizable bool `json:"resizable"`
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
	configPath, err := getConfigPath()
	if err != nil {
		return nil, fmt.Errorf("failed to get config path: %w", err)
	}

	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Create default config
		appConfig = getDefaultConfig()
		if err := SaveConfig(appConfig); err != nil {
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
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	appConfig = &config
	return appConfig, nil
}

// SaveConfig saves the configuration to file
func SaveConfig(config *AppConfig) error {
	configPath, err := getConfigPath()
	if err != nil {
		return fmt.Errorf("failed to get config path: %w", err)
	}

	// Ensure the directory exists
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Marshal config to JSON
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write to file
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// getConfigPath returns the path to the configuration file
func getConfigPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}

	appDir := filepath.Join(configDir, "Postulator")
	configPath := filepath.Join(appDir, "config.json")

	return configPath, nil
}

// getDefaultConfig returns the default application configuration
func getDefaultConfig() *AppConfig {
	return &AppConfig{
		DatabasePath: "", // Will be set by database package
		LogLevel:     "info",
		WindowConfig: WindowConfig{
			Width:     1024,
			Height:    768,
			MinWidth:  800,
			MinHeight: 600,
			Resizable: true,
		},
	}
}

// UpdateWindowConfig updates the window configuration
func UpdateWindowConfig(width, height int) error {
	config := GetConfig()
	config.WindowConfig.Width = width
	config.WindowConfig.Height = height
	return SaveConfig(config)
}
