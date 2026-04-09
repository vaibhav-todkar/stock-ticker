// Package config provides configuration management for the stock ticker service.
package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config holds the entire application configuration.
type Config struct {
	API     APIConfig     `mapstructure:"api"`
	Server  ServerConfig  `mapstructure:"server"`
	Logging LoggingConfig `mapstructure:"logging"`
}

// APIConfig holds Finnhub API-related settings.
type APIConfig struct {
	BaseURL      string        `mapstructure:"base_url"`
	APIKey       string        `mapstructure:"api_key"`
	Symbols      []string      `mapstructure:"symbols"`
	PollInterval time.Duration `mapstructure:"poll_interval"`
	Timeout      time.Duration `mapstructure:"timeout"`
	MaxRetries   int           `mapstructure:"max_retries"`
}

// ServerConfig holds HTTP server settings.
type ServerConfig struct {
	Port int `mapstructure:"port"`
}

// LoggingConfig holds logging settings.
type LoggingConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

// Load reads configuration from the config file and environment variables.
// Environment variables override file values. The FINNHUB_API_KEY env var
// maps to api.api_key.
func Load() (*Config, error) {
	v := viper.New()

	// Defaults
	v.SetDefault("api.base_url", "https://finnhub.io/api/v1")
	v.SetDefault("api.symbols", []string{"AAPL", "GOOGL", "MSFT", "TSLA", "AMZN"})
	v.SetDefault("api.poll_interval", "10s")
	v.SetDefault("api.timeout", "10s")
	v.SetDefault("api.max_retries", 3)
	v.SetDefault("server.port", 8080)
	v.SetDefault("logging.level", "info")
	v.SetDefault("logging.format", "json")

	// Config file
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AddConfigPath("./config")

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
		// Config file not found is acceptable; rely on defaults + env vars.
	}

	// Environment variable support
	v.SetEnvPrefix("")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Explicit binding for the API key from FINNHUB_API_KEY
	if err := v.BindEnv("api.api_key", "FINNHUB_API_KEY"); err != nil {
		return nil, fmt.Errorf("error binding env var: %w", err)
	}

	cfg := &Config{}
	if err := v.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("error unmarshalling config: %w", err)
	}

	return cfg, nil
}
