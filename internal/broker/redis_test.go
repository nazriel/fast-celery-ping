package broker

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestRedisBroker_NewRedisBroker(t *testing.T) {
	config := Config{
		URL:      "redis://localhost:6379/0",
		Database: 0,
		Username: "",
		Password: "",
	}

	broker := NewRedisBroker(config)

	if broker == nil {
		t.Fatal("Expected broker to be created, got nil")
	}

	if broker.config.URL != config.URL {
		t.Errorf("Expected URL %s, got %s", config.URL, broker.config.URL)
	}

	if broker.handler == nil {
		t.Fatal("Expected handler to be initialized")
	}
}

func TestRedisBroker_Health_NoConnection(t *testing.T) {
	config := Config{
		URL: "redis://localhost:6379/0",
	}

	broker := NewRedisBroker(config)
	ctx := context.Background()

	// Should fail without connection
	err := broker.Health(ctx)
	if err == nil {
		t.Error("Expected health check to fail without connection")
	}
}

func TestRedisBroker_Connect(t *testing.T) {
	tests := []struct {
		name        string
		config      Config
		wantErr     bool
		errContains string
	}{
		{
			name: "invalid Redis URL",
			config: Config{
				URL: "invalid-url",
			},
			wantErr:     true,
			errContains: "failed to parse Redis URL",
		},
		{
			name: "invalid port",
			config: Config{
				URL: "redis://localhost:99999/0", // Non-existent port
			},
			wantErr:     true,
			errContains: "invalid port",
		},
		{
			name: "valid URL but unreachable",
			config: Config{
				URL: "redis://192.0.2.1:6379/0", // RFC5737 TEST-NET-1 (unreachable)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			broker := NewRedisBroker(tt.config)
			ctx := context.Background()

			err := broker.Connect(ctx)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected connection error, got nil")
				} else if tt.errContains != "" {
					if !strings.Contains(err.Error(), tt.errContains) {
						t.Errorf("Expected error to contain '%s', got: %v", tt.errContains, err)
					}
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
			}

			// Always try to close, even if Connect failed
			broker.Close()
		})
	}
}

func TestRedisBroker_Close(t *testing.T) {
	// Test closing without connection
	broker := NewRedisBroker(Config{URL: "redis://localhost:6379/0"})
	err := broker.Close()
	if err != nil {
		t.Errorf("Expected no error closing unconnected broker, got: %v", err)
	}

	// Test closing after failed connection attempt
	badBroker := NewRedisBroker(Config{URL: "invalid-url"})
	ctx := context.Background()
	badBroker.Connect(ctx) // This will fail
	err = badBroker.Close()
	if err != nil {
		t.Errorf("Expected no error closing failed broker, got: %v", err)
	}
}

func TestRedisBroker_Ping_Errors(t *testing.T) {
	tests := []struct {
		name      string
		setupFunc func() *RedisBroker
		wantErr   bool
		errMsg    string
	}{
		{
			name: "uninitialized client",
			setupFunc: func() *RedisBroker {
				return NewRedisBroker(Config{URL: "redis://localhost:6379/0"})
			},
			wantErr: true,
			errMsg:  "Redis client not initialized",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			broker := tt.setupFunc()
			ctx := context.Background()

			responses, err := broker.Ping(ctx, time.Second, nil)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected ping error, got nil")
				} else if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Expected error to contain '%s', got: %v", tt.errMsg, err)
				}
				if responses != nil {
					t.Error("Expected nil responses on error")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
			}
		})
	}
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: Config{
				URL:          "redis://localhost:6379/0",
				Timeout:      time.Second,
				OutputFormat: "json",
				MaxWorkers:   10,
			},
			wantErr: false,
		},
		{
			name: "empty broker URL",
			config: Config{
				URL:          "",
				Timeout:      time.Second,
				OutputFormat: "json",
				MaxWorkers:   10,
			},
			wantErr: true,
		},
		{
			name: "invalid timeout",
			config: Config{
				URL:          "redis://localhost:6379/0",
				Timeout:      0,
				OutputFormat: "json",
				MaxWorkers:   10,
			},
			wantErr: true,
		},
		{
			name: "invalid output format",
			config: Config{
				URL:          "redis://localhost:6379/0",
				Timeout:      time.Second,
				OutputFormat: "invalid",
				MaxWorkers:   10,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNewBroker(t *testing.T) {
	config := Config{
		URL:      "redis://localhost:6379/0",
		Database: 0,
		Username: "",
		Password: "",
	}

	tests := []struct {
		name        string
		brokerType  string
		expectError bool
	}{
		{
			name:        "redis broker",
			brokerType:  "redis",
			expectError: false,
		},
		{
			name:        "amqp broker",
			brokerType:  "amqp",
			expectError: false,
		},
		{
			name:        "unsupported broker",
			brokerType:  "kafka",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			broker, err := NewBroker(tt.brokerType, config)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error for unsupported broker type, got nil")
				}
				if broker != nil {
					t.Error("Expected nil broker for unsupported type, got non-nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if broker == nil {
					t.Error("Expected non-nil broker, got nil")
				}
			}
		})
	}
}
