package broker

import (
	"context"
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
