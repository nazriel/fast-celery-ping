package broker

import (
	"context"
	"fmt"
	"time"
)

// PingResponse represents a response from a Celery worker
type PingResponse struct {
	WorkerName string `json:"worker_name"`
	Status     string `json:"status"`
	Timestamp  int64  `json:"timestamp"`
}

// Broker interface defines the contract for different message brokers
type Broker interface {
	// Ping sends a ping command to workers and returns their responses
	// If destinations is empty, ping all workers. Otherwise, ping only specified workers.
	Ping(ctx context.Context, timeout time.Duration, destinations []string) (map[string]PingResponse, error)

	// Connect establishes connection to the broker
	Connect(ctx context.Context) error

	// Close closes the connection to the broker
	Close() error

	// Health checks if the broker is reachable
	Health(ctx context.Context) error
}

// Config holds configuration for broker connections
type Config struct {
	URL          string
	Database     int
	Username     string
	Password     string
	Timeout      time.Duration
	OutputFormat string
	MaxWorkers   int
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.URL == "" {
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
