package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds all configuration options
type Config struct {
	// Broker configuration
	BrokerURL      string
	BrokerType     string
	Database       int
	Username       string
	Password       string
	
	// Ping configuration
	Timeout        time.Duration
	OutputFormat   string
	Verbose        bool
	
	// Advanced options
	MaxWorkers     int
	RetryAttempts  int
}

// DefaultConfig returns a configuration with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		BrokerURL:     getEnvWithDefault("CELERY_BROKER_URL", "redis://localhost:6379/0"),
		BrokerType:    "redis",
		Database:      0,
		Username:      "",
		Password:      "",
		Timeout:       time.Second * 1,
		OutputFormat:  "json",
		Verbose:       false,
		MaxWorkers:    10,
		RetryAttempts: 3,
	}
}

// LoadFromEnv loads configuration from environment variables
func (c *Config) LoadFromEnv() error {
	if brokerURL := os.Getenv("CELERY_BROKER_URL"); brokerURL != "" {
		c.BrokerURL = brokerURL
	}
	
	if username := os.Getenv("REDIS_USERNAME"); username != "" {
		c.Username = username
	}
	
	if password := os.Getenv("REDIS_PASSWORD"); password != "" {
		c.Password = password
	}
	
	if dbStr := os.Getenv("REDIS_DB"); dbStr != "" {
		if db, err := strconv.Atoi(dbStr); err == nil {
			c.Database = db
		}
	}
	
	if timeoutStr := os.Getenv("CELERY_PING_TIMEOUT"); timeoutStr != "" {
		if timeout, err := time.ParseDuration(timeoutStr); err == nil {
			c.Timeout = timeout
		}
	}
	
	if format := os.Getenv("OUTPUT_FORMAT"); format != "" {
		c.OutputFormat = format
	}
	
	if verboseStr := os.Getenv("VERBOSE"); verboseStr != "" {
		c.Verbose = verboseStr == "true" || verboseStr == "1"
	}
	
	return nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.BrokerURL == "" {
		return fmt.Errorf("broker URL is required")
	}
	
	if c.Timeout <= 0 {
		return fmt.Errorf("timeout must be positive")
	}
	
	if c.OutputFormat != "json" && c.OutputFormat != "text" {
		return fmt.Errorf("output format must be 'json' or 'text'")
	}
	
	if c.MaxWorkers <= 0 {
		return fmt.Errorf("max workers must be positive")
	}
	
	return nil
}

// getEnvWithDefault gets environment variable with a default value
func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}