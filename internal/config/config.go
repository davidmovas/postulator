package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/adrg/xdg"
)

type Config struct {
	LogLevel    string `json:"logLevel"`
	LogDir      string `json:"logDir"`
	ConsoleOut  bool   `json:"consoleOut"`
	PrettyPrint bool   `json:"prettyPrint"`
	AppLogFile  string `json:"appLogFile"`
	ErrLogFile  string `json:"errLogFile"`
}

func LoadConfig() (*Config, error) {
	configPath := getHomePath()
	var cfg *Config

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		cfg = getDefaultConfig()
		if err = SaveConfig(cfg); err != nil {
			return nil, fmt.Errorf("failed to save default config: %w", err)
		}
		return cfg, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	if err = json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return cfg, nil
}

func SaveConfig(config *Config) error {
	configPath := getHomePath()

	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err = os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

func getHomePath() string {
	return filepath.Join(xdg.ConfigHome, "Postulator", "config.json")
}

func getDefaultConfig() *Config {
	return &Config{
		LogLevel: "info",
	}
}
