package broker

import (
	"context"
	"fmt"
	"time"

	"fast-celery-ping/internal/protocol"

	"github.com/redis/go-redis/v9"
)

// RedisBroker implements the Broker interface for Redis
type RedisBroker struct {
	client  *redis.Client
	config  Config
	handler *protocol.Handler
}

// NewRedisBroker creates a new Redis broker instance
func NewRedisBroker(config Config) *RedisBroker {
	return &RedisBroker{
		config:  config,
		handler: protocol.NewHandler(),
	}
}

// Connect establishes connection to Redis
func (r *RedisBroker) Connect(ctx context.Context) error {
	opts, err := redis.ParseURL(r.config.URL)
	if err != nil {
		return fmt.Errorf("failed to parse Redis URL: %w", err)
	}

	if r.config.Database != 0 {
		opts.DB = r.config.Database
	}
	if r.config.Username != "" {
		opts.Username = r.config.Username
	}
	if r.config.Password != "" {
		opts.Password = r.config.Password
	}

	r.client = redis.NewClient(opts)

	// Test connection
	return r.Health(ctx)
}

// Close closes the Redis connection
func (r *RedisBroker) Close() error {
	if r.client != nil {
		return r.client.Close()
	}
	return nil
}

// Health checks Redis connectivity
func (r *RedisBroker) Health(ctx context.Context) error {
	if r.client == nil {
		return fmt.Errorf("Redis client not initialized")
	}

	return r.client.Ping(ctx).Err()
}

// Ping implements the Celery ping functionality for Redis
func (r *RedisBroker) Ping(ctx context.Context, timeout time.Duration) (map[string]PingResponse, error) {
	if r.client == nil {
		return nil, fmt.Errorf("Redis client not initialized")
	}

	// Create reply queue
	replyTo := r.handler.CreateReplyQueue()

	// Create ping message
	pingData, err := r.handler.CreatePingMessage(replyTo)
	if err != nil {
		return nil, fmt.Errorf("failed to create ping message: %w", err)
	}

	// Setup reply queue and send broadcast message
	pipe := r.client.Pipeline()
	pipe.Del(ctx, replyTo)
	pipe.Expire(ctx, replyTo, timeout+time.Second)
	pipe.LPush(ctx, r.handler.GetBroadcastQueue(), string(pingData))

	_, err = pipe.Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to send ping message: %w", err)
	}

	// Wait for responses
	responses := make(map[string]PingResponse)
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		// Check for responses in the reply queue
		result, err := r.client.BRPop(ctx, time.Millisecond*100, replyTo).Result()
		if err != nil {
			if err == redis.Nil {
				// No response yet, continue waiting
				continue
			}
			return nil, fmt.Errorf("failed to receive response: %w", err)
		}

		if len(result) < 2 {
			continue
		}

		// Parse response using protocol handler
		response, err := r.handler.ParseWorkerResponse([]byte(result[1]))
		if err != nil {
			continue // Skip malformed responses
		}

		// Validate and extract worker information
		if r.handler.ValidateResponse(response) {
			workerName := r.handler.ExtractWorkerName(response)
			if workerName != "" {
				responses[workerName] = PingResponse{
					WorkerName: workerName,
					Status:     "pong",
					Timestamp:  time.Now().Unix(),
				}
			}
		}
	}

	// Clean up reply queue
	r.client.Del(ctx, replyTo)

	return responses, nil
}
