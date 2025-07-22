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
func (r *RedisBroker) Ping(ctx context.Context, timeout time.Duration, destinations []string) (map[string]PingResponse, error) {
	if r.client == nil {
		return nil, fmt.Errorf("Redis client not initialized")
	}

	// Create reply queue with simple UUID format
	replyTo := r.handler.CreateReplyQueue()

	// Create ping message in enveloped format (base64 + envelope wrapper)
	pingData, err := r.handler.CreatePingMessage(replyTo, destinations, protocol.MessageFormatEnveloped)
	if err != nil {
		return nil, fmt.Errorf("failed to create ping message: %w", err)
	}

	// Use the correct reply queue format: UUID.reply.celery.pidbox
	baseReplyQueue := replyTo + ".reply.celery.pidbox"

	// Python celery listens on multiple queue variants with different priorities
	replyQueues := []string{
		baseReplyQueue,
		baseReplyQueue + string([]byte{0x06, 0x16}) + "3", // priority 3
		baseReplyQueue + string([]byte{0x06, 0x16}) + "6", // priority 6
		baseReplyQueue + string([]byte{0x06, 0x16}) + "9", // priority 9
	}

	// Publish the message to the broadcast channel
	err = r.client.Publish(ctx, "/0.celery.pidbox", string(pingData)).Err()
	if err != nil {
		return nil, fmt.Errorf("failed to publish ping message: %w", err)
	}

	// Register reply queue binding like Python celery does
	bindingKey := replyTo + string([]byte{0x06, 0x16, 0x06, 0x16}) + baseReplyQueue
	err = r.client.SAdd(ctx, "_kombu.binding.reply.celery.pidbox", bindingKey).Err()
	if err != nil {
		return nil, fmt.Errorf("failed to register reply queue binding: %w", err)
	}

	// Wait for responses using blocking pop with timeout
	responses := make(map[string]PingResponse)
	deadline := time.Now().Add(timeout)

	// Give workers a moment to see the reply queue binding
	time.Sleep(50 * time.Millisecond)

	// Track last response time for early exit logic
	lastResponseTime := time.Now()
	responseGapTimeout := 200 * time.Millisecond // Exit if no responses for 200ms

	for time.Now().Before(deadline) {
		// Calculate remaining time
		remaining := time.Until(deadline)
		if remaining <= 0 {
			break
		}

		// Use shorter BRPOP timeout for more responsive behavior
		brpopTimeout := 100 * time.Millisecond
		if remaining < brpopTimeout {
			brpopTimeout = remaining
		}

		// Early exit if we haven't received responses for a while and have some responses
		if len(responses) > 0 && time.Since(lastResponseTime) > responseGapTimeout {
			break
		}

		// BRPOP on all queue variants
		result, err := r.client.BRPop(ctx, brpopTimeout, replyQueues...).Result()
		if err != nil {
			if err == redis.Nil {
				// Timeout - continue checking
				continue
			}
			// Other error - break
			break
		}

		if len(result) < 2 {
			continue
		}

		// Process the response
		response, err := r.handler.ParseWorkerResponse([]byte(result[1]))
		if err != nil {
			continue
		}

		if r.handler.ValidateResponse(response) {
			workerName := r.handler.ExtractWorkerName(response)
			if workerName != "" {
				// Add response (map will naturally deduplicate)
				responses[workerName] = PingResponse{
					WorkerName: workerName,
					Status:     "pong",
					Timestamp:  time.Now().Unix(),
				}
				// Update last response time for early exit logic
				lastResponseTime = time.Now()
			}
		}
	}

	// Clean up reply queue binding and queues
	r.client.SRem(ctx, "_kombu.binding.reply.celery.pidbox", bindingKey)
	r.client.Del(ctx, replyQueues...)

	return responses, nil
}
