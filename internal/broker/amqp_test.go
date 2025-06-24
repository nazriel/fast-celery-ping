package broker

import (
	"context"
	"testing"
	"time"
)

func TestNewAMQPBroker(t *testing.T) {
	config := Config{
		URL:      "amqp://guest:guest@localhost:5672/",
		Database: 0,
		Username: "guest",
		Password: "guest",
	}

	broker := NewAMQPBroker(config)
	if broker == nil {
		t.Fatal("NewAMQPBroker returned nil")
	}

	if broker.config.URL != config.URL {
		t.Errorf("Expected URL %s, got %s", config.URL, broker.config.URL)
	}
}

func TestAMQPBroker_Connect_InvalidURL(t *testing.T) {
	config := Config{
		URL: "invalid://url",
	}

	broker := NewAMQPBroker(config)
	ctx := context.Background()

	err := broker.Connect(ctx)
	if err == nil {
		t.Error("Expected error when connecting with invalid URL, got nil")
	}
}

func TestAMQPBroker_Health_NotConnected(t *testing.T) {
	config := Config{
		URL: "amqp://guest:guest@localhost:5672/",
	}

	broker := NewAMQPBroker(config)
	ctx := context.Background()

	err := broker.Health(ctx)
	if err == nil {
		t.Error("Expected error when checking health without connection, got nil")
	}
}

func TestAMQPBroker_Close_NotConnected(t *testing.T) {
	config := Config{
		URL: "amqp://guest:guest@localhost:5672/",
	}

	broker := NewAMQPBroker(config)

	// Should not panic or error when closing without connection
	err := broker.Close()
	if err != nil {
		t.Errorf("Unexpected error when closing unconnected broker: %v", err)
	}
}

func TestAMQPBroker_Ping_NotConnected(t *testing.T) {
	config := Config{
		URL: "amqp://guest:guest@localhost:5672/",
	}

	broker := NewAMQPBroker(config)
	ctx := context.Background()

	_, err := broker.Ping(ctx, time.Second, nil)
	if err == nil {
		t.Error("Expected error when pinging without connection, got nil")
	}
}

// Integration test - only runs if AMQP broker is available
func TestAMQPBroker_Integration(t *testing.T) {
	// Skip if not running integration tests
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	config := Config{
		URL:      "amqp://guest:guest@localhost:5672/",
		Database: 0,
		Username: "guest",
		Password: "guest",
	}

	broker := NewAMQPBroker(config)
	ctx := context.Background()

	// Try to connect
	err := broker.Connect(ctx)
	if err != nil {
		t.Skipf("Skipping integration test - could not connect to AMQP broker: %v", err)
	}
	defer broker.Close()

	// Test health check
	err = broker.Health(ctx)
	if err != nil {
		t.Errorf("Health check failed: %v", err)
	}

	// Test ping (should timeout since no workers are running)
	responses, err := broker.Ping(ctx, time.Millisecond*100, nil)
	if err != nil {
		t.Errorf("Ping failed: %v", err)
	}

	// Should get empty response since no workers are running
	if len(responses) != 0 {
		t.Errorf("Expected no responses, got %d", len(responses))
	}
}

func TestAMQPBroker_Ping_WithDestination(t *testing.T) {
	// Skip if not running integration tests
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	config := Config{
		URL:      "amqp://guest:guest@localhost:5672/",
		Database: 0,
		Username: "guest",
		Password: "guest",
	}

	broker := NewAMQPBroker(config)
	ctx := context.Background()

	// Try to connect
	err := broker.Connect(ctx)
	if err != nil {
		t.Skipf("Skipping integration test - could not connect to AMQP broker: %v", err)
	}
	defer broker.Close()

	// Test ping with specific destination
	destinations := []string{"worker1@localhost", "worker2@localhost"}
	responses, err := broker.Ping(ctx, time.Millisecond*100, destinations)
	if err != nil {
		t.Errorf("Ping with destinations failed: %v", err)
	}

	// Should get empty response since no workers are running
	if len(responses) != 0 {
		t.Errorf("Expected no responses, got %d", len(responses))
	}
}
