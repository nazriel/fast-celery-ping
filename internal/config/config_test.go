package config

import (
	"os"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config == nil {
		t.Fatal("Expected non-nil config")
	}

	if config.BrokerURL == "" {
		t.Error("Expected non-empty default broker URL")
	}

	if config.BrokerType != "redis" {
		t.Errorf("Expected default broker type 'redis', got %s", config.BrokerType)
	}

	if config.Timeout <= 0 {
		t.Error("Expected positive default timeout")
	}

	if config.OutputFormat == "" {
		t.Error("Expected non-empty default output format")
	}

	if config.MaxWorkers <= 0 {
		t.Error("Expected positive default max workers")
	}

	if config.RetryAttempts <= 0 {
		t.Error("Expected positive default retry attempts")
	}
}

func TestConfig_LoadFromEnv(t *testing.T) {
	// Save original environment
	originalEnv := map[string]string{
		"CELERY_BROKER_URL":   os.Getenv("CELERY_BROKER_URL"),
		"REDIS_USERNAME":      os.Getenv("REDIS_USERNAME"),
		"REDIS_PASSWORD":      os.Getenv("REDIS_PASSWORD"),
		"REDIS_DB":            os.Getenv("REDIS_DB"),
		"CELERY_PING_TIMEOUT": os.Getenv("CELERY_PING_TIMEOUT"),
		"OUTPUT_FORMAT":       os.Getenv("OUTPUT_FORMAT"),
		"VERBOSE":             os.Getenv("VERBOSE"),
	}

	// Clean up function to restore environment
	defer func() {
		for key, value := range originalEnv {
			if value == "" {
				os.Unsetenv(key)
			} else {
				os.Setenv(key, value)
			}
		}
	}()

	tests := []struct {
		name     string
		envVars  map[string]string
		expected func(*Config) bool
	}{
		{
			name: "broker URL from env",
			envVars: map[string]string{
				"CELERY_BROKER_URL": "redis://test:6379/1",
			},
			expected: func(c *Config) bool {
				return c.BrokerURL == "redis://test:6379/1"
			},
		},
		{
			name: "redis credentials from env",
			envVars: map[string]string{
				"REDIS_USERNAME": "testuser",
				"REDIS_PASSWORD": "testpass",
			},
			expected: func(c *Config) bool {
				return c.Username == "testuser" && c.Password == "testpass"
			},
		},
		{
			name: "redis db from env",
			envVars: map[string]string{
				"REDIS_DB": "5",
			},
			expected: func(c *Config) bool {
				return c.Database == 5
			},
		},
		{
			name: "invalid redis db from env",
			envVars: map[string]string{
				"REDIS_DB": "invalid",
			},
			expected: func(c *Config) bool {
				return c.Database == 0 // should keep default
			},
		},
		{
			name: "timeout from env",
			envVars: map[string]string{
				"CELERY_PING_TIMEOUT": "5s",
			},
			expected: func(c *Config) bool {
				return c.Timeout == 5*time.Second
			},
		},
		{
			name: "invalid timeout from env",
			envVars: map[string]string{
				"CELERY_PING_TIMEOUT": "invalid",
			},
			expected: func(c *Config) bool {
				return c.Timeout == time.Second*15/10 // should keep default (1.5s)
			},
		},
		{
			name: "output format from env",
			envVars: map[string]string{
				"OUTPUT_FORMAT": "text",
			},
			expected: func(c *Config) bool {
				return c.OutputFormat == "text"
			},
		},
		{
			name: "verbose true from env",
			envVars: map[string]string{
				"VERBOSE": "true",
			},
			expected: func(c *Config) bool {
				return c.Verbose == true
			},
		},
		{
			name: "verbose 1 from env",
			envVars: map[string]string{
				"VERBOSE": "1",
			},
			expected: func(c *Config) bool {
				return c.Verbose == true
			},
		},
		{
			name: "verbose false from env",
			envVars: map[string]string{
				"VERBOSE": "false",
			},
			expected: func(c *Config) bool {
				return c.Verbose == false
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear environment
			for key := range originalEnv {
				os.Unsetenv(key)
			}

			// Set test environment variables
			for key, value := range tt.envVars {
				os.Setenv(key, value)
			}

			config := DefaultConfig()
			err := config.LoadFromEnv()

			if err != nil {
				t.Fatalf("Unexpected error loading from env: %v", err)
			}

			if !tt.expected(config) {
				t.Error("Expected condition not met for config loaded from environment")
			}
		})
	}
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid config",
			config:  DefaultConfig(),
			wantErr: false,
		},
		{
			name: "empty broker URL",
			config: &Config{
				BrokerURL:    "",
				Timeout:      time.Second,
				OutputFormat: "json",
				MaxWorkers:   10,
			},
			wantErr: true,
			errMsg:  "broker URL is required",
		},
		{
			name: "zero timeout",
			config: &Config{
				BrokerURL:    "redis://localhost:6379/0",
				Timeout:      0,
				OutputFormat: "json",
				MaxWorkers:   10,
			},
			wantErr: true,
			errMsg:  "timeout must be positive",
		},
		{
			name: "negative timeout",
			config: &Config{
				BrokerURL:    "redis://localhost:6379/0",
				Timeout:      -time.Second,
				OutputFormat: "json",
				MaxWorkers:   10,
			},
			wantErr: true,
			errMsg:  "timeout must be positive",
		},
		{
			name: "invalid output format",
			config: &Config{
				BrokerURL:    "redis://localhost:6379/0",
				Timeout:      time.Second,
				OutputFormat: "invalid",
				MaxWorkers:   10,
			},
			wantErr: true,
			errMsg:  "output format must be 'json' or 'text'",
		},
		{
			name: "zero max workers",
			config: &Config{
				BrokerURL:    "redis://localhost:6379/0",
				Timeout:      time.Second,
				OutputFormat: "json",
				MaxWorkers:   0,
			},
			wantErr: true,
			errMsg:  "max workers must be positive",
		},
		{
			name: "negative max workers",
			config: &Config{
				BrokerURL:    "redis://localhost:6379/0",
				Timeout:      time.Second,
				OutputFormat: "json",
				MaxWorkers:   -1,
			},
			wantErr: true,
			errMsg:  "max workers must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if tt.wantErr {
				if err == nil {
					t.Error("Expected validation error, got nil")
				} else if err.Error() != tt.errMsg {
					t.Errorf("Expected error message '%s', got '%s'", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no validation error, got: %v", err)
				}
			}
		})
	}
}

func TestGetEnvWithDefault(t *testing.T) {
	// Save original environment
	originalValue := os.Getenv("TEST_ENV_VAR")
	defer func() {
		if originalValue == "" {
			os.Unsetenv("TEST_ENV_VAR")
		} else {
			os.Setenv("TEST_ENV_VAR", originalValue)
		}
	}()

	tests := []struct {
		name         string
		envValue     string
		defaultValue string
		expected     string
		setEnv       bool
	}{
		{
			name:         "env var exists",
			envValue:     "test_value",
			defaultValue: "default_value",
			expected:     "test_value",
			setEnv:       true,
		},
		{
			name:         "env var empty",
			envValue:     "",
			defaultValue: "default_value",
			expected:     "default_value",
			setEnv:       false,
		},
		{
			name:         "env var not set",
			defaultValue: "default_value",
			expected:     "default_value",
			setEnv:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up
			os.Unsetenv("TEST_ENV_VAR")

			if tt.setEnv {
				os.Setenv("TEST_ENV_VAR", tt.envValue)
			}

			result := getEnvWithDefault("TEST_ENV_VAR", tt.defaultValue)

			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}
