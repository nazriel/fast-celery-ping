package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all configuration options
type Config struct {
	// Broker configuration
	BrokerURL  string
	BrokerType string
	Database   int
	Username   string
	Password   string

	// Ping configuration
	Timeout      time.Duration
	OutputFormat string
	Verbose      bool
	Destination  []string

	// Advanced options
	MaxWorkers    int
	RetryAttempts int
}

// DefaultConfig returns a configuration with sensible defaults
func DefaultConfig() *Config {
	brokerURL := getEnvWithDefault("BROKER_URL", "redis://localhost:6379/0")
	brokerType := DetectBrokerType(brokerURL)

	return &Config{
		BrokerURL:     brokerURL,
		BrokerType:    brokerType,
		Database:      0,
		Username:      "",
		Password:      "",
		Timeout:       time.Second * 15 / 10, // 1.5 seconds
		OutputFormat:  "text",
		Verbose:       false,
		MaxWorkers:    10,
		RetryAttempts: 3,
	}
}

// LoadFromEnv loads configuration from environment variables
func (c *Config) LoadFromEnv() error {
	if brokerURL := os.Getenv("BROKER_URL"); brokerURL != "" {
		c.BrokerURL = brokerURL
		c.BrokerType = DetectBrokerType(brokerURL)
	}

	// Support generic broker username/password environment variables
	if username := os.Getenv("BROKER_USERNAME"); username != "" {
		c.Username = username
	}

	if password := os.Getenv("BROKER_PASSWORD"); password != "" {
		c.Password = password
	}

	if dbStr := os.Getenv("BROKER_DB"); dbStr != "" {
		if db, err := strconv.Atoi(dbStr); err == nil {
			c.Database = db
		}
	}

	if timeoutStr := os.Getenv("BROKER_TIMEOUT"); timeoutStr != "" {
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

	if _, err := url.Parse(c.BrokerURL); err != nil {
		return fmt.Errorf("invalid broker URL format: %w", err)
	}

	if c.BrokerType != "redis" && c.BrokerType != "amqp" {
		return fmt.Errorf("unsupported broker type: %s (supported: redis, amqp)", c.BrokerType)
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

func DetectBrokerType(brokerURL string) string {
	if brokerURL == "" {
		return "redis" // default
	}

	parsedURL, err := url.Parse(brokerURL)
	if err != nil {
		return "redis" // fallback to redis if URL is invalid
	}

	scheme := strings.ToLower(parsedURL.Scheme)
	switch scheme {
	case "amqp", "amqps":
		return "amqp"
	case "redis", "rediss":
		return "redis"
	default:
		return "redis" // default fallback
	}
}
